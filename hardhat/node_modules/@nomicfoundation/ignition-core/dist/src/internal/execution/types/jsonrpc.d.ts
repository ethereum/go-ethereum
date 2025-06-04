/**
 * The result of a static call, as returned by eth_call.
 */
export interface RawStaticCallResult {
    /**
     * The data returned by the call.
     */
    returnData: string;
    /**
     * A boolean indicating whether the call was successful or not.
     */
    success: boolean;
    /**
     * A boolean indicating whether the JSON-RPC server that run the
     * call reported that the call failed due to a custom error.
     */
    customErrorReported: boolean;
}
/**
 * The relevant subset of a transaction log, as returned by eth_getTransactionReceipt.
 */
export interface TransactionLog {
    address: string;
    logIndex: number;
    data: string;
    topics: string[];
}
/**
 * The status of a transaction, as represented in its receipt.
 */
export declare enum TransactionReceiptStatus {
    FAILURE = "FAILURE",
    SUCCESS = "SUCCESS"
}
/**
 * The relevant subset of the receipt.
 */
export interface TransactionReceipt {
    blockHash: string;
    blockNumber: number;
    contractAddress?: string;
    status: TransactionReceiptStatus;
    logs: TransactionLog[];
}
/**
 * Network fees for EIP-1559 transactions.
 */
export interface EIP1559NetworkFees {
    maxPriorityFeePerGas: bigint;
    maxFeePerGas: bigint;
}
/**
 * Network fees for non-EIP-1559 transactions.
 */
export interface LegacyNetworkFees {
    gasPrice: bigint;
}
/**
 * The params to pay for the network fees.
 */
export type NetworkFees = EIP1559NetworkFees | LegacyNetworkFees;
/**
 * This interface represents a transaction that was sent to the network.
 */
export interface Transaction {
    hash: string;
    fees: NetworkFees;
    receipt?: TransactionReceipt;
}
/**
 * This interface represents a transaction with all of its available fields.
 */
export interface FullTransaction {
    hash: string;
    blockNumber: bigint | null;
    blockHash: string | null;
    nonce: number;
    chainId: number;
    from: string;
    to: string | undefined;
    value: bigint;
    data: string;
    gasLimit?: bigint;
    gasPrice?: bigint;
    maxPriorityFeePerGas?: bigint;
    maxFeePerGas?: bigint;
}
interface BaseNetworkTransaction {
    hash: string;
    blockNumber: string | null;
    blockHash: string | null;
    nonce: string;
    chainId: string;
    from: string;
    to: string | null;
    value: string;
    input: string;
    gas?: string;
}
type LegacyNetworkTransaction = BaseNetworkTransaction & {
    [P in keyof LegacyNetworkFees]: string;
};
type EIP1559NetworkTransaction = BaseNetworkTransaction & {
    [P in keyof EIP1559NetworkFees]: string;
};
/**
 * This type represents a transaction that was retrieved from the network.
 */
export type NetworkTransaction = LegacyNetworkTransaction | EIP1559NetworkTransaction;
export {};
//# sourceMappingURL=jsonrpc.d.ts.map