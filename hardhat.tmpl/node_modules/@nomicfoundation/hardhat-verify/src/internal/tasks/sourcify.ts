import type { VerificationResponse, VerifyTaskArgs } from "../..";
import type {
  ExtendedContractInformation,
  LibraryToAddress,
} from "../solc/artifacts";

import picocolors from "picocolors";
import { subtask, types } from "hardhat/config";
import { isFullyQualifiedName } from "hardhat/utils/contract-names";
import { HARDHAT_NETWORK_NAME } from "hardhat/plugins";

import { Sourcify } from "../sourcify";
import {
  CompilerVersionsMismatchError,
  ContractVerificationFailedError,
  HardhatNetworkNotSupportedError,
  HardhatVerifyError,
  InvalidAddressError,
  InvalidContractNameError,
  MissingAddressError,
} from "../errors";
import {
  TASK_VERIFY_SOURCIFY,
  TASK_VERIFY_SOURCIFY_RESOLVE_ARGUMENTS,
  TASK_VERIFY_GET_CONTRACT_INFORMATION,
  TASK_VERIFY_SOURCIFY_ATTEMPT_VERIFICATION,
  TASK_VERIFY_SOURCIFY_DISABLED_WARNING,
} from "../task-names";
import { getCompilerVersions, resolveLibraries } from "../utilities";
import { Bytecode } from "../solc/bytecode";

// parsed verification args
interface VerificationArgs {
  address: string;
  libraries: LibraryToAddress;
  contractFQN?: string;
}

interface AttemptVerificationArgs {
  address: string;
  verificationInterface: Sourcify;
  contractInformation: ExtendedContractInformation;
}

/**
 * Main Sourcify verification subtask.
 *
 * Verifies a contract in Sourcify by coordinating various subtasks related
 * to contract verification.
 */
subtask(TASK_VERIFY_SOURCIFY)
  .addParam("address")
  .addOptionalParam("contract")
  .addOptionalParam("libraries", undefined, undefined, types.any)
  .setAction(async (taskArgs: VerifyTaskArgs, { config, network, run }) => {
    const { address, libraries, contractFQN }: VerificationArgs = await run(
      TASK_VERIFY_SOURCIFY_RESOLVE_ARGUMENTS,
      taskArgs
    );

    if (network.name === HARDHAT_NETWORK_NAME) {
      throw new HardhatNetworkNotSupportedError();
    }

    const currentChainId = parseInt(
      await network.provider.send("eth_chainId"),
      16
    );

    const { apiUrl, browserUrl } = config.sourcify;

    if (apiUrl === undefined) {
      throw new HardhatVerifyError("Sourcify `apiUrl` is not defined");
    }

    if (browserUrl === undefined) {
      throw new HardhatVerifyError("Sourcify `browserUrl` is not defined");
    }

    const sourcify = new Sourcify(currentChainId, apiUrl, browserUrl);

    const status = await sourcify.isVerified(address);
    if (status !== false) {
      const contractURL = sourcify.getContractUrl(address, status);
      console.log(`The contract ${address} has already been verified on Sourcify.
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

    const {
      success: verificationSuccess,
      message: verificationMessage,
    }: VerificationResponse = await run(
      TASK_VERIFY_SOURCIFY_ATTEMPT_VERIFICATION,
      {
        address,
        verificationInterface: sourcify,
        contractInformation,
      }
    );

    if (verificationSuccess) {
      return;
    }

    throw new ContractVerificationFailedError(
      verificationMessage,
      contractInformation.undetectableLibraries
    );
  });

subtask(TASK_VERIFY_SOURCIFY_RESOLVE_ARGUMENTS)
  .addOptionalParam("address")
  .addOptionalParam("contract")
  .addOptionalParam("libraries", undefined, undefined, types.any)
  .setAction(
    async ({
      address,
      contract,
      libraries: librariesModule,
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
      };
    }
  );

subtask(TASK_VERIFY_SOURCIFY_ATTEMPT_VERIFICATION)
  .addParam("address")
  .addParam("contractInformation", undefined, undefined, types.any)
  .addParam("verificationInterface", undefined, undefined, types.any)
  .setAction(
    async ({
      address,
      verificationInterface,
      contractInformation,
    }: AttemptVerificationArgs): Promise<VerificationResponse> => {
      const { sourceName, contractName, contractOutput, compilerInput } =
        contractInformation;

      const librarySourcesToContent = Object.keys(
        contractInformation.libraries
      ).reduce((acc: Record<string, string>, libSourceName) => {
        const libContent = compilerInput.sources[libSourceName].content;
        acc[libSourceName] = libContent;
        return acc;
      }, {});

      const response = await verificationInterface.verify(address, {
        "metadata.json": (contractOutput as any).metadata,
        [sourceName]: compilerInput.sources[sourceName].content,
        ...librarySourcesToContent,
      });

      if (response.isOk()) {
        const contractURL = verificationInterface.getContractUrl(
          address,
          response.status
        );
        console.log(`Successfully verified contract ${contractName} on Sourcify.
${contractURL}
`);
      }

      return {
        success: response.isSuccess(),
        message: "Contract successfully verified on Sourcify",
      };
    }
  );

subtask(TASK_VERIFY_SOURCIFY_DISABLED_WARNING, async () => {
  console.info(
    picocolors.cyan(
      `[INFO] Sourcify Verification Skipped: Sourcify verification is currently disabled. To enable it, add the following entry to your Hardhat configuration:

sourcify: {
  enabled: true
}

Or set 'enabled' to false to hide this message.

For more information, visit https://hardhat.org/hardhat-runner/plugins/nomicfoundation-hardhat-verify#verifying-on-sourcify`
    )
  );
});
