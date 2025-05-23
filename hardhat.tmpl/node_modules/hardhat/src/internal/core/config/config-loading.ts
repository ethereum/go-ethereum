import type StackTraceParserT from "stacktrace-parser";

import picocolors from "picocolors";
import debug from "debug";
import fsExtra from "fs-extra";
import path from "path";
import semver from "semver";

import {
  HardhatArguments,
  HardhatConfig,
  HardhatUserConfig,
  SolcConfig,
} from "../../../types";
import { HardhatContext } from "../../context";
import { findClosestPackageJson } from "../../util/packageInfo";
import { HardhatError } from "../errors";
import { ERRORS } from "../errors-list";
import { getUserConfigPath } from "../project-structure";

import { SUPPORTED_SOLIDITY_VERSION_RANGE } from "../../hardhat-network/stack-traces/constants";
import { resolveConfig } from "./config-resolution";
import { DEFAULT_SOLC_VERSION } from "./default-config";

const log = debug("hardhat:core:config");

export function importCsjOrEsModule(filePath: string): any {
  try {
    const imported = require(filePath);
    return imported.default !== undefined ? imported.default : imported;
  } catch (e: any) {
    // An ESM project that has a Hardhat config with a .js extension will fail to be loaded,
    // because Hardhat configs can only be CJS but a .js extension will be interpreted as ESM.
    // The kind of error we get in these cases depends on the Node.js version.
    const node20Heuristic = e.code === "ERR_REQUIRE_ESM";
    const node22Heuristic =
      e.message === "module is not defined" ||
      e.message === "require is not defined";
    if (node20Heuristic || node22Heuristic) {
      throw new HardhatError(
        ERRORS.GENERAL.ESM_PROJECT_WITHOUT_CJS_CONFIG,
        {},
        e
      );
    }

    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw e;
  }
}

export function resolveConfigPath(configPath: string | undefined) {
  if (configPath === undefined) {
    configPath = getUserConfigPath();
  } else {
    if (!path.isAbsolute(configPath)) {
      configPath = path.join(process.cwd(), configPath);
      configPath = path.normalize(configPath);
    }
  }
  return configPath;
}

export function loadConfigAndTasks(
  hardhatArguments?: Partial<HardhatArguments>,
  {
    showEmptyConfigWarning = false,
    showSolidityConfigWarnings = false,
  }: {
    showEmptyConfigWarning?: boolean;
    showSolidityConfigWarnings?: boolean;
  } = {
    showEmptyConfigWarning: false,
    showSolidityConfigWarnings: false,
  }
): { resolvedConfig: HardhatConfig; userConfig: HardhatUserConfig } {
  const { validateConfig, validateResolvedConfig } =
    require("./config-validation") as typeof import("./config-validation");
  let configPath =
    hardhatArguments !== undefined ? hardhatArguments.config : undefined;

  configPath = resolveConfigPath(configPath);
  log(`Loading Hardhat config from ${configPath}`);
  // Before loading the builtin tasks, the default and user's config we expose
  // the config env in the global object.
  const configEnv = require("./config-env");

  const globalAsAny: any = global;

  Object.entries(configEnv).forEach(
    ([key, value]) => (globalAsAny[key] = value)
  );

  const ctx = HardhatContext.getHardhatContext();

  ctx.setConfigLoadingAsStarted();

  let userConfig;

  try {
    require("../tasks/builtin-tasks");
    userConfig = importCsjOrEsModule(configPath);
  } catch (e) {
    analyzeModuleNotFoundError(e, configPath);

    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw e;
  } finally {
    ctx.setConfigLoadingAsFinished();
  }

  if (showEmptyConfigWarning) {
    checkEmptyConfig(userConfig, { showSolidityConfigWarnings });
  }

  validateConfig(userConfig);

  if (showSolidityConfigWarnings) {
    checkMissingSolidityConfig(userConfig);
  }

  // To avoid bad practices we remove the previously exported stuff
  Object.keys(configEnv).forEach((key) => (globalAsAny[key] = undefined));

  const frozenUserConfig = deepFreezeUserConfig(userConfig);

  const resolved = resolveConfig(configPath, userConfig);

  for (const extender of HardhatContext.getHardhatContext().configExtenders) {
    extender(resolved, frozenUserConfig);
  }

  validateResolvedConfig(resolved);

  if (showSolidityConfigWarnings) {
    checkUnsupportedSolidityConfig(resolved);
    checkUnsupportedRemappings(resolved);
  }

  return { resolvedConfig: resolved, userConfig: frozenUserConfig };
}

function deepFreezeUserConfig(
  config: any,
  propertyPath: Array<string | number | symbol> = []
) {
  if (typeof config !== "object" || config === null) {
    return config;
  }

  return new Proxy(config, {
    get(target: any, property: string | number | symbol, receiver: any): any {
      return deepFreezeUserConfig(Reflect.get(target, property, receiver), [
        ...propertyPath,
        property,
      ]);
    },

    set(
      target: any,
      property: string | number | symbol,
      _value: any,
      _receiver: any
    ): boolean {
      throw new HardhatError(ERRORS.GENERAL.USER_CONFIG_MODIFIED, {
        path: [...propertyPath, property]
          .map((pathPart) => pathPart.toString())
          .join("."),
      });
    },
  });
}

/**
 * Receives an Error and checks if it's a MODULE_NOT_FOUND and the reason that
 * caused it.
 *
 * If it can infer the reason, it throws an appropriate error. Otherwise it does
 * nothing.
 */
export function analyzeModuleNotFoundError(error: any, configPath: string) {
  const stackTraceParser =
    require("stacktrace-parser") as typeof StackTraceParserT;

  if (error.code !== "MODULE_NOT_FOUND") {
    return;
  }
  const stackTrace = stackTraceParser.parse(error.stack);
  const throwingFile = stackTrace
    .filter((x) => x.file !== null)
    .map((x) => x.file!)
    // ignore frames related to source map support
    .filter((x) => !x.includes(path.join("@cspotcode", "source-map-support")))
    .find((x) => path.isAbsolute(x));

  if (throwingFile === null || throwingFile === undefined) {
    return;
  }

  // if the error comes from the config file, we ignore it because we know it's
  // a direct import that's missing
  if (throwingFile === configPath) {
    return;
  }

  const packageJsonPath = findClosestPackageJson(throwingFile);

  if (packageJsonPath === undefined) {
    return;
  }

  const packageJson = fsExtra.readJsonSync(packageJsonPath);
  const peerDependencies: { [name: string]: string } =
    packageJson.peerDependencies ?? {};

  if (peerDependencies["@nomiclabs/buidler"] !== undefined) {
    throw new HardhatError(ERRORS.PLUGINS.BUIDLER_PLUGIN, {
      plugin: packageJson.name,
    });
  }

  // if the problem doesn't come from a hardhat plugin, we ignore it
  if (peerDependencies.hardhat === undefined) {
    return;
  }

  const missingPeerDependencies: { [name: string]: string } = {};
  for (const [peerDependency, version] of Object.entries(peerDependencies)) {
    const peerDependencyPackageJson = readPackageJson(
      peerDependency,
      configPath
    );
    if (peerDependencyPackageJson === undefined) {
      missingPeerDependencies[peerDependency] = version;
    }
  }

  const missingPeerDependenciesNames = Object.keys(missingPeerDependencies);
  if (missingPeerDependenciesNames.length > 0) {
    throw new HardhatError(ERRORS.PLUGINS.MISSING_DEPENDENCIES, {
      plugin: packageJson.name,
      missingDependencies: missingPeerDependenciesNames.join(", "),
      missingDependenciesVersions: Object.entries(missingPeerDependencies)
        .map(([name, version]) => `"${name}@${version}"`)
        .join(" "),
    });
  }
}

interface PackageJson {
  name: string;
  version: string;
  peerDependencies?: {
    [name: string]: string;
  };
}

function readPackageJson(
  packageName: string,
  configPath: string
): PackageJson | undefined {
  const resolve = require("resolve") as typeof import("resolve");

  try {
    const packageJsonPath = resolve.sync(
      path.join(packageName, "package.json"),
      {
        basedir: path.dirname(configPath),
      }
    );

    return require(packageJsonPath);
  } catch {
    return undefined;
  }
}

function checkEmptyConfig(
  userConfig: any,
  { showSolidityConfigWarnings }: { showSolidityConfigWarnings: boolean }
) {
  if (userConfig === undefined || Object.keys(userConfig).length === 0) {
    let warning = `Hardhat config is returning an empty config object, check the export from the config file if this is unexpected.\n`;

    // This 'learn more' section is also printed by the solidity config warning,
    // so we need to check to avoid printing it twice
    if (!showSolidityConfigWarnings) {
      warning += `\nLearn more about configuring Hardhat at https://hardhat.org/config\n`;
    }

    console.warn(picocolors.yellow(warning));
  }
}

function checkMissingSolidityConfig(userConfig: any) {
  if (userConfig.solidity === undefined) {
    console.warn(
      picocolors.yellow(
        `Solidity compiler is not configured. Version ${DEFAULT_SOLC_VERSION} will be used by default. Add a 'solidity' entry to your configuration to suppress this warning.

Learn more about compiler configuration at https://hardhat.org/config
`
      )
    );
  }
}

function checkUnsupportedSolidityConfig(resolvedConfig: HardhatConfig) {
  const configuredCompilers = getConfiguredCompilers(resolvedConfig.solidity);
  const solcVersions = configuredCompilers.map((x) => x.version);

  const unsupportedVersions: string[] = [];
  for (const solcVersion of solcVersions) {
    if (
      !semver.satisfies(solcVersion, SUPPORTED_SOLIDITY_VERSION_RANGE) &&
      !unsupportedVersions.includes(solcVersion)
    ) {
      unsupportedVersions.push(solcVersion);
    }
  }

  if (unsupportedVersions.length > 0) {
    console.warn(
      picocolors.yellow(
        `Solidity ${unsupportedVersions.join(", ")} ${
          unsupportedVersions.length === 1 ? "is" : "are"
        } not fully supported yet. You can still use Hardhat, but some features, like stack traces, might not work correctly.

Learn more at https://hardhat.org/hardhat-runner/docs/reference/solidity-support
`
      )
    );
  }
}

function checkUnsupportedRemappings({ solidity }: HardhatConfig) {
  const solcConfigs = [
    ...solidity.compilers,
    ...Object.values(solidity.overrides),
  ];
  const remappings = solcConfigs.filter(
    ({ settings }) => settings.remappings !== undefined
  );

  if (remappings.length > 0) {
    console.warn(
      picocolors.yellow(
        `Solidity remappings are not currently supported; you may experience unexpected compilation results. Remove any 'remappings' fields from your configuration to suppress this warning.

Learn more about compiler configuration at https://hardhat.org/config
`
      )
    );
  }
}

export function getConfiguredCompilers(
  solidityConfig: HardhatConfig["solidity"]
): SolcConfig[] {
  const compilerVersions = solidityConfig.compilers;
  const overrideVersions = Object.values(solidityConfig.overrides);
  return [...compilerVersions, ...overrideVersions];
}
