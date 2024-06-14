import type { EthereumProvider } from "hardhat/types";
import type { ChainConfig, ApiKey } from "../types";
import type { EtherscanVerifyResponse } from "./etherscan.types";
import { ValidationResponse } from "./utilities";
/**
 * Etherscan verification provider for verifying smart contracts.
 * It should work with other verification providers as long as the interface
 * is compatible with Etherscan's.
 */
export declare class Etherscan {
    apiKey: string;
    apiUrl: string;
    browserUrl: string;
    /**
     * Create a new instance of the Etherscan verification provider.
     * @param apiKey - The Etherscan API key.
     * @param apiUrl - The Etherscan API URL, e.g. https://api.etherscan.io/api.
     * @param browserUrl - The Etherscan browser URL, e.g. https://etherscan.io.
     */
    constructor(apiKey: string, apiUrl: string, browserUrl: string);
    static getCurrentChainConfig(networkName: string, ethereumProvider: EthereumProvider, customChains: ChainConfig[]): Promise<ChainConfig>;
    static fromChainConfig(apiKey: ApiKey | undefined, chainConfig: ChainConfig): Etherscan;
    /**
     * Check if a smart contract is verified on Etherscan.
     * @link https://docs.etherscan.io/api-endpoints/contracts#get-contract-source-code-for-verified-contract-source-codes
     * @param address - The address of the smart contract.
     * @returns True if the contract is verified, false otherwise.
     * @throws {NetworkRequestError} if there is an error on the request.
     * @throws {ContractVerificationInvalidStatusCodeError} if the API returns an invalid status code.
     */
    isVerified(address: string): Promise<boolean>;
    /**
     * Verify a smart contract on Etherscan.
     * @link https://docs.etherscan.io/api-endpoints/contracts#verify-source-code
     * @param contractAddress - The address of the smart contract to verify.
     * @param sourceCode - The source code of the smart contract.
     * @param contractName - The name of the smart contract, e.g. "contracts/Sample.sol:MyContract"
     * @param compilerVersion - The version of the Solidity compiler used, e.g. `v0.8.19+commit.7dd6d404`
     * @param constructorArguments - The encoded constructor arguments of the smart contract.
     * @returns A promise that resolves to an `EtherscanResponse` object.
     * @throws {NetworkRequestError} if there is an error on the request.
     * @throws {ContractVerificationInvalidStatusCodeError} if the API returns an invalid status code.
     * @throws {ContractVerificationMissingBytecodeError} if the bytecode is not found on the block explorer.
     * @throws {ContractAlreadyVerifiedError} if the contract is already verified.
     * @throws {HardhatVerifyError} if the response status is not OK.
     */
    verify(contractAddress: string, sourceCode: string, contractName: string, compilerVersion: string, constructorArguments: string): Promise<EtherscanResponse>;
    /**
     * Get the verification status of a smart contract from Etherscan.
     * This method performs polling of the verification status if it's pending.
     * @link https://docs.etherscan.io/api-endpoints/contracts#check-source-code-verification-submission-status
     * @param guid - The verification GUID to check.
     * @returns A promise that resolves to an `EtherscanResponse` object.
     * @throws {NetworkRequestError} if there is an error on the request.
     * @throws {ContractStatusPollingInvalidStatusCodeError} if the API returns an invalid status code.
     * @throws {ContractStatusPollingResponseNotOkError} if the response status is not OK.
     */
    getVerificationStatus(guid: string): Promise<EtherscanResponse>;
    /**
     * Get the Etherscan URL for viewing a contract's details.
     * @param address - The address of the smart contract.
     * @returns The URL to view the contract on Etherscan's website.
     */
    getContractUrl(address: string): string;
}
declare class EtherscanResponse implements ValidationResponse {
    readonly status: number;
    readonly message: string;
    constructor(response: EtherscanVerifyResponse);
    isPending(): boolean;
    isFailure(): boolean;
    isSuccess(): boolean;
    isBytecodeMissingInNetworkError(): boolean;
    isAlreadyVerified(): boolean;
    isOk(): boolean;
}
export {};
//# sourceMappingURL=etherscan.d.ts.map