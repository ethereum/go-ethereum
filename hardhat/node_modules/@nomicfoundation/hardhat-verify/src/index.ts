import type {
  ContractInformation,
  ExtendedContractInformation,
  LibraryToAddress,
} from "./internal/solc/artifacts";
import type { Bytecode } from "./internal/solc/bytecode";

import picocolors from "picocolors";
import { extendConfig, subtask, task, types } from "hardhat/config";

import {
  TASK_VERIFY,
  TASK_VERIFY_GET_VERIFICATION_SUBTASKS,
  TASK_VERIFY_VERIFY,
  TASK_VERIFY_ETHERSCAN,
  TASK_VERIFY_PRINT_SUPPORTED_NETWORKS,
  TASK_VERIFY_SOURCIFY,
  TASK_VERIFY_SOURCIFY_DISABLED_WARNING,
  TASK_VERIFY_GET_CONTRACT_INFORMATION,
  TASK_VERIFY_BLOCKSCOUT,
} from "./internal/task-names";
import {
  etherscanConfigExtender,
  sourcifyConfigExtender,
  blockscoutConfigExtender,
} from "./internal/config";
import {
  InvalidConstructorArgumentsError,
  InvalidLibrariesError,
  HardhatVerifyError,
  ContractNotFoundError,
  BuildInfoNotFoundError,
  BuildInfoCompilerVersionMismatchError,
  DeployedBytecodeMismatchError,
} from "./internal/errors";
import {
  printSupportedNetworks,
  printVerificationErrors,
} from "./internal/utilities";
import {
  extractMatchingContractInformation,
  extractInferredContractInformation,
  getLibraryInformation,
} from "./internal/solc/artifacts";

import "./internal/type-extensions";
import "./internal/tasks/etherscan";
import "./internal/tasks/sourcify";
import "./internal/tasks/blockscout";

// Main task args
export interface VerifyTaskArgs {
  address?: string;
  constructorArgsParams: string[];
  constructorArgs?: string;
  libraries?: string;
  contract?: string;
  force: boolean;
  listNetworks: boolean;
}

// verify:verify subtask args
interface VerifySubtaskArgs {
  address?: string;
  constructorArguments: string[];
  libraries: LibraryToAddress;
  contract?: string;
  force?: boolean;
}

export interface VerificationResponse {
  success: boolean;
  message: string;
}

interface GetContractInformationArgs {
  contractFQN?: string;
  deployedBytecode: Bytecode;
  matchingCompilerVersions: string[];
  libraries: LibraryToAddress;
}

export interface VerificationSubtask {
  label: string;
  subtaskName: string;
}

extendConfig(etherscanConfigExtender);
extendConfig(sourcifyConfigExtender);
extendConfig(blockscoutConfigExtender);

/**
 * Main verification task.
 *
 * This is a meta-task that gets all the verification tasks and runs them.
 * It supports Etherscan and Sourcify.
 */
task(TASK_VERIFY, "Verifies a contract on Etherscan or Sourcify")
  .addOptionalPositionalParam("address", "Address of the contract to verify")
  .addOptionalVariadicPositionalParam(
    "constructorArgsParams",
    "Contract constructor arguments. Cannot be used if the --constructor-args option is provided",
    []
  )
  .addOptionalParam(
    "constructorArgs",
    "Path to a Javascript module that exports the constructor arguments",
    undefined,
    types.inputFile
  )
  .addOptionalParam(
    "libraries",
    "Path to a Javascript module that exports a dictionary of library addresses. " +
      "Use if there are undetectable library addresses in your contract. " +
      "Library addresses are undetectable if they are only used in the contract constructor",
    undefined,
    types.inputFile
  )
  .addOptionalParam(
    "contract",
    "Fully qualified name of the contract to verify. Skips automatic detection of the contract. " +
      "Use if the deployed bytecode matches more than one contract in your project"
  )
  .addFlag(
    "force",
    "Enforce contract verification even if the contract is already verified. " +
      "Use to re-verify partially verified contracts on Blockscout"
  )
  .addFlag("listNetworks", "Print the list of supported networks")
  .setAction(async (taskArgs: VerifyTaskArgs, { run }) => {
    if (taskArgs.listNetworks) {
      await run(TASK_VERIFY_PRINT_SUPPORTED_NETWORKS);
      return;
    }

    const verificationSubtasks: VerificationSubtask[] = await run(
      TASK_VERIFY_GET_VERIFICATION_SUBTASKS
    );

    const errors: Record<string, HardhatVerifyError> = {};
    for (const { label, subtaskName } of verificationSubtasks) {
      try {
        await run(subtaskName, taskArgs);
      } catch (error) {
        errors[label] = error as HardhatVerifyError;
      }
    }

    const hasErrors = Object.keys(errors).length > 0;
    if (hasErrors) {
      printVerificationErrors(errors);
      process.exit(1);
    }
  });

subtask(
  TASK_VERIFY_PRINT_SUPPORTED_NETWORKS,
  "Prints the supported networks list"
).setAction(async ({}, { config }) => {
  await printSupportedNetworks(config.etherscan.customChains);
});

subtask(
  TASK_VERIFY_GET_VERIFICATION_SUBTASKS,
  async (_, { config, userConfig }): Promise<VerificationSubtask[]> => {
    const verificationSubtasks: VerificationSubtask[] = [];

    if (config.etherscan.enabled) {
      verificationSubtasks.push({
        label: "Etherscan",
        subtaskName: TASK_VERIFY_ETHERSCAN,
      });
    }

    if (config.sourcify.enabled) {
      verificationSubtasks.push({
        label: "Sourcify",
        subtaskName: TASK_VERIFY_SOURCIFY,
      });
    } else if (userConfig.sourcify?.enabled === undefined) {
      verificationSubtasks.unshift({
        label: "Common",
        subtaskName: TASK_VERIFY_SOURCIFY_DISABLED_WARNING,
      });
    }

    if (config.blockscout.enabled) {
      verificationSubtasks.push({
        label: "Blockscout",
        subtaskName: TASK_VERIFY_BLOCKSCOUT,
      });
    }

    if (
      !config.etherscan.enabled &&
      !config.sourcify.enabled &&
      !config.blockscout.enabled
    ) {
      console.warn(
        picocolors.yellow(
          `[WARNING] No verification services are enabled. Please enable at least one verification service in your configuration.`
        )
      );
    }

    return verificationSubtasks;
  }
);

subtask(TASK_VERIFY_GET_CONTRACT_INFORMATION)
  .addParam("deployedBytecode", undefined, undefined, types.any)
  .addParam("matchingCompilerVersions", undefined, undefined, types.any)
  .addParam("libraries", undefined, undefined, types.any)
  .addOptionalParam("contractFQN")
  .setAction(
    async (
      {
        contractFQN,
        deployedBytecode,
        matchingCompilerVersions,
        libraries,
      }: GetContractInformationArgs,
      { network, artifacts }
    ): Promise<ExtendedContractInformation> => {
      let contractInformation: ContractInformation | null;

      if (contractFQN !== undefined) {
        const artifactExists = await artifacts.artifactExists(contractFQN);

        if (!artifactExists) {
          throw new ContractNotFoundError(contractFQN);
        }

        const buildInfo = await artifacts.getBuildInfo(contractFQN);
        if (buildInfo === undefined) {
          throw new BuildInfoNotFoundError(contractFQN);
        }

        if (
          !matchingCompilerVersions.includes(buildInfo.solcVersion) &&
          !deployedBytecode.isOvm()
        ) {
          throw new BuildInfoCompilerVersionMismatchError(
            contractFQN,
            deployedBytecode.getVersion(),
            deployedBytecode.hasVersionRange(),
            buildInfo.solcVersion,
            network.name
          );
        }

        contractInformation = extractMatchingContractInformation(
          contractFQN,
          buildInfo,
          deployedBytecode
        );

        if (contractInformation === null) {
          throw new DeployedBytecodeMismatchError(network.name, contractFQN);
        }
      } else {
        contractInformation = await extractInferredContractInformation(
          artifacts,
          network,
          matchingCompilerVersions,
          deployedBytecode
        );
      }

      // map contractInformation libraries
      const libraryInformation = await getLibraryInformation(
        contractInformation,
        libraries
      );

      return {
        ...contractInformation,
        ...libraryInformation,
      };
    }
  );

/**
 * This subtask is used to programmatically verify a contract on Etherscan or Sourcify.
 */
subtask(TASK_VERIFY_VERIFY)
  .addOptionalParam("address")
  .addOptionalParam("constructorArguments", undefined, [], types.any)
  .addOptionalParam("libraries", undefined, {}, types.any)
  .addOptionalParam("contract")
  .addFlag("force")
  .setAction(
    async (
      {
        address,
        constructorArguments,
        libraries,
        contract,
        force,
      }: VerifySubtaskArgs,
      { run, config }
    ) => {
      // This can only happen if the subtask is invoked from within Hardhat by a user script or another task.
      if (!Array.isArray(constructorArguments)) {
        throw new InvalidConstructorArgumentsError();
      }

      if (typeof libraries !== "object" || Array.isArray(libraries)) {
        throw new InvalidLibrariesError();
      }

      if (config.etherscan.enabled) {
        await run(TASK_VERIFY_ETHERSCAN, {
          address,
          constructorArgsParams: constructorArguments,
          libraries,
          contract,
          force,
        });
      }

      if (config.sourcify.enabled) {
        await run(TASK_VERIFY_SOURCIFY, {
          address,
          libraries,
          contract,
        });
      }
    }
  );
