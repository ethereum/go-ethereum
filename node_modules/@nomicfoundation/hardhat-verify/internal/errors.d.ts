import { NomicLabsHardhatPluginError } from "hardhat/plugins";
import { ABIArgumentLengthErrorType, ABIArgumentOverflowErrorType, ABIArgumentTypeErrorType } from "./abi-validation-extras";
export declare class HardhatVerifyError extends NomicLabsHardhatPluginError {
    constructor(message: string, parent?: Error);
}
export declare class MissingAddressError extends HardhatVerifyError {
    constructor();
}
export declare class InvalidAddressError extends HardhatVerifyError {
    constructor(address: string);
}
export declare class InvalidContractNameError extends HardhatVerifyError {
    constructor(contractName: string);
}
export declare class MissingApiKeyError extends HardhatVerifyError {
    constructor(network: string);
}
export declare class InvalidConstructorArgumentsError extends HardhatVerifyError {
    constructor();
}
export declare class ExclusiveConstructorArgumentsError extends HardhatVerifyError {
    constructor();
}
export declare class InvalidConstructorArgumentsModuleError extends HardhatVerifyError {
    constructor(constructorArgsModulePath: string);
}
export declare class InvalidLibrariesError extends HardhatVerifyError {
    constructor();
}
export declare class InvalidLibrariesModuleError extends HardhatVerifyError {
    constructor(librariesModulePath: string);
}
export declare class ImportingModuleError extends HardhatVerifyError {
    constructor(module: string, parent: Error);
}
export declare class HardhatNetworkNotSupportedError extends HardhatVerifyError {
    constructor();
}
export declare class ChainConfigNotFoundError extends HardhatVerifyError {
    constructor(chainId: number);
}
export declare class ContractVerificationInvalidStatusCodeError extends HardhatVerifyError {
    constructor(url: string, statusCode: number, responseText: string);
}
export declare class ContractVerificationMissingBytecodeError extends HardhatVerifyError {
    constructor(url: string, contractAddress: string);
}
export declare class ContractStatusPollingInvalidStatusCodeError extends HardhatVerifyError {
    constructor(statusCode: number, responseText: string);
}
export declare class ContractStatusPollingResponseNotOkError extends HardhatVerifyError {
    constructor(message: string);
}
export declare class EtherscanVersionNotSupportedError extends HardhatVerifyError {
    constructor();
}
export declare class DeployedBytecodeNotFoundError extends HardhatVerifyError {
    constructor(address: string, network: string);
}
export declare class CompilerVersionsMismatchError extends HardhatVerifyError {
    constructor(configCompilerVersions: string[], inferredCompilerVersion: string, network: string);
}
export declare class ContractNotFoundError extends HardhatVerifyError {
    constructor(contractFQN: string);
}
export declare class BuildInfoNotFoundError extends HardhatVerifyError {
    constructor(contractFQN: string);
}
export declare class BuildInfoCompilerVersionMismatchError extends HardhatVerifyError {
    constructor(contractFQN: string, compilerVersion: string, isVersionRange: boolean, buildInfoCompilerVersion: string, network: string);
}
export declare class DeployedBytecodeMismatchError extends HardhatVerifyError {
    constructor(network: string, contractFQN?: string);
}
export declare class DeployedBytecodeMultipleMatchesError extends HardhatVerifyError {
    constructor(fqnMatches: string[]);
}
export declare class InvalidLibraryAddressError extends HardhatVerifyError {
    constructor(contractName: string, libraryName: string, libraryAddress: string);
}
export declare class DuplicatedLibraryError extends HardhatVerifyError {
    constructor(libraryName: string, libraryFQN: string);
}
export declare class LibraryNotFoundError extends HardhatVerifyError {
    constructor(contractName: string, libraryName: string, allLibraries: string[], detectableLibraries: string[], undetectableLibraries: string[]);
}
export declare class LibraryMultipleMatchesError extends HardhatVerifyError {
    constructor(contractName: string, libraryName: string, fqnMatches: string[]);
}
export declare class MissingLibrariesError extends HardhatVerifyError {
    constructor(contractName: string, allLibraries: string[], mergedLibraries: string[], undetectableLibraries: string[]);
}
export declare class LibraryAddressesMismatchError extends HardhatVerifyError {
    constructor(conflicts: Array<{
        library: string;
        detectedAddress: string;
        inputAddress: string;
    }>);
}
export declare class UnexpectedNumberOfFilesError extends HardhatVerifyError {
    constructor();
}
export declare class ABIArgumentLengthError extends HardhatVerifyError {
    constructor(sourceName: string, contractName: string, error: ABIArgumentLengthErrorType);
}
export declare class ABIArgumentTypeError extends HardhatVerifyError {
    constructor(error: ABIArgumentTypeErrorType);
}
export declare class ABIArgumentOverflowError extends HardhatVerifyError {
    constructor(error: ABIArgumentOverflowErrorType);
}
/**
 * `VerificationAPIUnexpectedMessageError` is thrown when the block explorer API
 * does not behave as expected, such as when it returns an unexpected response message.
 */
export declare class VerificationAPIUnexpectedMessageError extends HardhatVerifyError {
    constructor(message: string);
}
export declare class NetworkRequestError extends HardhatVerifyError {
    constructor(e: Error);
}
export declare class ContractVerificationFailedError extends HardhatVerifyError {
    constructor(message: string, undetectableLibraries: string[]);
}
export declare class ContractAlreadyVerifiedError extends HardhatVerifyError {
    constructor(contractFQN: string, contractAddress: string);
}
//# sourceMappingURL=errors.d.ts.map