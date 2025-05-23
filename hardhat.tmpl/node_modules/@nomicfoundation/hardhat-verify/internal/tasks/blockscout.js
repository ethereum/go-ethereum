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
const contract_names_1 = require("hardhat/utils/contract-names");
const errors_1 = require("../errors");
const blockscout_1 = require("../blockscout");
const bytecode_1 = require("../solc/bytecode");
const task_names_1 = require("../task-names");
const utilities_1 = require("../utilities");
/**
 * Main Blockscout verification subtask.
 *
 * Verifies a contract in Blockscout by coordinating various subtasks related
 * to contract verification.
 */
(0, config_1.subtask)(task_names_1.TASK_VERIFY_BLOCKSCOUT)
    .addParam("address")
    .addOptionalParam("libraries", undefined, undefined, config_1.types.any)
    .addOptionalParam("contract")
    .addFlag("force")
    .setAction(async (taskArgs, { config: config, network: network, run }) => {
    const { address, libraries, contractFQN, force } = await run(task_names_1.TASK_VERIFY_BLOCKSCOUT_RESOLVE_ARGUMENTS, taskArgs);
    const chainConfig = await blockscout_1.Blockscout.getCurrentChainConfig(network.name, network.provider, config.blockscout.customChains);
    const blockscout = blockscout_1.Blockscout.fromChainConfig(chainConfig);
    const isVerified = await blockscout.isVerified(address);
    if (!force && isVerified) {
        const contractURL = blockscout.getContractUrl(address);
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
    const contractInformation = await run(task_names_1.TASK_VERIFY_GET_CONTRACT_INFORMATION, {
        contractFQN,
        deployedBytecode,
        matchingCompilerVersions,
        libraries,
    });
    const minimalInput = await run(task_names_1.TASK_VERIFY_ETHERSCAN_GET_MINIMAL_INPUT, {
        sourceName: contractInformation.sourceName,
    });
    // First, try to verify the contract using the minimal input
    const { success: minimalInputVerificationSuccess } = await run(task_names_1.TASK_VERIFY_BLOCKSCOUT_ATTEMPT_VERIFICATION, {
        address,
        compilerInput: minimalInput,
        contractInformation,
        verificationInterface: blockscout,
    });
    if (minimalInputVerificationSuccess) {
        return;
    }
    console.log(`We tried verifying your contract ${contractInformation.contractName} without including any unrelated one, but it failed.
Trying again with the full solc input used to compile and deploy it.
This means that unrelated contracts may be displayed on Blockscout...
`);
    // If verifying with the minimal input failed, try again with the full compiler input
    const { success: fullCompilerInputVerificationSuccess, message: verificationMessage, } = await run(task_names_1.TASK_VERIFY_BLOCKSCOUT_ATTEMPT_VERIFICATION, {
        address,
        compilerInput: contractInformation.compilerInput,
        contractInformation,
        verificationInterface: blockscout,
    });
    if (fullCompilerInputVerificationSuccess) {
        return;
    }
    throw new errors_1.ContractVerificationFailedError(verificationMessage, contractInformation.undetectableLibraries);
});
(0, config_1.subtask)(task_names_1.TASK_VERIFY_BLOCKSCOUT_RESOLVE_ARGUMENTS)
    .addOptionalParam("address")
    .addOptionalParam("libraries", undefined, undefined, config_1.types.any)
    .addOptionalParam("contract")
    .addFlag("force")
    .setAction(async ({ address, contract, libraries: librariesModule, force, }) => {
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
    let libraries;
    if (typeof librariesModule === "object") {
        libraries = librariesModule;
    }
    else {
        libraries = await (0, utilities_1.resolveLibraries)(librariesModule);
    }
    return {
        address,
        libraries,
        contractFQN: contract,
        force,
    };
});
(0, config_1.subtask)(task_names_1.TASK_VERIFY_BLOCKSCOUT_ATTEMPT_VERIFICATION)
    .addParam("address")
    .addParam("compilerInput", undefined, undefined, config_1.types.any)
    .addParam("contractInformation", undefined, undefined, config_1.types.any)
    .addParam("verificationInterface", undefined, undefined, config_1.types.any)
    .setAction(async ({ address, compilerInput, contractInformation, verificationInterface, }, { run }) => {
    return run(task_names_1.TASK_VERIFY_ETHERSCAN_ATTEMPT_VERIFICATION, {
        address,
        compilerInput,
        contractInformation,
        verificationInterface,
        encodedConstructorArguments: "",
    });
});
//# sourceMappingURL=blockscout.js.map