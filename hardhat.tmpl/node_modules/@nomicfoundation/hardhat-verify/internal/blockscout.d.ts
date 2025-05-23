import type { EthereumProvider } from "hardhat/types";
import type { ChainConfig } from "../types";
import { ValidationResponse } from "./utilities";
/**
 * Blockscout verification provider for verifying smart contracts.
 */
export declare class Blockscout {
    apiUrl: string;
    browserUrl: string;
    private _etherscan;
    /**
     * Create a new instance of the Blockscout verification provider.
     * @param apiUrl - The Blockscout API URL, e.g. https://eth.blockscout.com/api.
     * @param browserUrl - The Blockscout browser URL, e.g. https://eth.blockscout.com.
     */
    constructor(apiUrl: string, browserUrl: string);
    static getCurrentChainConfig(networkName: string, ethereumProvider: EthereumProvider, customChains: ChainConfig[]): Promise<ChainConfig>;
    static fromChainConfig(chainConfig: ChainConfig): Blockscout;
    /**
     * Check if a smart contract is verified on Blockscout.
     * @link https://docs.blockscout.com/for-users/api/rpc-endpoints/contract#get-contract-source-code-for-a-verified-contract
     * @param address - The address of the smart contract.
     * @returns True if the contract is verified, false otherwise.
     * @throws {NetworkRequestError} if there is an error on the request.
     * @throws {ContractVerificationInvalidStatusCodeError} if the API returns an invalid status code.
     */
    isVerified(address: string): Promise<boolean>;
    /**
     * Verify a smart contract on Blockscout.
     * @link https://docs.blockscout.com/for-users/api/rpc-endpoints/contract#verify-a-contract-with-standard-input-json-file
     * @param contractAddress - The address of the smart contract to verify.
     * @param sourceCode - The source code of the smart contract.
     * @param contractName - The name of the smart contract, e.g. "contracts/Sample.sol:MyContract"
     * @param compilerVersion - The version of the Solidity compiler used, e.g. `v0.8.19+commit.7dd6d404`
     * @returns A promise that resolves to an `BlockscoutResponse` object.
     * @throws {NetworkRequestError} if there is an error on the request.
     * @throws {ContractVerificationInvalidStatusCodeError} if the API returns an invalid status code.
     * @throws {ContractVerificationMissingBytecodeError} if the bytecode is not found on the block explorer.
     * @throws {ContractAlreadyVerifiedError} if the contract is already verified.
     * @throws {HardhatVerifyError} if the response status is not OK.
     */
    verify(contractAddress: string, sourceCode: string, contractName: string, compilerVersion: string): Promise<BlockscoutResponse>;
    /**
     * Get the verification status of a smart contract from Blockscout.
     * This method performs polling of the verification status if it's pending.
     * @link https://docs.blockscout.com/for-users/api/rpc-endpoints/contract#return-status-of-a-verification-attempt
     * @param guid - The verification GUID to check.
     * @returns A promise that resolves to an `BlockscoutResponse` object.
     * @throws {NetworkRequestError} if there is an error on the request.
     * @throws {ContractStatusPollingInvalidStatusCodeError} if the API returns an invalid status code.
     * @throws {ContractStatusPollingResponseNotOkError} if the response status is not OK.
     */
    getVerificationStatus(guid: string): Promise<BlockscoutResponse>;
    /**
     * Get the Blockscout URL for viewing a contract's details.
     * @param address - The address of the smart contract.
     * @returns The URL to view the contract on Blockscout's website.
     */
    getContractUrl(address: string): string;
}
declare class BlockscoutResponse implements ValidationResponse {
    readonly status: number;
    readonly message: string;
    constructor(status: number, message: string);
    isPending(): boolean;
    isFailure(): boolean;
    isSuccess(): boolean;
    isAlreadyVerified(): boolean;
    isOk(): boolean;
}
export {};
//# sourceMappingURL=blockscout.d.ts.map