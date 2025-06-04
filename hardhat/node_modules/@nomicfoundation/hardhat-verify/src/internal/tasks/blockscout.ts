import type { CompilerInput } from "hardhat/types";
import type { VerificationResponse, VerifyTaskArgs } from "../..";
import type {
  LibraryToAddress,
  ExtendedContractInformation,
} from "../solc/artifacts";

import { subtask, types } from "hardhat/config";
import { isFullyQualifiedName } from "hardhat/utils/contract-names";

import {
  CompilerVersionsMismatchError,
  ContractVerificationFailedError,
  InvalidAddressError,
  InvalidContractNameError,
  MissingAddressError,
} from "../errors";
import { Blockscout } from "../blockscout";
import { Bytecode } from "../solc/bytecode";
import {
  TASK_VERIFY_BLOCKSCOUT,
  TASK_VERIFY_BLOCKSCOUT_ATTEMPT_VERIFICATION,
  TASK_VERIFY_BLOCKSCOUT_RESOLVE_ARGUMENTS,
  TASK_VERIFY_ETHERSCAN_ATTEMPT_VERIFICATION,
  TASK_VERIFY_ETHERSCAN_GET_MINIMAL_INPUT,
  TASK_VERIFY_GET_CONTRACT_INFORMATION,
} from "../task-names";
import { getCompilerVersions, resolveLibraries } from "../utilities";

// parsed verification args
interface VerificationArgs {
  address: string;
  libraries: LibraryToAddress;
  contractFQN?: string;
  force: boolean;
}

interface AttemptVerificationArgs {
  address: string;
  compilerInput: CompilerInput;
  contractInformation: ExtendedContractInformation;
  verificationInterface: Blockscout;
}

/**
 * Main Blockscout verification subtask.
 *
 * Verifies a contract in Blockscout by coordinating various subtasks related
 * to contract verification.
 */
subtask(TASK_VERIFY_BLOCKSCOUT)
  .addParam("address")
  .addOptionalParam("libraries", undefined, undefined, types.any)
  .addOptionalParam("contract")
  .addFlag("force")
  .setAction(
    async (
      taskArgs: VerifyTaskArgs,
      { config: config, network: network, run }
    ) => {
      const { address, libraries, contractFQN, force }: VerificationArgs =
        await run(TASK_VERIFY_BLOCKSCOUT_RESOLVE_ARGUMENTS, taskArgs);

      const chainConfig = await Blockscout.getCurrentChainConfig(
        network.name,
        network.provider,
        config.blockscout.customChains
      );

      const blockscout = Blockscout.fromChainConfig(chainConfig);

      const isVerified = await blockscout.isVerified(address);
      if (!force && isVerified) {
        const contractURL = blockscout.getContractUrl(address);
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

      const matchingCompilerVersions =
        await deployedBytecode.getMatchingVersions(configCompilerVersions);
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

      // First, try to verify the contract using the minimal input
      const { success: minimalInputVerificationSuccess }: VerificationResponse =
        await run(TASK_VERIFY_BLOCKSCOUT_ATTEMPT_VERIFICATION, {
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
      const {
        success: fullCompilerInputVerificationSuccess,
        message: verificationMessage,
      }: VerificationResponse = await run(
        TASK_VERIFY_BLOCKSCOUT_ATTEMPT_VERIFICATION,
        {
          address,
          compilerInput: contractInformation.compilerInput,
          contractInformation,
          verificationInterface: blockscout,
        }
      );

      if (fullCompilerInputVerificationSuccess) {
        return;
      }

      throw new ContractVerificationFailedError(
        verificationMessage,
        contractInformation.undetectableLibraries
      );
    }
  );

subtask(TASK_VERIFY_BLOCKSCOUT_RESOLVE_ARGUMENTS)
  .addOptionalParam("address")
  .addOptionalParam("libraries", undefined, undefined, types.any)
  .addOptionalParam("contract")
  .addFlag("force")
  .setAction(
    async ({
      address,
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

      let libraries;
      if (typeof librariesModule === "object") {
        libraries = librariesModule;
      } else {
        libraries = await resolveLibraries(librariesModule);
      }

      return {
        address,
        libraries,
        contractFQN: contract,
        force,
      };
    }
  );

subtask(TASK_VERIFY_BLOCKSCOUT_ATTEMPT_VERIFICATION)
  .addParam("address")
  .addParam("compilerInput", undefined, undefined, types.any)
  .addParam("contractInformation", undefined, undefined, types.any)
  .addParam("verificationInterface", undefined, undefined, types.any)
  .setAction(
    async (
      {
        address,
        compilerInput,
        contractInformation,
        verificationInterface,
      }: AttemptVerificationArgs,
      { run }
    ): Promise<VerificationResponse> => {
      return run(TASK_VERIFY_ETHERSCAN_ATTEMPT_VERIFICATION, {
        address,
        compilerInput,
        contractInformation,
        verificationInterface,
        encodedConstructorArguments: "",
      });
    }
  );
