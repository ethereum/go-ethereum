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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.encodeArguments = exports.getCompilerVersions = exports.resolveLibraries = exports.resolveConstructorArguments = exports.printVerificationErrors = exports.printSupportedNetworks = exports.sleep = void 0;
const chalk_1 = __importDefault(require("chalk"));
const path_1 = __importDefault(require("path"));
const chain_config_1 = require("./chain-config");
const errors_1 = require("./errors");
const abi_validation_extras_1 = require("./abi-validation-extras");
async function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}
exports.sleep = sleep;
/**
 * Prints a table of networks supported by hardhat-verify, including both
 * built-in and custom networks.
 */
async function printSupportedNetworks(customChains) {
    const { table } = await Promise.resolve().then(() => __importStar(require("table")));
    // supported networks
    const supportedNetworks = chain_config_1.builtinChains.map(({ network, chainId }) => [
        network,
        chainId,
    ]);
    const supportedNetworksTable = table([
        [chalk_1.default.bold("network"), chalk_1.default.bold("chain id")],
        ...supportedNetworks,
    ]);
    // custom networks
    const customNetworks = customChains.map(({ network, chainId }) => [
        network,
        chainId,
    ]);
    const customNetworksTable = customNetworks.length > 0
        ? table([
            [chalk_1.default.bold("network"), chalk_1.default.bold("chain id")],
            ...customNetworks,
        ])
        : table([["No custom networks were added"]]);
    // print message
    console.log(`
Networks supported by hardhat-verify:

${supportedNetworksTable}

Custom networks added by you or by plugins:

${customNetworksTable}

To learn how to add custom networks, follow these instructions: https://hardhat.org/verify-custom-networks
`.trimStart());
}
exports.printSupportedNetworks = printSupportedNetworks;
/**
 * Prints verification errors to the console.
 * @param errors - An object containing verification errors, where the keys
 * are the names of verification subtasks and the values are HardhatVerifyError
 * objects describing the specific errors.
 * @remarks This function formats and logs the verification errors to the
 * console with a red color using chalk. Each error is displayed along with the
 * name of the verification provider it belongs to.
 * @example
 * const errors: Record<string, HardhatVerifyError> = {
 *   verify:etherscan: { message: 'Error message for Etherscan' },
 *   verify:sourcify: { message: 'Error message for Sourcify' },
 *   // Add more errors here...
 * };
 * printVerificationErrors(errors);
 * // Output:
 * // hardhat-verify found one or more errors during the verification process:
 * //
 * // Etherscan:
 * // Error message for Etherscan
 * //
 * // Sourcify:
 * // Error message for Sourcify
 * //
 * // ... (more errors if present)
 */
function printVerificationErrors(errors) {
    let errorMessage = "hardhat-verify found one or more errors during the verification process:\n\n";
    for (const [subtaskLabel, error] of Object.entries(errors)) {
        errorMessage += `${subtaskLabel}:\n${error.message}\n\n`;
    }
    console.error(chalk_1.default.red(errorMessage));
}
exports.printVerificationErrors = printVerificationErrors;
/**
 * Returns the list of constructor arguments from the constructorArgsModule
 * or the constructorArgsParams if the first is not defined.
 */
async function resolveConstructorArguments(constructorArgsParams, constructorArgsModule) {
    if (constructorArgsModule === undefined) {
        return constructorArgsParams;
    }
    if (constructorArgsParams.length > 0) {
        throw new errors_1.ExclusiveConstructorArgumentsError();
    }
    const constructorArgsModulePath = path_1.default.resolve(process.cwd(), constructorArgsModule);
    try {
        const constructorArguments = (await Promise.resolve(`${constructorArgsModulePath}`).then(s => __importStar(require(s))))
            .default;
        if (!Array.isArray(constructorArguments)) {
            throw new errors_1.InvalidConstructorArgumentsModuleError(constructorArgsModulePath);
        }
        return constructorArguments;
    }
    catch (error) {
        throw new errors_1.ImportingModuleError("constructor arguments list", error);
    }
}
exports.resolveConstructorArguments = resolveConstructorArguments;
/**
 * Returns a dictionary of library addresses from the librariesModule or
 * an empty object if not defined.
 */
async function resolveLibraries(librariesModule) {
    if (librariesModule === undefined) {
        return {};
    }
    const librariesModulePath = path_1.default.resolve(process.cwd(), librariesModule);
    try {
        const libraries = (await Promise.resolve(`${librariesModulePath}`).then(s => __importStar(require(s)))).default;
        if (typeof libraries !== "object" || Array.isArray(libraries)) {
            throw new errors_1.InvalidLibrariesModuleError(librariesModulePath);
        }
        return libraries;
    }
    catch (error) {
        throw new errors_1.ImportingModuleError("libraries dictionary", error);
    }
}
exports.resolveLibraries = resolveLibraries;
/**
 * Retrieves the list of Solidity compiler versions for a given Solidity
 * configuration.
 * It checks that the versions are supported by Etherscan, and throws an
 * error if any are not.
 */
async function getCompilerVersions({ compilers, overrides, }) {
    {
        const compilerVersions = compilers.map(({ version }) => version);
        if (overrides !== undefined) {
            for (const { version } of Object.values(overrides)) {
                compilerVersions.push(version);
            }
        }
        // Etherscan only supports solidity versions higher than or equal to v0.4.11.
        // See https://etherscan.io/solcversions
        const supportedSolcVersionRange = ">=0.4.11";
        const semver = await Promise.resolve().then(() => __importStar(require("semver")));
        if (compilerVersions.some((version) => !semver.satisfies(version, supportedSolcVersionRange))) {
            throw new errors_1.EtherscanVersionNotSupportedError();
        }
        return compilerVersions;
    }
}
exports.getCompilerVersions = getCompilerVersions;
/**
 * Encodes the constructor arguments for a given contract.
 */
async function encodeArguments(abi, sourceName, contractName, constructorArguments) {
    const { Interface } = await Promise.resolve().then(() => __importStar(require("@ethersproject/abi")));
    const contractInterface = new Interface(abi);
    let encodedConstructorArguments;
    try {
        // encodeDeploy doesn't catch subtle type mismatches, such as a number
        // being passed when a string is expected, so we have to validate the
        // scenario manually.
        const expectedConstructorArgs = contractInterface.deploy.inputs;
        constructorArguments.forEach((arg, i) => {
            if (expectedConstructorArgs[i]?.type === "string" &&
                typeof arg !== "string") {
                throw new errors_1.ABIArgumentTypeError({
                    code: "INVALID_ARGUMENT",
                    argument: expectedConstructorArgs[i].name,
                    value: arg,
                    reason: "invalid string value",
                });
            }
        });
        encodedConstructorArguments = contractInterface
            .encodeDeploy(constructorArguments)
            .replace("0x", "");
    }
    catch (error) {
        if ((0, abi_validation_extras_1.isABIArgumentLengthError)(error)) {
            throw new errors_1.ABIArgumentLengthError(sourceName, contractName, error);
        }
        if ((0, abi_validation_extras_1.isABIArgumentTypeError)(error)) {
            throw new errors_1.ABIArgumentTypeError(error);
        }
        if ((0, abi_validation_extras_1.isABIArgumentOverflowError)(error)) {
            throw new errors_1.ABIArgumentOverflowError(error);
        }
        // Should be unreachable.
        throw error;
    }
    return encodedConstructorArguments;
}
exports.encodeArguments = encodeArguments;
//# sourceMappingURL=utilities.js.map