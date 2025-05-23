export interface RpcBlockOutput {
    difficulty: string;
    extraData: string;
    gasLimit: string;
    gasUsed: string;
    hash: string | null;
    logsBloom: string;
    miner: string;
    mixHash: string | null;
    nonce: string | null;
    number: string | null;
    parentHash: string;
    receiptsRoot: string;
    sha3Uncles: string;
    size: string;
    stateRoot: string;
    timestamp: string;
    totalDifficulty: string;
    transactions: string[] | RpcTransactionOutput[];
    transactionsRoot: string;
    uncles: string[];
    baseFeePerGas?: string;
    withdrawals?: RpcWithdrawalItem[];
    withdrawalsRoot?: string;
    parentBeaconBlockRoot?: string | null;
    blobGasUsed?: string | null;
    excessBlobGas?: string | null;
}
export type RpcTransactionOutput = LegacyRpcTransactionOutput | AccessListEIP2930RpcTransactionOutput | EIP1559RpcTransactionOutput | EOACodeEIP7702TransactionOutput;
interface BaseRpcTransactionOutput {
    blockHash: string | null;
    blockNumber: string | null;
    from: string;
    gas: string;
    hash: string;
    input: string;
    nonce: string;
    r: string;
    s: string;
    to: string | null;
    transactionIndex: string | null;
    v: string;
    value: string;
    type?: string;
}
export interface LegacyRpcTransactionOutput extends BaseRpcTransactionOutput {
    gasPrice: string;
}
export type RpcAccessListOutput = Array<{
    address: string;
    storageKeys: string[];
}>;
export type RpcAuthorizationListOutput = Array<{
    chainId: string;
    address: string;
    nonce: string;
    yParity: string;
    r: string;
    s: string;
}>;
export interface AccessListEIP2930RpcTransactionOutput extends BaseRpcTransactionOutput {
    gasPrice: string;
    accessList?: RpcAccessListOutput;
    chainId: string;
}
export interface EIP1559RpcTransactionOutput extends BaseRpcTransactionOutput {
    gasPrice: string;
    maxFeePerGas: string;
    maxPriorityFeePerGas: string;
    accessList?: RpcAccessListOutput;
    chainId: string;
}
export interface EOACodeEIP7702TransactionOutput extends EIP1559RpcTransactionOutput {
    authorizationList?: RpcAuthorizationListOutput;
}
export interface RpcReceiptOutput {
    blockHash: string;
    blockNumber: string;
    contractAddress: string | null;
    cumulativeGasUsed: string;
    from: string;
    gasUsed: string;
    logs: RpcLogOutput[];
    logsBloom: string;
    to: string | null;
    transactionHash: string;
    transactionIndex: string;
    status?: string;
    root?: string;
    type?: string;
    effectiveGasPrice?: string;
}
export interface RpcLogOutput {
    address: string;
    blockHash: string | null;
    blockNumber: string | null;
    data: string;
    logIndex: string | null;
    removed: boolean;
    topics: string[];
    transactionHash: string | null;
    transactionIndex: string | null;
}
export interface RpcStructLog {
    depth: number;
    gas: number;
    gasCost: number;
    op: string;
    pc: number;
    memory?: string[];
    stack?: string[];
    storage?: Record<string, string>;
    memSize?: number;
    error?: object;
}
export interface RpcDebugTraceOutput {
    failed: boolean;
    gas: number;
    returnValue: string;
    structLogs: RpcStructLog[];
}
export interface RpcWithdrawalItem {
    index: string;
    validatorIndex: string;
    address: string;
    amount: string;
}
export {};
//# sourceMappingURL=output.d.ts.map