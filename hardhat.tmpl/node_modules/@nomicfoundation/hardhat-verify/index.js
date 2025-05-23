"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const picocolors_1 = __importDefault(require("picocolors"));
const config_1 = require("hardhat/config");
const task_names_1 = require("./internal/task-names");
const config_2 = require("./internal/config");
const errors_1 = require("./internal/errors");
const utilities_1 = require("./internal/utilities");
const artifacts_1 = require("./internal/solc/artifacts");
require("./internal/type-extensions");
require("./internal/tasks/etherscan");
require("./internal/tasks/sourcify");
require("./internal/tasks/blockscout");
(0, config_1.extendConfig)(config_2.etherscanConfigExtender);
(0, config_1.extendConfig)(config_2.sourcifyConfigExtender);
(0, config_1.extendConfig)(config_2.blockscoutConfigExtender);
/**
 * Main verification task.
 *
 * This is a meta-task that gets all the verification tasks and runs them.
 * It supports Etherscan and Sourcify.
 */
(0, config_1.task)(task_names_1.TASK_VERIFY, "Verifies a contract on Etherscan or Sourcify")
    .addOptionalPositionalParam("address", "Address of the contract to verify")
    .addOptionalVariadicPositionalParam("constructorArgsParams", "Contract constructor arguments. Cannot be used if the --constructor-args option is provided", [])
    .addOptionalParam("constructorArgs", "Path to a Javascript module that exports the constructor arguments", undefined, config_1.types.inputFile)
    .addOptionalParam("libraries", "Path to a Javascript module that exports a dictionary of library addresses. " +
    "Use if there are undetectable library addresses in your contract. " +
    "Library addresses are undetectable if they are only used in the contract constructor", undefined, config_1.types.inputFile)
    .addOptionalParam("contract", "Fully qualified name of the contract to verify. Skips automatic detection of the contract. " +
    "Use if the deployed bytecode matches more than one contract in your project")
    .addFlag("force", "Enforce contract verification even if the contract is already verified. " +
    "Use to re-verify partially verified contracts on Blockscout")
    .addFlag("listNetworks", "Print the list of supported networks")
    .setAction(async (taskArgs, { run }) => {
    if (taskArgs.listNetworks) {
        await run(task_names_1.TASK_VERIFY_PRINT_SUPPORTED_NETWORKS);
        return;
    }
    const verificationSubtasks = await run(task_names_1.TASK_VERIFY_GET_VERIFICATION_SUBTASKS);
    const errors = {};
    for (const { label, subtaskName } of verificationSubtasks) {
        try {
            await run(subtaskName, taskArgs);
        }
        catch (error) {
            errors[label] = error;
        }
    }
    const hasErrors = Object.keys(errors).length > 0;
    if (hasErrors) {
        (0, utilities_1.printVerificationErrors)(errors);
        process.exit(1);
    }
});
(0, config_1.subtask)(task_names_1.TASK_VERIFY_PRINT_SUPPORTED_NETWORKS, "Prints the supported networks list").setAction(async ({}, { config }) => {
    await (0, utilities_1.printSupportedNetworks)(config.etherscan.customChains);
});
(0, config_1.subtask)(task_names_1.TASK_VERIFY_GET_VERIFICATION_SUBTASKS, async (_, { config, userConfig }) => {
    const verificationSubtasks = [];
    if (config.etherscan.enabled) {
        verificationSubtasks.push({
            label: "Etherscan",
            subtaskName: task_names_1.TASK_VERIFY_ETHERSCAN,
        });
    }
    if (config.sourcify.enabled) {
        verificationSubtasks.push({
            label: "Sourcify",
            subtaskName: task_names_1.TASK_VERIFY_SOURCIFY,
        });
    }
    else if (userConfig.sourcify?.enabled === undefined) {
        verificationSubtasks.unshift({
            label: "Common",
            subtaskName: task_names_1.TASK_VERIFY_SOURCIFY_DISABLED_WARNING,
        });
    }
    if (config.blockscout.enabled) {
        verificationSubtasks.push({
            label: "Blockscout",
            subtaskName: task_names_1.TASK_VERIFY_BLOCKSCOUT,
        });
    }
    if (!config.etherscan.enabled &&
        !config.sourcify.enabled &&
        !config.blockscout.enabled) {
        console.warn(picocolors_1.default.yellow(`[WARNING] No verification services are enabled. Please enable at least one verification service in your configuration.`));
    }
    return verificationSubtasks;
});
(0, config_1.subtask)(task_names_1.TASK_VERIFY_GET_CONTRACT_INFORMATION)
    .addParam("deployedBytecode", undefined, undefined, config_1.types.any)
    .addParam("matchingCompilerVersions", undefined, undefined, config_1.types.any)
    .addParam("libraries", undefined, undefined, config_1.types.any)
    .addOptionalParam("contractFQN")
    .setAction(async ({ contractFQN, deployedBytecode, matchingCompilerVersions, libraries, }, { network, artifacts }) => {
    let contractInformation;
    if (contractFQN !== undefined) {
        const artifactExists = await artifacts.artifactExists(contractFQN);
        if (!artifactExists) {
            throw new errors_1.ContractNotFoundError(contractFQN);
        }
        const buildInfo = await artifacts.getBuildInfo(contractFQN);
        if (buildInfo === undefined) {
            throw new errors_1.BuildInfoNotFoundError(contractFQN);
        }
        if (!matchingCompilerVersions.includes(buildInfo.solcVersion) &&
            !deployedBytecode.isOvm()) {
            throw new errors_1.BuildInfoCompilerVersionMismatchError(contractFQN, deployedBytecode.getVersion(), deployedBytecode.hasVersionRange(), buildInfo.solcVersion, network.name);
        }
        contractInformation = (0, artifacts_1.extractMatchingContractInformation)(contractFQN, buildInfo, deployedBytecode);
        if (contractInformation === null) {
            throw new errors_1.DeployedBytecodeMismatchError(network.name, contractFQN);
        }
    }
    else {
        contractInformation = await (0, artifacts_1.extractInferredContractInformation)(artifacts, network, matchingCompilerVersions, deployedBytecode);
    }
    // map contractInformation libraries
    const libraryInformation = await (0, artifacts_1.getLibraryInformation)(contractInformation, libraries);
    return {
        ...contractInformation,
        ...libraryInformation,
    };
});
/**
 * This subtask is used to programmatically verify a contract on Etherscan or Sourcify.
 */
(0, config_1.subtask)(task_names_1.TASK_VERIFY_VERIFY)
    .addOptionalParam("address")
    .addOptionalParam("constructorArguments", undefined, [], config_1.types.any)
    .addOptionalParam("libraries", undefined, {}, config_1.types.any)
    .addOptionalParam("contract")
    .addFlag("force")
    .setAction(async ({ address, constructorArguments, libraries, contract, force, }, { run, config }) => {
    // This can only happen if the subtask is invoked from within Hardhat by a user script or another task.
    if (!Array.isArray(constructorArguments)) {
        throw new errors_1.InvalidConstructorArgumentsError();
    }
    if (typeof libraries !== "object" || Array.isArray(libraries)) {
        throw new errors_1.InvalidLibrariesError();
    }
    if (config.etherscan.enabled) {
        await run(task_names_1.TASK_VERIFY_ETHERSCAN, {
            address,
            constructorArgsParams: constructorArguments,
            libraries,
            contract,
            force,
        });
    }
    if (config.sourcify.enabled) {
        await run(task_names_1.TASK_VERIFY_SOURCIFY, {
            address,
            libraries,
            contract,
        });
    }
});
//# sourceMappingURL=index.js.map