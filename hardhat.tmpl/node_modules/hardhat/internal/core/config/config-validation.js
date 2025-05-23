"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.validateResolvedConfig = exports.getValidationErrors = exports.validateConfig = exports.decimalString = exports.address = exports.hexString = exports.DotPathReporter = exports.success = exports.failure = void 0;
const t = __importStar(require("io-ts"));
const constants_1 = require("../../constants");
const io_ts_1 = require("../../util/io-ts");
const lang_1 = require("../../util/lang");
const errors_1 = require("../errors");
const errors_list_1 = require("../errors-list");
const hardforks_1 = require("../../util/hardforks");
const default_config_1 = require("./default-config");
function stringify(v) {
    if (typeof v === "function") {
        const { getFunctionName } = require("io-ts/lib");
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
function getContextPath(context) {
    const keysPath = context
        .slice(1)
        .map((c) => c.key)
        .join(".");
    return `${context[0].type.name}.${keysPath}`;
}
function getMessage(e) {
    const lastContext = e.context[e.context.length - 1];
    return e.message !== undefined
        ? e.message
        : getErrorMessage(getContextPath(e.context), e.value, lastContext.type.name);
}
function getErrorMessage(path, value, expectedType) {
    return `Invalid value ${stringify(value)} for ${path} - Expected a value of type ${expectedType}.`;
}
function getPrivateKeyError(index, network, message) {
    return `Invalid account: #${index} for network: ${network} - ${message}`;
}
function validatePrivateKey(privateKey, index, network, errors) {
    if (typeof privateKey !== "string") {
        errors.push(getPrivateKeyError(index, network, `Expected string, received ${typeof privateKey}`));
    }
    else {
        // private key validation
        const pkWithPrefix = /^0x/.test(privateKey)
            ? privateKey
            : `0x${privateKey}`;
        // 32 bytes = 64 characters + 2 char prefix = 66
        if (pkWithPrefix.length < 66) {
            errors.push(getPrivateKeyError(index, network, "private key too short, expected 32 bytes"));
        }
        else if (pkWithPrefix.length > 66) {
            errors.push(getPrivateKeyError(index, network, "private key too long, expected 32 bytes"));
        }
        else if (exports.hexString.decode(pkWithPrefix).isLeft()) {
            errors.push(getPrivateKeyError(index, network, "invalid hex character(s) found in string"));
        }
    }
}
function failure(es) {
    return es.map(getMessage);
}
exports.failure = failure;
function success() {
    return [];
}
exports.success = success;
exports.DotPathReporter = {
    report: (validation) => validation.fold(failure, success),
};
const HEX_STRING_REGEX = /^(0x)?([0-9a-f]{2})+$/gi;
const DEC_STRING_REGEX = /^(0|[1-9][0-9]*)$/g;
function isHexString(v) {
    if (typeof v !== "string") {
        return false;
    }
    return v.trim().match(HEX_STRING_REGEX) !== null;
}
function isDecimalString(v) {
    if (typeof v !== "string") {
        return false;
    }
    return v.match(DEC_STRING_REGEX) !== null;
}
exports.hexString = new t.Type("hex string", isHexString, (u, c) => (isHexString(u) ? t.success(u) : t.failure(u, c)), t.identity);
function isAddress(v) {
    if (typeof v !== "string") {
        return false;
    }
    const trimmed = v.trim();
    return (trimmed.match(HEX_STRING_REGEX) !== null &&
        trimmed.startsWith("0x") &&
        trimmed.length === 42);
}
exports.address = new t.Type("address", isAddress, (u, c) => (isAddress(u) ? t.success(u) : t.failure(u, c)), t.identity);
exports.decimalString = new t.Type("decimal string", isDecimalString, (u, c) => (isDecimalString(u) ? t.success(u) : t.failure(u, c)), t.identity);
// TODO: These types have outdated name. They should match the UserConfig types.
// IMPORTANT: This t.types MUST be kept in sync with the actual types.
const HardhatNetworkAccount = t.type({
    privateKey: exports.hexString,
    balance: exports.decimalString,
});
const commonHDAccountsFields = {
    initialIndex: (0, io_ts_1.optional)(t.number),
    count: (0, io_ts_1.optional)(t.number),
    path: (0, io_ts_1.optional)(t.string),
};
const HardhatNetworkHDAccountsConfig = t.type({
    mnemonic: (0, io_ts_1.optional)(t.string),
    accountsBalance: (0, io_ts_1.optional)(exports.decimalString),
    passphrase: (0, io_ts_1.optional)(t.string),
    ...commonHDAccountsFields,
});
const Integer = new t.Type("Integer", (num) => typeof num === "number", (u, c) => {
    try {
        return typeof u === "string"
            ? t.success(parseInt(u, 10))
            : t.failure(u, c);
    }
    catch {
        return t.failure(u, c);
    }
}, t.identity);
const HardhatNetworkForkingConfig = t.type({
    enabled: (0, io_ts_1.optional)(t.boolean),
    url: t.string,
    blockNumber: (0, io_ts_1.optional)(t.number),
});
const HardhatNetworkMempoolConfig = t.type({
    order: (0, io_ts_1.optional)(t.keyof((0, lang_1.fromEntries)(constants_1.HARDHAT_MEMPOOL_SUPPORTED_ORDERS.map((order) => [order, null])))),
});
const HardhatNetworkMiningConfig = t.type({
    auto: (0, io_ts_1.optional)(t.boolean),
    interval: (0, io_ts_1.optional)(t.union([t.number, t.tuple([t.number, t.number])])),
    mempool: (0, io_ts_1.optional)(HardhatNetworkMempoolConfig),
});
function isValidHardforkName(name) {
    return Object.values(hardforks_1.HardforkName).includes(name);
}
const HardforkNameType = new t.Type(Object.values(hardforks_1.HardforkName)
    .map((v) => `"${v}"`)
    .join(" | "), (name) => typeof name === "string" && isValidHardforkName(name), (u, c) => {
    return typeof u === "string" && isValidHardforkName(u)
        ? t.success(u)
        : t.failure(u, c);
}, t.identity);
const HardhatNetworkHardforkHistory = t.record(HardforkNameType, t.number, "HardhatNetworkHardforkHistory");
const HardhatNetworkChainConfig = t.type({
    hardforkHistory: HardhatNetworkHardforkHistory,
});
const HardhatNetworkChainsConfig = t.record(Integer, HardhatNetworkChainConfig);
const commonNetworkConfigFields = {
    chainId: (0, io_ts_1.optional)(t.number),
    from: (0, io_ts_1.optional)(t.string),
    gas: (0, io_ts_1.optional)(t.union([t.literal("auto"), t.number])),
    gasPrice: (0, io_ts_1.optional)(t.union([t.literal("auto"), t.number])),
    gasMultiplier: (0, io_ts_1.optional)(t.number),
};
const HardhatNetworkConfig = t.type({
    ...commonNetworkConfigFields,
    hardfork: (0, io_ts_1.optional)(t.keyof((0, lang_1.fromEntries)(constants_1.HARDHAT_NETWORK_SUPPORTED_HARDFORKS.map((hf) => [hf, null])))),
    accounts: (0, io_ts_1.optional)(t.union([t.array(HardhatNetworkAccount), HardhatNetworkHDAccountsConfig])),
    blockGasLimit: (0, io_ts_1.optional)(t.number),
    minGasPrice: (0, io_ts_1.optional)(t.union([t.number, t.string])),
    throwOnTransactionFailures: (0, io_ts_1.optional)(t.boolean),
    throwOnCallFailures: (0, io_ts_1.optional)(t.boolean),
    allowUnlimitedContractSize: (0, io_ts_1.optional)(t.boolean),
    initialDate: (0, io_ts_1.optional)(t.string),
    loggingEnabled: (0, io_ts_1.optional)(t.boolean),
    forking: (0, io_ts_1.optional)(HardhatNetworkForkingConfig),
    mining: (0, io_ts_1.optional)(HardhatNetworkMiningConfig),
    coinbase: (0, io_ts_1.optional)(exports.address),
    chains: (0, io_ts_1.optional)(HardhatNetworkChainsConfig),
});
const HDAccountsConfig = t.type({
    mnemonic: t.string,
    passphrase: (0, io_ts_1.optional)(t.string),
    ...commonHDAccountsFields,
});
const NetworkConfigAccounts = t.union([
    t.literal("remote"),
    t.array(exports.hexString),
    HDAccountsConfig,
]);
const HttpHeaders = t.record(t.string, t.string, "httpHeaders");
const HttpNetworkConfig = t.type({
    ...commonNetworkConfigFields,
    url: (0, io_ts_1.optional)(t.string),
    accounts: (0, io_ts_1.optional)(NetworkConfigAccounts),
    httpHeaders: (0, io_ts_1.optional)(HttpHeaders),
    timeout: (0, io_ts_1.optional)(t.number),
});
const NetworkConfig = t.union([HardhatNetworkConfig, HttpNetworkConfig]);
const Networks = t.record(t.string, NetworkConfig);
const ProjectPaths = t.type({
    root: (0, io_ts_1.optional)(t.string),
    cache: (0, io_ts_1.optional)(t.string),
    artifacts: (0, io_ts_1.optional)(t.string),
    sources: (0, io_ts_1.optional)(t.string),
    tests: (0, io_ts_1.optional)(t.string),
});
const SingleSolcConfig = t.type({
    version: t.string,
    settings: (0, io_ts_1.optional)(t.any),
});
const MultiSolcConfig = t.type({
    compilers: t.array(SingleSolcConfig),
    overrides: (0, io_ts_1.optional)(t.record(t.string, SingleSolcConfig)),
});
const SolidityConfig = t.union([t.string, SingleSolcConfig, MultiSolcConfig]);
const HardhatConfig = t.type({
    defaultNetwork: (0, io_ts_1.optional)(t.string),
    networks: (0, io_ts_1.optional)(Networks),
    paths: (0, io_ts_1.optional)(ProjectPaths),
    solidity: (0, io_ts_1.optional)(SolidityConfig),
}, "HardhatConfig");
/**
 * Validates the config, throwing a HardhatError if invalid.
 * @param config
 */
function validateConfig(config) {
    const errors = getValidationErrors(config);
    if (errors.length === 0) {
        return;
    }
    let errorList = errors.join("\n  * ");
    errorList = `  * ${errorList}`;
    throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.INVALID_CONFIG, { errors: errorList });
}
exports.validateConfig = validateConfig;
function getValidationErrors(config) {
    const errors = [];
    // These can't be validated with io-ts
    if (config !== undefined && typeof config.networks === "object") {
        const hardhatNetwork = config.networks[constants_1.HARDHAT_NETWORK_NAME];
        if (hardhatNetwork !== undefined && typeof hardhatNetwork === "object") {
            if ("url" in hardhatNetwork) {
                errors.push(`HardhatConfig.networks.${constants_1.HARDHAT_NETWORK_NAME} can't have a url`);
            }
            // Validating the accounts with io-ts leads to very confusing errors messages
            const { accounts, ...configExceptAccounts } = hardhatNetwork;
            const netConfigResult = HardhatNetworkConfig.decode(configExceptAccounts);
            if (netConfigResult.isLeft()) {
                errors.push(getErrorMessage(`HardhatConfig.networks.${constants_1.HARDHAT_NETWORK_NAME}`, hardhatNetwork, "HardhatNetworkConfig"));
            }
            // manual validation of accounts
            if (Array.isArray(accounts)) {
                for (const [index, account] of accounts.entries()) {
                    if (typeof account !== "object") {
                        errors.push(getPrivateKeyError(index, constants_1.HARDHAT_NETWORK_NAME, `Expected object, received ${typeof account}`));
                        continue;
                    }
                    const { privateKey, balance } = account;
                    validatePrivateKey(privateKey, index, constants_1.HARDHAT_NETWORK_NAME, errors);
                    if (typeof balance !== "string") {
                        errors.push(getErrorMessage(`HardhatConfig.networks.${constants_1.HARDHAT_NETWORK_NAME}.accounts[].balance`, balance, "string"));
                    }
                    else if (exports.decimalString.decode(balance).isLeft()) {
                        errors.push(getErrorMessage(`HardhatConfig.networks.${constants_1.HARDHAT_NETWORK_NAME}.accounts[].balance`, balance, "decimal(wei)"));
                    }
                }
            }
            else if (typeof hardhatNetwork.accounts === "object") {
                const hdConfigResult = HardhatNetworkHDAccountsConfig.decode(hardhatNetwork.accounts);
                if (hdConfigResult.isLeft()) {
                    errors.push(getErrorMessage(`HardhatConfig.networks.${constants_1.HARDHAT_NETWORK_NAME}.accounts`, hardhatNetwork.accounts, "[{privateKey: string, balance: string}] | HardhatNetworkHDAccountsConfig | undefined"));
                }
            }
            else if (hardhatNetwork.accounts !== undefined) {
                errors.push(getErrorMessage(`HardhatConfig.networks.${constants_1.HARDHAT_NETWORK_NAME}.accounts`, hardhatNetwork.accounts, "[{privateKey: string, balance: string}] | HardhatNetworkHDAccountsConfig | undefined"));
            }
            const hardfork = hardhatNetwork.hardfork ?? default_config_1.defaultHardhatNetworkParams.hardfork;
            if ((0, hardforks_1.hardforkGte)(hardfork, hardforks_1.HardforkName.LONDON)) {
                if (hardhatNetwork.minGasPrice !== undefined) {
                    errors.push(`Unexpected config HardhatConfig.networks.${constants_1.HARDHAT_NETWORK_NAME}.minGasPrice found - This field is not valid for networks with EIP-1559. Try an older hardfork or remove it.`);
                }
            }
            else {
                if (hardhatNetwork.initialBaseFeePerGas !== undefined) {
                    errors.push(`Unexpected config HardhatConfig.networks.${constants_1.HARDHAT_NETWORK_NAME}.initialBaseFeePerGas found - This field is only valid for networks with EIP-1559. Try a newer hardfork or remove it.`);
                }
            }
            if (hardhatNetwork.chains !== undefined) {
                Object.entries(hardhatNetwork.chains).forEach((chainEntry) => {
                    const [chainId, chainConfig] = chainEntry;
                    const { hardforkHistory } = chainConfig;
                    if (hardforkHistory !== undefined) {
                        Object.keys(hardforkHistory).forEach((hardforkName) => {
                            if (!constants_1.HARDHAT_NETWORK_SUPPORTED_HARDFORKS.includes(hardforkName)) {
                                errors.push(getErrorMessage(`HardhatConfig.networks.${constants_1.HARDHAT_NETWORK_NAME}.chains[${chainId}].hardforkHistory`, hardforkName, `"${constants_1.HARDHAT_NETWORK_SUPPORTED_HARDFORKS.join('" | "')}"`));
                            }
                        });
                    }
                });
            }
            if (hardhatNetwork.hardfork !== undefined) {
                if (!(0, hardforks_1.hardforkGte)(hardhatNetwork.hardfork, hardforks_1.HardforkName.CANCUN) &&
                    hardhatNetwork.enableTransientStorage === true) {
                    errors.push(`'enableTransientStorage' cannot be enabled if the hardfork is explicitly set to a pre-cancun value. If you want to use transient storage, use 'cancun' as the hardfork.`);
                }
                if ((0, hardforks_1.hardforkGte)(hardhatNetwork.hardfork, hardforks_1.HardforkName.CANCUN) &&
                    hardhatNetwork.enableTransientStorage === false) {
                    errors.push(`'enableTransientStorage' cannot be disabled if the hardfork is explicitly set to cancun or later. If you want to disable transient storage, use a hardfork before 'cancun'.`);
                }
            }
        }
        for (const [networkName, netConfig] of Object.entries(config.networks)) {
            if (networkName === constants_1.HARDHAT_NETWORK_NAME) {
                continue;
            }
            if (networkName !== "localhost" || netConfig.url !== undefined) {
                if (typeof netConfig.url !== "string") {
                    errors.push(getErrorMessage(`HardhatConfig.networks.${networkName}.url`, netConfig.url, "string"));
                }
            }
            const { accounts, ...configExceptAccounts } = netConfig;
            const netConfigResult = HttpNetworkConfig.decode(configExceptAccounts);
            if (netConfigResult.isLeft()) {
                errors.push(getErrorMessage(`HardhatConfig.networks.${networkName}`, netConfig, "HttpNetworkConfig"));
            }
            // manual validation of accounts
            if (Array.isArray(accounts)) {
                accounts.forEach((privateKey, index) => validatePrivateKey(privateKey, index, networkName, errors));
            }
            else if (typeof accounts === "object") {
                const hdConfigResult = HDAccountsConfig.decode(accounts);
                if (hdConfigResult.isLeft()) {
                    errors.push(getErrorMessage(`HardhatConfig.networks.${networkName}`, accounts, "HttpNetworkHDAccountsConfig"));
                }
            }
            else if (typeof accounts === "string") {
                if (accounts !== "remote") {
                    errors.push(`Invalid 'accounts' entry for network '${networkName}': expected an array of accounts or the string 'remote', but got the string '${accounts}'`);
                }
            }
            else if (accounts !== undefined) {
                errors.push(getErrorMessage(`HardhatConfig.networks.${networkName}.accounts`, accounts, '"remote" | string[] | HttpNetworkHDAccountsConfig | undefined'));
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
    const ioTsErrors = exports.DotPathReporter.report(result);
    return [...errors, ...ioTsErrors];
}
exports.getValidationErrors = getValidationErrors;
function validateResolvedConfig(resolvedConfig) {
    const solcConfigs = [
        ...resolvedConfig.solidity.compilers,
        ...Object.values(resolvedConfig.solidity.overrides),
    ];
    const runs = solcConfigs
        .filter(({ settings }) => settings?.optimizer?.runs !== undefined)
        .map(({ settings }) => settings?.optimizer?.runs);
    for (const run of runs) {
        if (run >= 2 ** 32) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.INVALID_CONFIG, {
                errors: "The number of optimizer runs exceeds the maximum of 2**32 - 1",
            });
        }
    }
}
exports.validateResolvedConfig = validateResolvedConfig;
//# sourceMappingURL=config-validation.js.map