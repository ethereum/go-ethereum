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
const config_1 = require("hardhat/config");
const task_names_1 = require("hardhat/builtin-tasks/task-names");
const contract_names_1 = require("hardhat/utils/contract-names");
const errors_1 = require("../errors");
const etherscan_1 = require("../etherscan");
const bytecode_1 = require("../solc/bytecode");
const task_names_2 = require("../task-names");
const utilities_1 = require("../utilities");
/**
 * Main Etherscan verification subtask.
 *
 * Verifies a contract in Etherscan by coordinating various subtasks related
 * to contract verification.
 */
(0, config_1.subtask)(task_names_2.TASK_VERIFY_ETHERSCAN)
    .addParam("address")
    .addOptionalParam("constructorArgsParams", undefined, undefined, config_1.types.any)
    .addOptionalParam("constructorArgs")
    .addOptionalParam("libraries", undefined, undefined, config_1.types.any)
    .addOptionalParam("contract")
    .addFlag("force")
    .setAction(async (taskArgs, { config, network, run }) => {
    const { address, constructorArgs, libraries, contractFQN, force, } = await run(task_names_2.TASK_VERIFY_ETHERSCAN_RESOLVE_ARGUMENTS, taskArgs);
    const chainConfig = await etherscan_1.Etherscan.getCurrentChainConfig(network.name, network.provider, config.etherscan.customChains);
    const etherscan = etherscan_1.Etherscan.fromChainConfig(config.etherscan.apiKey, chainConfig);
    const isVerified = await etherscan.isVerified(address);
    if (!force && isVerified) {
        const contractURL = etherscan.getContractUrl(address);
        console.log(`The contract ${address} has already been verified on the block explorer. If you're trying to verify a partially verified contract, please use the --force flag.
${contractURL}
`);
        return;
    }
    const configCompilerVersions = await (0, utilities_1.getCompilerVersions)(config.solidity);
    const deployedBytecode = await bytecode_1.Bytecode.getDeployedContractBytecode(address, network.provider, network.name);
    const matchingCompilerVersions = await deployedBytecode.getMatchingVersions(configCompilerVersions);
    // don't error if the bytecode appears to be OVM bytecode, because we can't infer a specific OVM solc version from the bytecode
    if (matchingCompilerVersions.length === 0 && !deployedBytecode.isOvm()) {
        throw new errors_1.CompilerVersionsMismatchError(configCompilerVersions, deployedBytecode.getVersion(), network.name);
    }
    const contractInformation = await run(task_names_2.TASK_VERIFY_GET_CONTRACT_INFORMATION, {
        contractFQN,
        deployedBytecode,
        matchingCompilerVersions,
        libraries,
    });
    const minimalInput = await run(task_names_2.TASK_VERIFY_ETHERSCAN_GET_MINIMAL_INPUT, {
        sourceName: contractInformation.sourceName,
    });
    const encodedConstructorArguments = await (0, utilities_1.encodeArguments)(contractInformation.contractOutput.abi, contractInformation.sourceName, contractInformation.contractName, constructorArgs);
    // First, try to verify the contract using the minimal input
    const { success: minimalInputVerificationSuccess } = await run(task_names_2.TASK_VERIFY_ETHERSCAN_ATTEMPT_VERIFICATION, {
        address,
        compilerInput: minimalInput,
        contractInformation,
        verificationInterface: etherscan,
        encodedConstructorArguments,
    });
    if (minimalInputVerificationSuccess) {
        return;
    }
    console.log(`We tried verifying your contract ${contractInformation.contractName} without including any unrelated one, but it failed.
Trying again with the full solc input used to compile and deploy it.
This means that unrelated contracts may be displayed on Etherscan...
`);
    // If verifying with the minimal input failed, try again with the full compiler input
    const { success: fullCompilerInputVerificationSuccess, message: verificationMessage, } = await run(task_names_2.TASK_VERIFY_ETHERSCAN_ATTEMPT_VERIFICATION, {
        address,
        compilerInput: contractInformation.compilerInput,
        contractInformation,
        verificationInterface: etherscan,
        encodedConstructorArguments,
    });
    if (fullCompilerInputVerificationSuccess) {
        return;
    }
    throw new errors_1.ContractVerificationFailedError(verificationMessage, contractInformation.undetectableLibraries);
});
(0, config_1.subtask)(task_names_2.TASK_VERIFY_ETHERSCAN_RESOLVE_ARGUMENTS)
    .addOptionalParam("address")
    .addOptionalParam("constructorArgsParams", undefined, [], config_1.types.any)
    .addOptionalParam("constructorArgs", undefined, undefined, config_1.types.inputFile)
    .addOptionalParam("libraries", undefined, undefined, config_1.types.any)
    .addOptionalParam("contract")
    .addFlag("force")
    .setAction(async ({ address, constructorArgsParams, constructorArgs: constructorArgsModule, contract, libraries: librariesModule, force, }) => {
    if (address === undefined) {
        throw new errors_1.MissingAddressError();
    }
    const { isAddress } = await Promise.resolve().then(() => __importStar(require("@ethersproject/address")));
    if (!isAddress(address)) {
        throw new errors_1.InvalidAddressError(address);
    }
    if (contract !== undefined && !(0, contract_names_1.isFullyQualifiedName)(contract)) {
        throw new errors_1.InvalidContractNameError(contract);
    }
    const constructorArgs = await (0, utilities_1.resolveConstructorArguments)(constructorArgsParams, constructorArgsModule);
    let libraries;
    if (typeof librariesModule === "object") {
        libraries = librariesModule;
    }
    else {
        libraries = await (0, utilities_1.resolveLibraries)(librariesModule);
    }
    return {
        address,
        constructorArgs,
        libraries,
        contractFQN: contract,
        force,
    };
});
(0, config_1.subtask)(task_names_2.TASK_VERIFY_ETHERSCAN_GET_MINIMAL_INPUT)
    .addParam("sourceName")
    .setAction(async ({ sourceName }, { run }) => {
    const cloneDeep = require("lodash.clonedeep");
    const dependencyGraph = await run(task_names_1.TASK_COMPILE_SOLIDITY_GET_DEPENDENCY_GRAPH, { sourceNames: [sourceName] });
    const resolvedFiles = dependencyGraph
        .getResolvedFiles()
        .filter((resolvedFile) => resolvedFile.sourceName === sourceName);
    if (resolvedFiles.length !== 1) {
        throw new errors_1.UnexpectedNumberOfFilesError();
    }
    const compilationJob = await run(task_names_1.TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOB_FOR_FILE, {
        dependencyGraph,
        file: resolvedFiles[0],
    });
    const minimalInput = await run(task_names_1.TASK_COMPILE_SOLIDITY_GET_COMPILER_INPUT, {
        compilationJob,
    });
    return cloneDeep(minimalInput);
});
(0, config_1.subtask)(task_names_2.TASK_VERIFY_ETHERSCAN_ATTEMPT_VERIFICATION)
    .addParam("address")
    .addParam("compilerInput", undefined, undefined, config_1.types.any)
    .addParam("contractInformation", undefined, undefined, config_1.types.any)
    .addParam("verificationInterface", undefined, undefined, config_1.types.any)
    .addParam("encodedConstructorArguments")
    .setAction(async ({ address, compilerInput, contractInformation, verificationInterface, encodedConstructorArguments, }) => {
    // Ensure the linking information is present in the compiler input;
    compilerInput.settings.libraries = contractInformation.libraries;
    const contractFQN = `${contractInformation.sourceName}:${contractInformation.contractName}`;
    const { message: guid } = await verificationInterface.verify(address, JSON.stringify(compilerInput), contractFQN, `v${contractInformation.solcLongVersion}`, encodedConstructorArguments);
    console.log(`Successfully submitted source code for contract
${contractFQN} at ${address}
for verification on the block explorer. Waiting for verification result...
`);
    // Compilation is bound to take some time so there's no sense in requesting status immediately.
    await (0, utilities_1.sleep)(700);
    const verificationStatus = await verificationInterface.getVerificationStatus(guid);
    // Etherscan answers with already verified message only when checking returned guid
    if (verificationStatus.isAlreadyVerified()) {
        throw new errors_1.ContractAlreadyVerifiedError(contractFQN, address);
    }
    if (!(verificationStatus.isFailure() || verificationStatus.isSuccess())) {
        // Reaching this point shouldn't be possible unless the API is behaving in a new way.
        throw new errors_1.VerificationAPIUnexpectedMessageError(verificationStatus.message);
    }
    if (verificationStatus.isSuccess()) {
        const contractURL = verificationInterface.getContractUrl(address);
        console.log(`Successfully verified contract ${contractInformation.contractName} on the block explorer.
${contractURL}\n`);
    }
    return {
        success: verificationStatus.isSuccess(),
        message: verificationStatus.message,
    };
});
//# sourceMappingURL=etherscan.js.map