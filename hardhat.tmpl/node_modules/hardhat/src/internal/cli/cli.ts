import picocolors from "picocolors";
import debug from "debug";
import "source-map-support/register";

import {
  TASK_COMPILE,
  TASK_HELP,
  TASK_TEST,
} from "../../builtin-tasks/task-names";
import { HardhatConfig, TaskArguments } from "../../types";
import { HARDHAT_NAME } from "../constants";
import { HardhatContext } from "../context";
import {
  getConfiguredCompilers,
  loadConfigAndTasks,
} from "../core/config/config-loading";
import {
  assertHardhatInvariant,
  HardhatError,
  HardhatPluginError,
} from "../core/errors";
import { ERRORS, getErrorCode } from "../core/errors-list";
import { isHardhatInstalledLocallyOrLinked } from "../core/execution-mode";
import { getEnvHardhatArguments } from "../core/params/env-variables";
import { HARDHAT_PARAM_DEFINITIONS } from "../core/params/hardhat-params";
import {
  getUserConfigPath,
  isCwdInsideProject,
} from "../core/project-structure";
import { Environment } from "../core/runtime-environment";
import { loadTsNode, willRunWithTypescript } from "../core/typescript-support";
import { Reporter } from "../sentry/reporter";
import { isRunningOnCiServer } from "../util/ci-detection";
import {
  hasConsentedTelemetry,
  hasPromptedForHHVSCode,
  writePromptedForHHVSCode,
} from "../util/global-dir";
import { getPackageJson } from "../util/packageInfo";
import { saveFlamegraph } from "../core/flamegraph";
import { Analytics, requestTelemetryConsent } from "./analytics";
import { ArgumentsParser } from "./ArgumentsParser";
import { enableEmoji } from "./emoji";
import { createProject, showSoliditySurveyMessage } from "./project-creation";
import { confirmHHVSCodeInstallation } from "./prompt";
import {
  InstallationState,
  installHardhatVSCode,
  isHardhatVSCodeInstalled,
} from "./hardhat-vscode-installation";
import { handleVars } from "./vars";
import { BannerManager } from "./banner-manager";

const log = debug("hardhat:core:cli");

const ANALYTICS_SLOW_TASK_THRESHOLD = 300;
const SHOULD_SHOW_STACK_TRACES_BY_DEFAULT = isRunningOnCiServer();

async function printVersionMessage() {
  const packageJson = await getPackageJson();
  console.log(packageJson.version);
}

async function suggestInstallingHardhatVscode() {
  const alreadyPrompted = hasPromptedForHHVSCode();
  if (alreadyPrompted) {
    return;
  }

  const isInstalled = isHardhatVSCodeInstalled();
  writePromptedForHHVSCode();

  if (isInstalled !== InstallationState.EXTENSION_NOT_INSTALLED) {
    return;
  }

  const installationConsent = await confirmHHVSCodeInstallation();

  if (installationConsent === true) {
    console.log("Installing Hardhat for Visual Studio Code...");
    const installed = installHardhatVSCode();

    if (installed) {
      console.log("Hardhat for Visual Studio Code was successfully installed");
    } else {
      console.log(
        "Hardhat for Visual Studio Code couldn't be installed. To learn more about it, go to https://hardhat.org/hardhat-vscode"
      );
    }
  } else {
    console.log(
      "To learn more about Hardhat for Visual Studio Code, go to https://hardhat.org/hardhat-vscode"
    );
  }
}

function showViaIRWarning(resolvedConfig: HardhatConfig) {
  const configuredCompilers = getConfiguredCompilers(resolvedConfig.solidity);
  const viaIREnabled = configuredCompilers.some(
    (compiler) => compiler.settings?.viaIR === true
  );

  if (viaIREnabled) {
    console.warn();
    console.warn(
      picocolors.yellow(
        `Your solidity settings have viaIR enabled, which is not fully supported yet. You can still use Hardhat, but some features, like stack traces, might not work correctly.

Learn more at https://hardhat.org/solc-viair`
      )
    );
  }
}

async function main() {
  // We first accept this argument anywhere, so we know if the user wants
  // stack traces before really parsing the arguments.
  let showStackTraces =
    process.argv.includes("--show-stack-traces") ||
    SHOULD_SHOW_STACK_TRACES_BY_DEFAULT;

  try {
    const envVariableArguments = getEnvHardhatArguments(
      HARDHAT_PARAM_DEFINITIONS,
      process.env
    );

    const argumentsParser = new ArgumentsParser();

    const { hardhatArguments, scopeOrTaskName, allUnparsedCLAs } =
      argumentsParser.parseHardhatArguments(
        HARDHAT_PARAM_DEFINITIONS,
        envVariableArguments,
        process.argv.slice(2)
      );

    if (hardhatArguments.verbose) {
      Reporter.setVerbose(true);
      debug.enable("hardhat*");
    }

    if (hardhatArguments.emoji) {
      enableEmoji();
    }

    showStackTraces = hardhatArguments.showStackTraces;

    // --version is a special case
    if (hardhatArguments.version) {
      await printVersionMessage();
      return;
    }

    // ATTENTION! DEPRECATED CODE!
    // The command `npx hardhat`, when used to create a new Hardhat project, will be removed with Hardhat V3.
    // It will become `npx hardhat init`.
    // The code marked with the tag #INIT-DEP can be deleted after HarhatV3 is out.

    // Create a new Hardhat project
    if (scopeOrTaskName === "init") {
      return await createNewProject();
    }
    // #INIT-DEP - START OF DEPRECATED CODE
    else {
      if (
        scopeOrTaskName === undefined &&
        hardhatArguments.config === undefined &&
        !isCwdInsideProject()
      ) {
        await createNewProject();

        // Warning for Hardhat V3 deprecation
        console.warn(
          picocolors.yellow(picocolors.bold("\n\nDEPRECATION WARNING\n\n")),
          picocolors.yellow(
            `Initializing a project with ${picocolors.white(
              picocolors.italic("npx hardhat")
            )} is deprecated and will be removed in the future.\n`
          ),
          picocolors.yellow(
            `Please use ${picocolors.white(
              picocolors.italic("npx hardhat init")
            )} instead.\n\n`
          )
        );

        return;
      }
    }
    // #INIT-DEP - END OF DEPRECATED CODE

    // Tasks are only allowed inside a Hardhat project (except the init task)
    if (hardhatArguments.config === undefined && !isCwdInsideProject()) {
      throw new HardhatError(ERRORS.GENERAL.NOT_INSIDE_PROJECT);
    }

    if (
      process.env.HARDHAT_EXPERIMENTAL_ALLOW_NON_LOCAL_INSTALLATION !==
        "true" &&
      !isHardhatInstalledLocallyOrLinked()
    ) {
      throw new HardhatError(ERRORS.GENERAL.NON_LOCAL_INSTALLATION);
    }

    if (willRunWithTypescript(hardhatArguments.config)) {
      loadTsNode(hardhatArguments.tsconfig, hardhatArguments.typecheck);
    } else {
      if (hardhatArguments.typecheck === true) {
        throw new HardhatError(
          ERRORS.ARGUMENTS.TYPECHECK_USED_IN_JAVASCRIPT_PROJECT
        );
      }
    }

    const ctx = HardhatContext.createHardhatContext();

    if (scopeOrTaskName === "vars" && allUnparsedCLAs.length > 1) {
      process.exit(await handleVars(allUnparsedCLAs, hardhatArguments.config));
    }

    const { resolvedConfig, userConfig } = loadConfigAndTasks(
      hardhatArguments,
      {
        showEmptyConfigWarning: true,
        showSolidityConfigWarnings: scopeOrTaskName === TASK_COMPILE,
      }
    );

    const envExtenders = ctx.environmentExtenders;
    const providerExtenders = ctx.providerExtenders;
    const taskDefinitions = ctx.tasksDSL.getTaskDefinitions();
    const scopesDefinitions = ctx.tasksDSL.getScopesDefinitions();

    const env = new Environment(
      resolvedConfig,
      hardhatArguments,
      taskDefinitions,
      scopesDefinitions,
      envExtenders,
      userConfig,
      providerExtenders
    );

    ctx.setHardhatRuntimeEnvironment(env);

    // eslint-disable-next-line prefer-const
    let { scopeName, taskName, unparsedCLAs } =
      argumentsParser.parseScopeAndTaskNames(
        allUnparsedCLAs,
        taskDefinitions,
        scopesDefinitions
      );

    let telemetryConsent: boolean | undefined = hasConsentedTelemetry();

    const isHelpCommand = hardhatArguments.help || taskName === TASK_HELP;
    if (
      telemetryConsent === undefined &&
      !isHelpCommand &&
      !isRunningOnCiServer() &&
      process.stdout.isTTY === true &&
      process.env.HARDHAT_DISABLE_TELEMETRY_PROMPT !== "true"
    ) {
      telemetryConsent = await requestTelemetryConsent();
    }

    const analytics = await Analytics.getInstance(telemetryConsent);

    Reporter.setConfigPath(resolvedConfig.paths.configFile);
    if (telemetryConsent === true) {
      Reporter.setEnabled(true);
    }

    const [abortAnalytics, hitPromise] = await analytics.sendTaskHit(
      scopeName,
      taskName
    );

    let taskArguments: TaskArguments;

    // --help is an also special case
    if (hardhatArguments.help && taskName !== TASK_HELP) {
      // we "move" the task and scope names to the task arguments,
      // and run the help task
      if (scopeName !== undefined) {
        taskArguments = { scopeOrTask: scopeName, task: taskName };
      } else {
        taskArguments = { scopeOrTask: taskName };
      }
      taskName = TASK_HELP;
      scopeName = undefined;
    } else {
      const taskDefinition = ctx.tasksDSL.getTaskDefinition(
        scopeName,
        taskName
      );

      if (taskDefinition === undefined) {
        if (scopeName !== undefined) {
          throw new HardhatError(ERRORS.ARGUMENTS.UNRECOGNIZED_SCOPED_TASK, {
            scope: scopeName,
            task: taskName,
          });
        }
        throw new HardhatError(ERRORS.ARGUMENTS.UNRECOGNIZED_TASK, {
          task: taskName,
        });
      }

      if (taskDefinition.isSubtask) {
        throw new HardhatError(ERRORS.ARGUMENTS.RUNNING_SUBTASK_FROM_CLI, {
          name: taskDefinition.name,
        });
      }

      taskArguments = argumentsParser.parseTaskArguments(
        taskDefinition,
        unparsedCLAs
      );
    }

    try {
      const timestampBeforeRun = new Date().getTime();

      await env.run({ scope: scopeName, task: taskName }, taskArguments);

      const timestampAfterRun = new Date().getTime();

      if (
        timestampAfterRun - timestampBeforeRun >
          ANALYTICS_SLOW_TASK_THRESHOLD &&
        taskName !== TASK_COMPILE
      ) {
        await hitPromise;
      } else {
        abortAnalytics();
      }
    } finally {
      if (hardhatArguments.flamegraph === true) {
        assertHardhatInvariant(
          env.entryTaskProfile !== undefined,
          "--flamegraph was set but entryTaskProfile is not defined"
        );

        const flamegraphPath = saveFlamegraph(env.entryTaskProfile);
        console.log("Created flamegraph file", flamegraphPath);
      }
    }

    // VSCode extension prompt for installation
    if (
      taskName === TASK_TEST &&
      !isRunningOnCiServer() &&
      process.stdout.isTTY === true
    ) {
      await suggestInstallingHardhatVscode();

      // we show the solidity survey message if the tests failed and only
      // 1/3 of the time
      if (
        process.exitCode !== 0 &&
        Math.random() < 0.3333 &&
        process.env.HARDHAT_HIDE_SOLIDITY_SURVEY_MESSAGE !== "true"
      ) {
        showSoliditySurveyMessage();
      }

      // we show the viaIR warning only if the tests failed
      if (process.exitCode !== 0) {
        showViaIRWarning(resolvedConfig);
      }

      // we notify of new versions only if the tests failed
      if (process.exitCode !== 0) {
        try {
          const { showNewVersionNotification } = await import(
            "./version-notifier"
          );
          await showNewVersionNotification();
        } catch {
          // ignore possible version notifier errors
        }
      }
    }

    if (!isRunningOnCiServer() && process.stdout.isTTY === true) {
      const hardhat3BannerManager = await BannerManager.getInstance();
      await hardhat3BannerManager.showBanner(200);
    }

    log(`Killing Hardhat after successfully running task ${taskName}`);
  } catch (error) {
    let isHardhatError = false;

    if (HardhatError.isHardhatError(error)) {
      isHardhatError = true;
      console.error(
        picocolors.red(picocolors.bold("Error")),
        error.message.replace(/^\w+:/, (t) =>
          picocolors.red(picocolors.bold(t))
        )
      );
    } else if (HardhatPluginError.isHardhatPluginError(error)) {
      isHardhatError = true;
      console.error(
        picocolors.red(picocolors.bold(`Error in plugin ${error.pluginName}:`)),
        error.message
      );
    } else if (error instanceof Error) {
      console.error(picocolors.red("An unexpected error occurred:"));
      showStackTraces = true;
    } else {
      console.error(picocolors.red("An unexpected error occurred."));
      showStackTraces = true;
    }

    console.log("");

    try {
      Reporter.reportError(error as Error);
    } catch (e) {
      log("Couldn't report error to sentry: %O", e);
    }

    if (showStackTraces || SHOULD_SHOW_STACK_TRACES_BY_DEFAULT) {
      console.error(error);
    } else {
      if (!isHardhatError) {
        console.error(
          `If you think this is a bug in Hardhat, please report it here: https://hardhat.org/report-bug`
        );
      }

      if (HardhatError.isHardhatError(error)) {
        const link = `https://hardhat.org/${getErrorCode(
          error.errorDescriptor
        )}`;

        console.error(
          `For more info go to ${link} or run ${HARDHAT_NAME} with --show-stack-traces`
        );
      } else {
        console.error(
          `For more info run ${HARDHAT_NAME} with --show-stack-traces`
        );
      }
    }

    await Reporter.close(1000);
    process.exit(1);
  }
}

async function createNewProject() {
  if (isCwdInsideProject()) {
    throw new HardhatError(ERRORS.GENERAL.HARDHAT_PROJECT_ALREADY_CREATED, {
      hardhatProjectRootPath: getUserConfigPath(),
    });
  }

  if (
    process.stdout.isTTY === true ||
    process.env.HARDHAT_CREATE_JAVASCRIPT_PROJECT_WITH_DEFAULTS !== undefined ||
    process.env.HARDHAT_CREATE_TYPESCRIPT_PROJECT_WITH_DEFAULTS !== undefined ||
    process.env.HARDHAT_CREATE_TYPESCRIPT_VIEM_PROJECT_WITH_DEFAULTS !==
      undefined
  ) {
    await createProject();
    return;
  }

  // Many terminal emulators in windows fail to run the createProject()
  // workflow, and don't present themselves as TTYs. If we are in this
  // situation we throw a special error instructing the user to use WSL or
  // powershell to initialize the project.
  if (process.platform === "win32") {
    throw new HardhatError(ERRORS.GENERAL.NOT_INSIDE_PROJECT_ON_WINDOWS);
  }

  throw new HardhatError(ERRORS.GENERAL.NOT_IN_INTERACTIVE_SHELL);
}

main()
  .then(() => process.exit(process.exitCode))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
