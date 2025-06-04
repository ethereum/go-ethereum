"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ContractAlreadyVerifiedError = exports.ContractVerificationFailedError = exports.NetworkRequestError = exports.VerificationAPIUnexpectedMessageError = exports.ABIArgumentOverflowError = exports.ABIArgumentTypeError = exports.ABIArgumentLengthError = exports.UnexpectedNumberOfFilesError = exports.LibraryAddressesMismatchError = exports.MissingLibrariesError = exports.LibraryMultipleMatchesError = exports.LibraryNotFoundError = exports.DuplicatedLibraryError = exports.InvalidLibraryAddressError = exports.DeployedBytecodeMultipleMatchesError = exports.DeployedBytecodeMismatchError = exports.BuildInfoCompilerVersionMismatchError = exports.BuildInfoNotFoundError = exports.ContractNotFoundError = exports.CompilerVersionsMismatchError = exports.DeployedBytecodeNotFoundError = exports.EtherscanVersionNotSupportedError = exports.ContractStatusPollingResponseNotOkError = exports.ContractStatusPollingInvalidStatusCodeError = exports.ContractVerificationMissingBytecodeError = exports.ContractVerificationInvalidStatusCodeError = exports.ChainConfigNotFoundError = exports.HardhatNetworkNotSupportedError = exports.ImportingModuleError = exports.InvalidLibrariesModuleError = exports.InvalidLibrariesError = exports.InvalidConstructorArgumentsModuleError = exports.ExclusiveConstructorArgumentsError = exports.InvalidConstructorArgumentsError = exports.MissingApiKeyError = exports.InvalidContractNameError = exports.InvalidAddressError = exports.MissingAddressError = exports.HardhatVerifyError = void 0;
const plugins_1 = require("hardhat/plugins");
const task_names_1 = require("./task-names");
class HardhatVerifyError extends plugins_1.NomicLabsHardhatPluginError {
    constructor(message, parent) {
        super("@nomicfoundation/hardhat-verify", message, parent);
        Object.setPrototypeOf(this, this.constructor.prototype);
    }
}
exports.HardhatVerifyError = HardhatVerifyError;
class MissingAddressError extends HardhatVerifyError {
    constructor() {
        super("You didn’t provide any address. Please re-run the 'verify' task with the address of the contract you want to verify.");
    }
}
exports.MissingAddressError = MissingAddressError;
class InvalidAddressError extends HardhatVerifyError {
    constructor(address) {
        super(`${address} is an invalid address.`);
    }
}
exports.InvalidAddressError = InvalidAddressError;
class InvalidContractNameError extends HardhatVerifyError {
    constructor(contractName) {
        super(`A valid fully qualified name was expected. Fully qualified names look like this: "contracts/AContract.sol:TheContract"
Instead, this name was received: ${contractName}`);
    }
}
exports.InvalidContractNameError = InvalidContractNameError;
class MissingApiKeyError extends HardhatVerifyError {
    constructor(network) {
        super(`You are trying to verify a contract in '${network}', but no API token was found for this network. Please provide one in your hardhat config. For example:

{
  ...
  etherscan: {
    apiKey: {
      ${network}: 'your API key'
    }
  }
}

See https://etherscan.io/apis`);
    }
}
exports.MissingApiKeyError = MissingApiKeyError;
class InvalidConstructorArgumentsError extends HardhatVerifyError {
    constructor() {
        super(`The constructorArguments parameter should be an array.
If your constructor has no arguments pass an empty array. E.g:

  await run("${task_names_1.TASK_VERIFY_VERIFY}", {
    <other args>,
    constructorArguments: []
  };`);
    }
}
exports.InvalidConstructorArgumentsError = InvalidConstructorArgumentsError;
class ExclusiveConstructorArgumentsError extends HardhatVerifyError {
    constructor() {
        super("The parameters constructorArgsParams and constructorArgsModule are exclusive. Please provide only one of them.");
    }
}
exports.ExclusiveConstructorArgumentsError = ExclusiveConstructorArgumentsError;
class InvalidConstructorArgumentsModuleError extends HardhatVerifyError {
    constructor(constructorArgsModulePath) {
        super(`The module ${constructorArgsModulePath} doesn't export a list. The module should look like this:

module.exports = [ arg1, arg2, ... ];`);
    }
}
exports.InvalidConstructorArgumentsModuleError = InvalidConstructorArgumentsModuleError;
class InvalidLibrariesError extends HardhatVerifyError {
    constructor() {
        super(`The libraries parameter should be a dictionary.
If your contract does not have undetectable libraries pass an empty object or omit the argument. E.g:

  await run("${task_names_1.TASK_VERIFY_VERIFY}", {
    <other args>,
    libraries: {}
  };`);
    }
}
exports.InvalidLibrariesError = InvalidLibrariesError;
class InvalidLibrariesModuleError extends HardhatVerifyError {
    constructor(librariesModulePath) {
        super(`The module ${librariesModulePath} doesn't export a dictionary. The module should look like this:

module.exports = { lib1: "0x...", lib2: "0x...", ... };`);
    }
}
exports.InvalidLibrariesModuleError = InvalidLibrariesModuleError;
class ImportingModuleError extends HardhatVerifyError {
    constructor(module, parent) {
        super(`Importing the module for the ${module} failed.
Reason: ${parent.message}`, parent);
    }
}
exports.ImportingModuleError = ImportingModuleError;
class HardhatNetworkNotSupportedError extends HardhatVerifyError {
    constructor() {
        super(`The selected network is "hardhat", which is not supported for contract verification.

If you intended to use a different network, ensure that you provide the --network parameter when running the command.

For example: npx hardhat verify --network <network-name>`);
    }
}
exports.HardhatNetworkNotSupportedError = HardhatNetworkNotSupportedError;
class ChainConfigNotFoundError extends HardhatVerifyError {
    constructor(chainId) {
        super(`Trying to verify a contract in a network with chain id ${chainId}, but the plugin doesn't recognize it as a supported chain.

You can manually add support for it by following these instructions: https://hardhat.org/verify-custom-networks

To see the list of supported networks, run this command:

  npx hardhat verify --list-networks`);
    }
}
exports.ChainConfigNotFoundError = ChainConfigNotFoundError;
class ContractVerificationInvalidStatusCodeError extends HardhatVerifyError {
    constructor(url, statusCode, responseText) {
        super(`Failed to send contract verification request.
Endpoint URL: ${url}
The HTTP server response is not ok. Status code: ${statusCode} Response text: ${responseText}`);
    }
}
exports.ContractVerificationInvalidStatusCodeError = ContractVerificationInvalidStatusCodeError;
class ContractVerificationMissingBytecodeError extends HardhatVerifyError {
    constructor(url, contractAddress) {
        super(`Failed to send contract verification request.
Endpoint URL: ${url}
Reason: The Etherscan API responded that the address ${contractAddress} does not have bytecode.
This can happen if the contract was recently deployed and this fact hasn't propagated to the backend yet.
Try waiting for a minute before verifying your contract. If you are invoking this from a script,
try to wait for five confirmations of your contract deployment transaction before running the verification subtask.`);
    }
}
exports.ContractVerificationMissingBytecodeError = ContractVerificationMissingBytecodeError;
class ContractStatusPollingInvalidStatusCodeError extends HardhatVerifyError {
    constructor(statusCode, responseText) {
        super(`The HTTP server response is not ok. Status code: ${statusCode} Response text: ${responseText}`);
    }
}
exports.ContractStatusPollingInvalidStatusCodeError = ContractStatusPollingInvalidStatusCodeError;
class ContractStatusPollingResponseNotOkError extends HardhatVerifyError {
    constructor(message) {
        super(`The Etherscan API responded with a failure status.
The verification may still succeed but should be checked manually.
Reason: ${message}`);
    }
}
exports.ContractStatusPollingResponseNotOkError = ContractStatusPollingResponseNotOkError;
class EtherscanVersionNotSupportedError extends HardhatVerifyError {
    constructor() {
        super(`Etherscan only supports compiler versions 0.4.11 and higher.
See https://etherscan.io/solcversions for more information.`);
    }
}
exports.EtherscanVersionNotSupportedError = EtherscanVersionNotSupportedError;
class DeployedBytecodeNotFoundError extends HardhatVerifyError {
    constructor(address, network) {
        super(`The address ${address} has no bytecode. Is the contract deployed to this network?
The selected network is ${network}.`);
    }
}
exports.DeployedBytecodeNotFoundError = DeployedBytecodeNotFoundError;
class CompilerVersionsMismatchError extends HardhatVerifyError {
    constructor(configCompilerVersions, inferredCompilerVersion, network) {
        const versionDetails = configCompilerVersions.length > 1
            ? `versions are: ${configCompilerVersions.join(", ")}`
            : `version is: ${configCompilerVersions[0]}`;
        super(`The contract you want to verify was compiled with solidity ${inferredCompilerVersion}, but your configured compiler ${versionDetails}.

Possible causes are:
- You are not in the same commit that was used to deploy the contract.
- Wrong compiler version selected in hardhat config.
- The given address is wrong.
- The selected network (${network}) is wrong.`);
    }
}
exports.CompilerVersionsMismatchError = CompilerVersionsMismatchError;
class ContractNotFoundError extends HardhatVerifyError {
    constructor(contractFQN) {
        super(`The contract ${contractFQN} is not present in your project.`);
    }
}
exports.ContractNotFoundError = ContractNotFoundError;
class BuildInfoNotFoundError extends HardhatVerifyError {
    constructor(contractFQN) {
        super(`The contract ${contractFQN} is present in your project, but we couldn't find its sources.
Please make sure that it has been compiled by Hardhat and that it is written in Solidity.`);
    }
}
exports.BuildInfoNotFoundError = BuildInfoNotFoundError;
class BuildInfoCompilerVersionMismatchError extends HardhatVerifyError {
    constructor(contractFQN, compilerVersion, isVersionRange, buildInfoCompilerVersion, network) {
        const versionDetails = isVersionRange
            ? `a solidity version in the range ${compilerVersion}`
            : `the solidity version ${compilerVersion}`;
        super(`The contract ${contractFQN} is being compiled with ${buildInfoCompilerVersion}.
However, the contract found in the address provided as argument has its bytecode marked with ${versionDetails}.

Possible causes are:
- Solidity compiler version settings were modified after the deployment was executed.
- The given address is wrong.
- The selected network (${network}) is wrong.`);
    }
}
exports.BuildInfoCompilerVersionMismatchError = BuildInfoCompilerVersionMismatchError;
class DeployedBytecodeMismatchError extends HardhatVerifyError {
    constructor(network, contractFQN) {
        const contractDetails = typeof contractFQN === "string"
            ? `the contract ${contractFQN}.`
            : `any of your local contracts.`;
        super(`The address provided as argument contains a contract, but its bytecode doesn't match ${contractDetails}

Possible causes are:
  - The artifact for that contract is outdated or missing. You can try compiling the project again with the --force flag before re-running the verification.
  - The contract's code changed after the deployment was executed. Sometimes this happens by changes in seemingly unrelated contracts.
  - The solidity compiler settings were modified after the deployment was executed (like the optimizer, target EVM, etc.)
  - The given address is wrong.
  - The selected network (${network}) is wrong.`);
    }
}
exports.DeployedBytecodeMismatchError = DeployedBytecodeMismatchError;
class DeployedBytecodeMultipleMatchesError extends HardhatVerifyError {
    constructor(fqnMatches) {
        super(`More than one contract was found to match the deployed bytecode.
Please use the contract parameter with one of the following contracts:
${fqnMatches.map((x) => `  * ${x}`).join("\n")}

For example:

hardhat verify --contract contracts/Example.sol:ExampleContract <other args>

If you are running the verify subtask from within Hardhat instead:

await run("${task_names_1.TASK_VERIFY_VERIFY}", {
<other args>,
contract: "contracts/Example.sol:ExampleContract"
};`);
    }
}
exports.DeployedBytecodeMultipleMatchesError = DeployedBytecodeMultipleMatchesError;
class InvalidLibraryAddressError extends HardhatVerifyError {
    constructor(contractName, libraryName, libraryAddress) {
        super(`You gave a link for the contract ${contractName} with the library ${libraryName}, but provided this invalid address: ${libraryAddress}`);
    }
}
exports.InvalidLibraryAddressError = InvalidLibraryAddressError;
class DuplicatedLibraryError extends HardhatVerifyError {
    constructor(libraryName, libraryFQN) {
        super(`The library names ${libraryName} and ${libraryFQN} refer to the same library and were given as two entries in the libraries dictionary.
Remove one of them and review your libraries dictionary before proceeding.`);
    }
}
exports.DuplicatedLibraryError = DuplicatedLibraryError;
class LibraryNotFoundError extends HardhatVerifyError {
    constructor(contractName, libraryName, allLibraries, detectableLibraries, undetectableLibraries) {
        const contractLibrariesDetails = `This contract uses the following external libraries:
${undetectableLibraries.map((x) => `  * ${x}`).join("\n")}
${detectableLibraries.map((x) => `  * ${x} (optional)`).join("\n")}
${detectableLibraries.length > 0
            ? "Libraries marked as optional don't need to be specified since their addresses are autodetected by the plugin."
            : ""}`;
        super(`You gave an address for the library ${libraryName} in the libraries dictionary, which is not one of the libraries of contract ${contractName}.
${allLibraries.length > 0
            ? contractLibrariesDetails
            : "This contract doesn't use any external libraries."}`);
    }
}
exports.LibraryNotFoundError = LibraryNotFoundError;
class LibraryMultipleMatchesError extends HardhatVerifyError {
    constructor(contractName, libraryName, fqnMatches) {
        super(`The library name ${libraryName} is ambiguous for the contract ${contractName}.
It may resolve to one of the following libraries:
${fqnMatches.map((x) => `  * ${x}`).join("\n")}

To fix this, choose one of these fully qualified library names and replace it in your libraries dictionary.`);
    }
}
exports.LibraryMultipleMatchesError = LibraryMultipleMatchesError;
class MissingLibrariesError extends HardhatVerifyError {
    constructor(contractName, allLibraries, mergedLibraries, undetectableLibraries) {
        const missingLibraries = allLibraries.filter((lib) => !mergedLibraries.some((mergedLib) => lib === mergedLib));
        super(`The contract ${contractName} has one or more library addresses that cannot be detected from deployed bytecode.
This can occur if the library is only called in the contract constructor. The missing libraries are:
${missingLibraries.map((x) => `  * ${x}`).join("\n")}

${missingLibraries.length === undetectableLibraries.length
            ? "Visit https://hardhat.org/hardhat-runner/plugins/nomicfoundation-hardhat-verify#libraries-with-undetectable-addresses to learn how to solve this."
            : "To solve this, you can add them to your --libraries dictionary with their corresponding addresses."}`);
    }
}
exports.MissingLibrariesError = MissingLibrariesError;
class LibraryAddressesMismatchError extends HardhatVerifyError {
    constructor(conflicts) {
        super(`The following detected library addresses are different from the ones provided:
${conflicts
            .map(({ library, inputAddress, detectedAddress }) => `  * ${library}
given address: ${inputAddress}
detected address: ${detectedAddress}`)
            .join("\n")}

You can either fix these addresses in your libraries dictionary or simply remove them to let the plugin autodetect them.`);
    }
}
exports.LibraryAddressesMismatchError = LibraryAddressesMismatchError;
class UnexpectedNumberOfFilesError extends HardhatVerifyError {
    constructor() {
        super("The plugin found an unexpected number of files for this contract. Please report this issue to the Hardhat team.");
    }
}
exports.UnexpectedNumberOfFilesError = UnexpectedNumberOfFilesError;
class ABIArgumentLengthError extends HardhatVerifyError {
    constructor(sourceName, contractName, error) {
        const { types: requiredArgs, values: providedArgs } = error.count;
        super(`The constructor for ${sourceName}:${contractName} has ${requiredArgs} parameters
but ${providedArgs} arguments were provided instead.`, error);
    }
}
exports.ABIArgumentLengthError = ABIArgumentLengthError;
class ABIArgumentTypeError extends HardhatVerifyError {
    constructor(error) {
        const { value: argValue, argument: argName, reason } = error;
        super(`Value ${argValue} cannot be encoded for the parameter ${argName}.
Encoder error reason: ${reason}`, error);
    }
}
exports.ABIArgumentTypeError = ABIArgumentTypeError;
class ABIArgumentOverflowError extends HardhatVerifyError {
    constructor(error) {
        const { value: argValue, fault: reason, operation } = error;
        super(`Value ${argValue} is not a safe integer and cannot be encoded.
Use a string instead of a plain number.
Encoder error reason: ${reason} fault in ${operation}`, error);
    }
}
exports.ABIArgumentOverflowError = ABIArgumentOverflowError;
/**
 * `VerificationAPIUnexpectedMessageError` is thrown when the block explorer API
 * does not behave as expected, such as when it returns an unexpected response message.
 */
class VerificationAPIUnexpectedMessageError extends HardhatVerifyError {
    constructor(message) {
        super(`The API responded with an unexpected message.
Contract verification may have succeeded and should be checked manually.
Message: ${message}`);
    }
}
exports.VerificationAPIUnexpectedMessageError = VerificationAPIUnexpectedMessageError;
class NetworkRequestError extends HardhatVerifyError {
    constructor(e) {
        super(`A network request failed. This is an error from the block explorer, not Hardhat. Error: ${e.message}`);
    }
}
exports.NetworkRequestError = NetworkRequestError;
class ContractVerificationFailedError extends HardhatVerifyError {
    constructor(message, undetectableLibraries) {
        super(`The contract verification failed.
Reason: ${message}
${undetectableLibraries.length > 0
            ? `
This contract makes use of libraries whose addresses are undetectable by the plugin.
Keep in mind that this verification failure may be due to passing in the wrong
address for one of these libraries:
${undetectableLibraries.map((x) => `  * ${x}`).join("\n")}`
            : ""}`);
    }
}
exports.ContractVerificationFailedError = ContractVerificationFailedError;
class ContractAlreadyVerifiedError extends HardhatVerifyError {
    constructor(contractFQN, contractAddress) {
        super(`The block explorer's API responded that the contract ${contractFQN} at ${contractAddress} is already verified.
This can happen if you used the '--force' flag. However, re-verification of contracts might not be supported
by the explorer (e.g., Etherscan), or the contract may have already been verified with a full match.`);
    }
}
exports.ContractAlreadyVerifiedError = ContractAlreadyVerifiedError;
//# sourceMappingURL=errors.js.map