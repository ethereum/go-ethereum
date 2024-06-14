import type LodashCloneDeepT from "lodash.clonedeep";
import type {
  CompilerInput,
  DependencyGraph,
  CompilationJob,
} from "hardhat/types";
import type { VerificationResponse, VerifyTaskArgs } from "../..";
import type {
  LibraryToAddress,
  ExtendedContractInformation,
} from "../solc/artifacts";

import { subtask, types } from "hardhat/config";
import {
  TASK_COMPILE_SOLIDITY_GET_DEPENDENCY_GRAPH,
  TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOB_FOR_FILE,
  TASK_COMPILE_SOLIDITY_GET_COMPILER_INPUT,
} from "hardhat/builtin-tasks/task-names";
import { isFullyQualifiedName } from "hardhat/utils/contract-names";

import {
  CompilerVersionsMismatchError,
  ContractVerificationFailedError,
  MissingAddressError,
  InvalidAddressError,
  InvalidContractNameError,
  UnexpectedNumberOfFilesError,
  VerificationAPIUnexpectedMessageError,
  ContractAlreadyVerifiedError,
} from "../errors";
import { Etherscan } from "../etherscan";
import { Bytecode } from "../solc/bytecode";
import {
  TASK_VERIFY_ETHERSCAN,
  TASK_VERIFY_ETHERSCAN_RESOLVE_ARGUMENTS,
  TASK_VERIFY_ETHERSCAN_GET_MINIMAL_INPUT,
  TASK_VERIFY_ETHERSCAN_ATTEMPT_VERIFICATION,
  TASK_VERIFY_GET_CONTRACT_INFORMATION,
} from "../task-names";
import {
  getCompilerVersions,
  encodeArguments,
  resolveConstructorArguments,
  resolveLibraries,
  sleep,
} from "../utilities";

// parsed verification args
interface VerificationArgs {
  address: string;
  constructorArgs: string[];
  libraries: LibraryToAddress;
  contractFQN?: string;
  force: boolean;
}

interface GetMinimalInputArgs {
  sourceName: string;
}

interface AttemptVerificationArgs {
  address: string;
  compilerInput: CompilerInput;
  contractInformation: ExtendedContractInformation;
  verificationInterface: Etherscan;
  encodedConstructorArguments: string;
}

/**
 * Main Etherscan verification subtask.
 *
 * Verifies a contract in Etherscan by coordinating various subtasks related
 * to contract verification.
 */
subtask(TASK_VERIFY_ETHERSCAN)
  .addParam("address")
  .addOptionalParam("constructorArgsParams", undefined, undefined, types.any)
  .addOptionalParam("constructorArgs")
  .addOptionalParam("libraries", undefined, undefined, types.any)
  .addOptionalParam("contract")
  .addFlag("force")
  .setAction(async (taskArgs: VerifyTaskArgs, { config, network, run }) => {
    const {
      address,
      constructorArgs,
      libraries,
      contractFQN,
      force,
    }: VerificationArgs = await run(
      TASK_VERIFY_ETHERSCAN_RESOLVE_ARGUMENTS,
      taskArgs
    );

    const chainConfig = await Etherscan.getCurrentChainConfig(
      network.name,
      network.provider,
      config.etherscan.customChains
    );

    const etherscan = Etherscan.fromChainConfig(
      config.etherscan.apiKey,
      chainConfig
    );

    const isVerified = await etherscan.isVerified(address);
    if (!force && isVerified) {
      const contractURL = etherscan.getContractUrl(address);
      console.log(`The contract ${address} has already been verified on the block explorer. If you're trying to verify a partially verified contract, please use the --force flag.
${contractURL}
`);
      return;
    }

    const configCompilerVersions = await getCompilerVersions(config.solidity);

    const deployedBytecode = await Bytecode.getDeployedContractBytecode(
      address,
      network.provider,
      network.name
    );

    const matchingCompilerVersions = await deployedBytecode.getMatchingVersions(
      configCompilerVersions
    );
    // don't error if the bytecode appears to be OVM bytecode, because we can't infer a specific OVM solc version from the bytecode
    if (matchingCompilerVersions.length === 0 && !deployedBytecode.isOvm()) {
      throw new CompilerVersionsMismatchError(
        configCompilerVersions,
        deployedBytecode.getVersion(),
        network.name
      );
    }

    const contractInformation: ExtendedContractInformation = await run(
      TASK_VERIFY_GET_CONTRACT_INFORMATION,
      {
        contractFQN,
        deployedBytecode,
        matchingCompilerVersions,
        libraries,
      }
    );

    const minimalInput: CompilerInput = await run(
      TASK_VERIFY_ETHERSCAN_GET_MINIMAL_INPUT,
      {
        sourceName: contractInformation.sourceName,
      }
    );

    const encodedConstructorArguments = await encodeArguments(
      contractInformation.contractOutput.abi,
      contractInformation.sourceName,
      contractInformation.contractName,
      constructorArgs
    );

    // First, try to verify the contract using the minimal input
    const { success: minimalInputVerificationSuccess }: VerificationResponse =
      await run(TASK_VERIFY_ETHERSCAN_ATTEMPT_VERIFICATION, {
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
    const {
      success: fullCompilerInputVerificationSuccess,
      message: verificationMessage,
    }: VerificationResponse = await run(
      TASK_VERIFY_ETHERSCAN_ATTEMPT_VERIFICATION,
      {
        address,
        compilerInput: contractInformation.compilerInput,
        contractInformation,
        verificationInterface: etherscan,
        encodedConstructorArguments,
      }
    );

    if (fullCompilerInputVerificationSuccess) {
      return;
    }

    throw new ContractVerificationFailedError(
      verificationMessage,
      contractInformation.undetectableLibraries
    );
  });

subtask(TASK_VERIFY_ETHERSCAN_RESOLVE_ARGUMENTS)
  .addOptionalParam("address")
  .addOptionalParam("constructorArgsParams", undefined, [], types.any)
  .addOptionalParam("constructorArgs", undefined, undefined, types.inputFile)
  .addOptionalParam("libraries", undefined, undefined, types.any)
  .addOptionalParam("contract")
  .addFlag("force")
  .setAction(
    async ({
      address,
      constructorArgsParams,
      constructorArgs: constructorArgsModule,
      contract,
      libraries: librariesModule,
      force,
    }: VerifyTaskArgs): Promise<VerificationArgs> => {
      if (address === undefined) {
        throw new MissingAddressError();
      }

      const { isAddress } = await import("@ethersproject/address");
      if (!isAddress(address)) {
        throw new InvalidAddressError(address);
      }

      if (contract !== undefined && !isFullyQualifiedName(contract)) {
        throw new InvalidContractNameError(contract);
      }

      const constructorArgs = await resolveConstructorArguments(
        constructorArgsParams,
        constructorArgsModule
      );

      let libraries;
      if (typeof librariesModule === "object") {
        libraries = librariesModule;
      } else {
        libraries = await resolveLibraries(librariesModule);
      }

      return {
        address,
        constructorArgs,
        libraries,
        contractFQN: contract,
        force,
      };
    }
  );

subtask(TASK_VERIFY_ETHERSCAN_GET_MINIMAL_INPUT)
  .addParam("sourceName")
  .setAction(async ({ sourceName }: GetMinimalInputArgs, { run }) => {
    const cloneDeep = require("lodash.clonedeep") as typeof LodashCloneDeepT;
    const dependencyGraph: DependencyGraph = await run(
      TASK_COMPILE_SOLIDITY_GET_DEPENDENCY_GRAPH,
      { sourceNames: [sourceName] }
    );

    const resolvedFiles = dependencyGraph
      .getResolvedFiles()
      .filter((resolvedFile) => resolvedFile.sourceName === sourceName);

    if (resolvedFiles.length !== 1) {
      throw new UnexpectedNumberOfFilesError();
    }

    const compilationJob: CompilationJob = await run(
      TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOB_FOR_FILE,
      {
        dependencyGraph,
        file: resolvedFiles[0],
      }
    );

    const minimalInput: CompilerInput = await run(
      TASK_COMPILE_SOLIDITY_GET_COMPILER_INPUT,
      {
        compilationJob,
      }
    );

    return cloneDeep(minimalInput);
  });

subtask(TASK_VERIFY_ETHERSCAN_ATTEMPT_VERIFICATION)
  .addParam("address")
  .addParam("compilerInput", undefined, undefined, types.any)
  .addParam("contractInformation", undefined, undefined, types.any)
  .addParam("verificationInterface", undefined, undefined, types.any)
  .addParam("encodedConstructorArguments")
  .setAction(
    async ({
      address,
      compilerInput,
      contractInformation,
      verificationInterface,
      encodedConstructorArguments,
    }: AttemptVerificationArgs): Promise<VerificationResponse> => {
      // Ensure the linking information is present in the compiler input;
      compilerInput.settings.libraries = contractInformation.libraries;

      const contractFQN = `${contractInformation.sourceName}:${contractInformation.contractName}`;
      const { message: guid } = await verificationInterface.verify(
        address,
        JSON.stringify(compilerInput),
        contractFQN,
        `v${contractInformation.solcLongVersion}`,
        encodedConstructorArguments
      );

      console.log(`Successfully submitted source code for contract
${contractFQN} at ${address}
for verification on the block explorer. Waiting for verification result...
`);

      // Compilation is bound to take some time so there's no sense in requesting status immediately.
      await sleep(700);
      const verificationStatus =
        await verificationInterface.getVerificationStatus(guid);

      // Etherscan answers with already verified message only when checking returned guid
      if (verificationStatus.isAlreadyVerified()) {
        throw new ContractAlreadyVerifiedError(contractFQN, address);
      }

      if (!(verificationStatus.isFailure() || verificationStatus.isSuccess())) {
        // Reaching this point shouldn't be possible unless the API is behaving in a new way.
        throw new VerificationAPIUnexpectedMessageError(
          verificationStatus.message
        );
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
    }
  );
