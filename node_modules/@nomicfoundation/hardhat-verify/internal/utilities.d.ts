import type { JsonFragment } from "@ethersproject/abi";
import type { SolidityConfig } from "hardhat/types";
import type { ChainConfig } from "../types";
import { HardhatVerifyError } from "./errors";
import { LibraryToAddress } from "./solc/artifacts";
export declare function sleep(ms: number): Promise<void>;
/**
 * Prints a table of networks supported by hardhat-verify, including both
 * built-in and custom networks.
 */
export declare function printSupportedNetworks(customChains: ChainConfig[]): Promise<void>;
/**
 * Prints verification errors to the console.
 * @param errors - An object containing verification errors, where the keys
 * are the names of verification subtasks and the values are HardhatVerifyError
 * objects describing the specific errors.
 * @remarks This function formats and logs the verification errors to the
 * console with a red color using chalk. Each error is displayed along with the
 * name of the verification provider it belongs to.
 * @example
 * const errors: Record<string, HardhatVerifyError> = {
 *   verify:etherscan: { message: 'Error message for Etherscan' },
 *   verify:sourcify: { message: 'Error message for Sourcify' },
 *   // Add more errors here...
 * };
 * printVerificationErrors(errors);
 * // Output:
 * // hardhat-verify found one or more errors during the verification process:
 * //
 * // Etherscan:
 * // Error message for Etherscan
 * //
 * // Sourcify:
 * // Error message for Sourcify
 * //
 * // ... (more errors if present)
 */
export declare function printVerificationErrors(errors: Record<string, HardhatVerifyError>): void;
/**
 * Returns the list of constructor arguments from the constructorArgsModule
 * or the constructorArgsParams if the first is not defined.
 */
export declare function resolveConstructorArguments(constructorArgsParams: string[], constructorArgsModule?: string): Promise<string[]>;
/**
 * Returns a dictionary of library addresses from the librariesModule or
 * an empty object if not defined.
 */
export declare function resolveLibraries(librariesModule?: string): Promise<LibraryToAddress>;
/**
 * Retrieves the list of Solidity compiler versions for a given Solidity
 * configuration.
 * It checks that the versions are supported by Etherscan, and throws an
 * error if any are not.
 */
export declare function getCompilerVersions({ compilers, overrides, }: SolidityConfig): Promise<string[]>;
/**
 * Encodes the constructor arguments for a given contract.
 */
export declare function encodeArguments(abi: JsonFragment[], sourceName: string, contractName: string, constructorArguments: any[]): Promise<string>;
export interface ValidationResponse {
    isPending(): void;
    isFailure(): void;
    isSuccess(): void;
    isOk(): void;
}
//# sourceMappingURL=utilities.d.ts.map