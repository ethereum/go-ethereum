import type { Artifacts, BuildInfo, CompilerInput, CompilerOutputBytecode, CompilerOutputContract, Network } from "hardhat/types";
import { Bytecode } from "./bytecode";
export interface ContractInformation {
    compilerInput: CompilerInput;
    solcLongVersion: string;
    sourceName: string;
    contractName: string;
    contractOutput: CompilerOutputContract;
    deployedBytecode: string;
}
interface LibraryInformation {
    libraries: SourceToLibraryToAddress;
    undetectableLibraries: string[];
}
export type ExtendedContractInformation = ContractInformation & LibraryInformation;
export type LibraryToAddress = Record<string, string>;
export type SourceToLibraryToAddress = Record<string, LibraryToAddress>;
export interface ByteOffset {
    start: number;
    length: number;
}
export declare function getLibraryOffsets(linkReferences?: CompilerOutputBytecode["linkReferences"]): ByteOffset[];
export declare function getImmutableOffsets(immutableReferences?: CompilerOutputBytecode["immutableReferences"]): ByteOffset[];
/**
 * To normalize a library object we need to take into account its call
 * protection mechanism.
 * See https://solidity.readthedocs.io/en/latest/contracts.html#call-protection-for-libraries
 */
export declare function getCallProtectionOffsets(bytecode: string, referenceBytecode: string): ByteOffset[];
/**
 * Given a contract's fully qualified name, obtains the corresponding contract
 * information from the build-info by comparing the provided bytecode with the
 * deployed bytecode. If the bytecodes match, the function returns the contract
 * information. Otherwise, it returns null.
 */
export declare function extractMatchingContractInformation(contractFQN: string, buildInfo: BuildInfo, bytecode: Bytecode): ContractInformation | null;
/**
 * Searches through the artifacts for a contract that matches the given
 * deployed bytecode. If it finds a match, the function returns the contract
 * information.
 */
export declare function extractInferredContractInformation(artifacts: Artifacts, network: Network, matchingCompilerVersions: string[], bytecode: Bytecode): Promise<ContractInformation>;
/**
 * Retrieves the libraries from the contract information and combines them
 * with the libraries provided by the user. Returns a list containing all
 * the libraries required by the contract. Additionally, it returns a list of
 * undetectable libraries for debugging purposes.
 */
export declare function getLibraryInformation(contractInformation: ContractInformation, userLibraries: LibraryToAddress): Promise<LibraryInformation>;
export {};
//# sourceMappingURL=artifacts.d.ts.map