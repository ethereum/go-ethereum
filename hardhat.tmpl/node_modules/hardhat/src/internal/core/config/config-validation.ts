import type { HardhatConfig as HardhatConfigT } from "../../../types";
import type {
  Context,
  ValidationError,
  getFunctionName as getFunctionNameT,
} from "io-ts/lib";
import type { Reporter } from "io-ts/lib/Reporter";

import * as t from "io-ts";

import {
  HARDHAT_MEMPOOL_SUPPORTED_ORDERS,
  HARDHAT_NETWORK_NAME,
  HARDHAT_NETWORK_SUPPORTED_HARDFORKS,
} from "../../constants";
import { optional } from "../../util/io-ts";
import { fromEntries } from "../../util/lang";
import { HardhatError } from "../errors";
import { ERRORS } from "../errors-list";
import { hardforkGte, HardforkName } from "../../util/hardforks";
import { HardhatNetworkChainUserConfig } from "../../../types/config";
import { defaultHardhatNetworkParams } from "./default-config";

function stringify(v: any): string {
  if (typeof v === "function") {
    const { getFunctionName } = require("io-ts/lib") as {
      getFunctionName: typeof getFunctionNameT;
    };
    return getFunctionName(v);
  }
  if (typeof v === "number" && !isFinite(v)) {
    if (isNaN(v)) {
      return "NaN";
    }
    return v > 0 ? "Infinity" : "-Infinity";
  }
  return JSON.stringify(v);
}

function getContextPath(context: Context): string {
  const keysPath = context
    .slice(1)
    .map((c) => c.key)
    .join(".");

  return `${context[0].type.name}.${keysPath}`;
}

function getMessage(e: ValidationError): string {
  const lastContext = e.context[e.context.length - 1];

  return e.message !== undefined
    ? e.message
    : getErrorMessage(
        getContextPath(e.context),
        e.value,
        lastContext.type.name
      );
}

function getErrorMessage(path: string, value: any, expectedType: string) {
  return `Invalid value ${stringify(
    value
  )} for ${path} - Expected a value of type ${expectedType}.`;
}

function getPrivateKeyError(index: number, network: string, message: string) {
  return `Invalid account: #${index} for network: ${network} - ${message}`;
}

function validatePrivateKey(
  privateKey: unknown,
  index: number,
  network: string,
  errors: string[]
) {
  if (typeof privateKey !== "string") {
    errors.push(
      getPrivateKeyError(
        index,
        network,
        `Expected string, received ${typeof privateKey}`
      )
    );
  } else {
    // private key validation
    const pkWithPrefix = /^0x/.test(privateKey)
      ? privateKey
      : `0x${privateKey}`;

    // 32 bytes = 64 characters + 2 char prefix = 66
    if (pkWithPrefix.length < 66) {
      errors.push(
        getPrivateKeyError(
          index,
          network,
          "private key too short, expected 32 bytes"
        )
      );
    } else if (pkWithPrefix.length > 66) {
      errors.push(
        getPrivateKeyError(
          index,
          network,
          "private key too long, expected 32 bytes"
        )
      );
    } else if (hexString.decode(pkWithPrefix).isLeft()) {
      errors.push(
        getPrivateKeyError(
          index,
          network,
          "invalid hex character(s) found in string"
        )
      );
    }
  }
}

export function failure(es: ValidationError[]): string[] {
  return es.map(getMessage);
}

export function success(): string[] {
  return [];
}

export const DotPathReporter: Reporter<string[]> = {
  report: (validation) => validation.fold(failure, success),
};

const HEX_STRING_REGEX = /^(0x)?([0-9a-f]{2})+$/gi;
const DEC_STRING_REGEX = /^(0|[1-9][0-9]*)$/g;

function isHexString(v: unknown): v is string {
  if (typeof v !== "string") {
    return false;
  }

  return v.trim().match(HEX_STRING_REGEX) !== null;
}

function isDecimalString(v: unknown): v is string {
  if (typeof v !== "string") {
    return false;
  }

  return v.match(DEC_STRING_REGEX) !== null;
}

export const hexString = new t.Type<string>(
  "hex string",
  isHexString,
  (u, c) => (isHexString(u) ? t.success(u) : t.failure(u, c)),
  t.identity
);

function isAddress(v: unknown): v is string {
  if (typeof v !== "string") {
    return false;
  }

  const trimmed = v.trim();

  return (
    trimmed.match(HEX_STRING_REGEX) !== null &&
    trimmed.startsWith("0x") &&
    trimmed.length === 42
  );
}

export const address = new t.Type<string>(
  "address",
  isAddress,
  (u, c) => (isAddress(u) ? t.success(u) : t.failure(u, c)),
  t.identity
);

export const decimalString = new t.Type<string>(
  "decimal string",
  isDecimalString,
  (u, c) => (isDecimalString(u) ? t.success(u) : t.failure(u, c)),
  t.identity
);
// TODO: These types have outdated name. They should match the UserConfig types.
// IMPORTANT: This t.types MUST be kept in sync with the actual types.

const HardhatNetworkAccount = t.type({
  privateKey: hexString,
  balance: decimalString,
});

const commonHDAccountsFields = {
  initialIndex: optional(t.number),
  count: optional(t.number),
  path: optional(t.string),
};

const HardhatNetworkHDAccountsConfig = t.type({
  mnemonic: optional(t.string),
  accountsBalance: optional(decimalString),
  passphrase: optional(t.string),
  ...commonHDAccountsFields,
});

const Integer = new t.Type<number>(
  "Integer",
  (num: unknown): num is number => typeof num === "number",
  (u, c) => {
    try {
      return typeof u === "string"
        ? t.success(parseInt(u, 10))
        : t.failure(u, c);
    } catch {
      return t.failure(u, c);
    }
  },
  t.identity
);

const HardhatNetworkForkingConfig = t.type({
  enabled: optional(t.boolean),
  url: t.string,
  blockNumber: optional(t.number),
});

const HardhatNetworkMempoolConfig = t.type({
  order: optional(
    t.keyof(
      fromEntries(
        HARDHAT_MEMPOOL_SUPPORTED_ORDERS.map((order) => [order, null])
      )
    )
  ),
});

const HardhatNetworkMiningConfig = t.type({
  auto: optional(t.boolean),
  interval: optional(t.union([t.number, t.tuple([t.number, t.number])])),
  mempool: optional(HardhatNetworkMempoolConfig),
});

function isValidHardforkName(name: string) {
  return Object.values(HardforkName).includes(name as HardforkName);
}

const HardforkNameType = new t.Type<HardforkName>(
  Object.values(HardforkName)
    .map((v) => `"${v}"`)
    .join(" | "),
  (name: unknown): name is HardforkName =>
    typeof name === "string" && isValidHardforkName(name),
  (u, c) => {
    return typeof u === "string" && isValidHardforkName(u)
      ? t.success(u as HardforkName)
      : t.failure(u, c);
  },
  t.identity
);

const HardhatNetworkHardforkHistory = t.record(
  HardforkNameType,
  t.number,
  "HardhatNetworkHardforkHistory"
);

const HardhatNetworkChainConfig = t.type({
  hardforkHistory: HardhatNetworkHardforkHistory,
});

const HardhatNetworkChainsConfig = t.record(Integer, HardhatNetworkChainConfig);

const commonNetworkConfigFields = {
  chainId: optional(t.number),
  from: optional(t.string),
  gas: optional(t.union([t.literal("auto"), t.number])),
  gasPrice: optional(t.union([t.literal("auto"), t.number])),
  gasMultiplier: optional(t.number),
};

const HardhatNetworkConfig = t.type({
  ...commonNetworkConfigFields,
  hardfork: optional(
    t.keyof(
      fromEntries(HARDHAT_NETWORK_SUPPORTED_HARDFORKS.map((hf) => [hf, null]))
    )
  ),
  accounts: optional(
    t.union([t.array(HardhatNetworkAccount), HardhatNetworkHDAccountsConfig])
  ),
  blockGasLimit: optional(t.number),
  minGasPrice: optional(t.union([t.number, t.string])),
  throwOnTransactionFailures: optional(t.boolean),
  throwOnCallFailures: optional(t.boolean),
  allowUnlimitedContractSize: optional(t.boolean),
  initialDate: optional(t.string),
  loggingEnabled: optional(t.boolean),
  forking: optional(HardhatNetworkForkingConfig),
  mining: optional(HardhatNetworkMiningConfig),
  coinbase: optional(address),
  chains: optional(HardhatNetworkChainsConfig),
});

const HDAccountsConfig = t.type({
  mnemonic: t.string,
  passphrase: optional(t.string),
  ...commonHDAccountsFields,
});

const NetworkConfigAccounts = t.union([
  t.literal("remote"),
  t.array(hexString),
  HDAccountsConfig,
]);

const HttpHeaders = t.record(t.string, t.string, "httpHeaders");

const HttpNetworkConfig = t.type({
  ...commonNetworkConfigFields,
  url: optional(t.string),
  accounts: optional(NetworkConfigAccounts),
  httpHeaders: optional(HttpHeaders),
  timeout: optional(t.number),
});

const NetworkConfig = t.union([HardhatNetworkConfig, HttpNetworkConfig]);

const Networks = t.record(t.string, NetworkConfig);

const ProjectPaths = t.type({
  root: optional(t.string),
  cache: optional(t.string),
  artifacts: optional(t.string),
  sources: optional(t.string),
  tests: optional(t.string),
});

const SingleSolcConfig = t.type({
  version: t.string,
  settings: optional(t.any),
});

const MultiSolcConfig = t.type({
  compilers: t.array(SingleSolcConfig),
  overrides: optional(t.record(t.string, SingleSolcConfig)),
});

const SolidityConfig = t.union([t.string, SingleSolcConfig, MultiSolcConfig]);

const HardhatConfig = t.type(
  {
    defaultNetwork: optional(t.string),
    networks: optional(Networks),
    paths: optional(ProjectPaths),
    solidity: optional(SolidityConfig),
  },
  "HardhatConfig"
);

/**
 * Validates the config, throwing a HardhatError if invalid.
 * @param config
 */
export function validateConfig(config: any) {
  const errors = getValidationErrors(config);

  if (errors.length === 0) {
    return;
  }

  let errorList = errors.join("\n  * ");
  errorList = `  * ${errorList}`;

  throw new HardhatError(ERRORS.GENERAL.INVALID_CONFIG, { errors: errorList });
}

export function getValidationErrors(config: any): string[] {
  const errors: string[] = [];

  // These can't be validated with io-ts
  if (config !== undefined && typeof config.networks === "object") {
    const hardhatNetwork = config.networks[HARDHAT_NETWORK_NAME];
    if (hardhatNetwork !== undefined && typeof hardhatNetwork === "object") {
      if ("url" in hardhatNetwork) {
        errors.push(
          `HardhatConfig.networks.${HARDHAT_NETWORK_NAME} can't have a url`
        );
      }

      // Validating the accounts with io-ts leads to very confusing errors messages
      const { accounts, ...configExceptAccounts } = hardhatNetwork;

      const netConfigResult = HardhatNetworkConfig.decode(configExceptAccounts);
      if (netConfigResult.isLeft()) {
        errors.push(
          getErrorMessage(
            `HardhatConfig.networks.${HARDHAT_NETWORK_NAME}`,
            hardhatNetwork,
            "HardhatNetworkConfig"
          )
        );
      }

      // manual validation of accounts
      if (Array.isArray(accounts)) {
        for (const [index, account] of accounts.entries()) {
          if (typeof account !== "object") {
            errors.push(
              getPrivateKeyError(
                index,
                HARDHAT_NETWORK_NAME,
                `Expected object, received ${typeof account}`
              )
            );
            continue;
          }

          const { privateKey, balance } = account;

          validatePrivateKey(privateKey, index, HARDHAT_NETWORK_NAME, errors);

          if (typeof balance !== "string") {
            errors.push(
              getErrorMessage(
                `HardhatConfig.networks.${HARDHAT_NETWORK_NAME}.accounts[].balance`,
                balance,
                "string"
              )
            );
          } else if (decimalString.decode(balance).isLeft()) {
            errors.push(
              getErrorMessage(
                `HardhatConfig.networks.${HARDHAT_NETWORK_NAME}.accounts[].balance`,
                balance,
                "decimal(wei)"
              )
            );
          }
        }
      } else if (typeof hardhatNetwork.accounts === "object") {
        const hdConfigResult = HardhatNetworkHDAccountsConfig.decode(
          hardhatNetwork.accounts
        );
        if (hdConfigResult.isLeft()) {
          errors.push(
            getErrorMessage(
              `HardhatConfig.networks.${HARDHAT_NETWORK_NAME}.accounts`,
              hardhatNetwork.accounts,
              "[{privateKey: string, balance: string}] | HardhatNetworkHDAccountsConfig | undefined"
            )
          );
        }
      } else if (hardhatNetwork.accounts !== undefined) {
        errors.push(
          getErrorMessage(
            `HardhatConfig.networks.${HARDHAT_NETWORK_NAME}.accounts`,
            hardhatNetwork.accounts,
            "[{privateKey: string, balance: string}] | HardhatNetworkHDAccountsConfig | undefined"
          )
        );
      }

      const hardfork =
        hardhatNetwork.hardfork ?? defaultHardhatNetworkParams.hardfork;
      if (hardforkGte(hardfork, HardforkName.LONDON)) {
        if (hardhatNetwork.minGasPrice !== undefined) {
          errors.push(
            `Unexpected config HardhatConfig.networks.${HARDHAT_NETWORK_NAME}.minGasPrice found - This field is not valid for networks with EIP-1559. Try an older hardfork or remove it.`
          );
        }
      } else {
        if (hardhatNetwork.initialBaseFeePerGas !== undefined) {
          errors.push(
            `Unexpected config HardhatConfig.networks.${HARDHAT_NETWORK_NAME}.initialBaseFeePerGas found - This field is only valid for networks with EIP-1559. Try a newer hardfork or remove it.`
          );
        }
      }

      if (hardhatNetwork.chains !== undefined) {
        Object.entries(hardhatNetwork.chains).forEach((chainEntry) => {
          const [chainId, chainConfig] = chainEntry as [
            string,
            HardhatNetworkChainUserConfig
          ];
          const { hardforkHistory } = chainConfig;
          if (hardforkHistory !== undefined) {
            Object.keys(hardforkHistory).forEach((hardforkName) => {
              if (!HARDHAT_NETWORK_SUPPORTED_HARDFORKS.includes(hardforkName)) {
                errors.push(
                  getErrorMessage(
                    `HardhatConfig.networks.${HARDHAT_NETWORK_NAME}.chains[${chainId}].hardforkHistory`,
                    hardforkName,
                    `"${HARDHAT_NETWORK_SUPPORTED_HARDFORKS.join('" | "')}"`
                  )
                );
              }
            });
          }
        });
      }

      if (hardhatNetwork.hardfork !== undefined) {
        if (
          !hardforkGte(hardhatNetwork.hardfork, HardforkName.CANCUN) &&
          hardhatNetwork.enableTransientStorage === true
        ) {
          errors.push(
            `'enableTransientStorage' cannot be enabled if the hardfork is explicitly set to a pre-cancun value. If you want to use transient storage, use 'cancun' as the hardfork.`
          );
        }
        if (
          hardforkGte(hardhatNetwork.hardfork, HardforkName.CANCUN) &&
          hardhatNetwork.enableTransientStorage === false
        ) {
          errors.push(
            `'enableTransientStorage' cannot be disabled if the hardfork is explicitly set to cancun or later. If you want to disable transient storage, use a hardfork before 'cancun'.`
          );
        }
      }
    }

    for (const [networkName, netConfig] of Object.entries<any>(
      config.networks
    )) {
      if (networkName === HARDHAT_NETWORK_NAME) {
        continue;
      }

      if (networkName !== "localhost" || netConfig.url !== undefined) {
        if (typeof netConfig.url !== "string") {
          errors.push(
            getErrorMessage(
              `HardhatConfig.networks.${networkName}.url`,
              netConfig.url,
              "string"
            )
          );
        }
      }

      const { accounts, ...configExceptAccounts } = netConfig;

      const netConfigResult = HttpNetworkConfig.decode(configExceptAccounts);
      if (netConfigResult.isLeft()) {
        errors.push(
          getErrorMessage(
            `HardhatConfig.networks.${networkName}`,
            netConfig,
            "HttpNetworkConfig"
          )
        );
      }

      // manual validation of accounts
      if (Array.isArray(accounts)) {
        accounts.forEach((privateKey, index) =>
          validatePrivateKey(privateKey, index, networkName, errors)
        );
      } else if (typeof accounts === "object") {
        const hdConfigResult = HDAccountsConfig.decode(accounts);
        if (hdConfigResult.isLeft()) {
          errors.push(
            getErrorMessage(
              `HardhatConfig.networks.${networkName}`,
              accounts,
              "HttpNetworkHDAccountsConfig"
            )
          );
        }
      } else if (typeof accounts === "string") {
        if (accounts !== "remote") {
          errors.push(
            `Invalid 'accounts' entry for network '${networkName}': expected an array of accounts or the string 'remote', but got the string '${accounts}'`
          );
        }
      } else if (accounts !== undefined) {
        errors.push(
          getErrorMessage(
            `HardhatConfig.networks.${networkName}.accounts`,
            accounts,
            '"remote" | string[] | HttpNetworkHDAccountsConfig | undefined'
          )
        );
      }
    }
  }

  // io-ts can get confused if there are errors that it can't understand.
  // Especially around Hardhat Network's config. It will treat it as an HTTPConfig,
  // and may give a loot of errors.
  if (errors.length > 0) {
    return errors;
  }

  const result = HardhatConfig.decode(config);

  if (result.isRight()) {
    return errors;
  }

  const ioTsErrors = DotPathReporter.report(result);
  return [...errors, ...ioTsErrors];
}

export function validateResolvedConfig(resolvedConfig: HardhatConfigT) {
  const solcConfigs = [
    ...resolvedConfig.solidity.compilers,
    ...Object.values(resolvedConfig.solidity.overrides),
  ];
  const runs = solcConfigs
    .filter(({ settings }) => settings?.optimizer?.runs !== undefined)
    .map(({ settings }) => settings?.optimizer?.runs);

  for (const run of runs) {
    if (run >= 2 ** 32) {
      throw new HardhatError(ERRORS.GENERAL.INVALID_CONFIG, {
        errors: "The number of optimizer runs exceeds the maximum of 2**32 - 1",
      });
    }
  }
}
