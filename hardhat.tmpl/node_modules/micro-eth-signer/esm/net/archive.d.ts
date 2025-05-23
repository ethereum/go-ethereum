import { TxVersions, type AccessList } from '../tx.ts';
import { type IWeb3Provider, type Web3CallArgs } from '../utils.ts';
declare const CONTRACT_CAPABILITIES: Record<string, string>;
export type BlockInfo = {
    baseFeePerGas: bigint;
    difficulty: bigint;
    extraData: string;
    gasLimit: bigint;
    gasUsed: bigint;
    hash: string;
    logsBloom: string;
    miner: string;
    mixHash: string;
    nonce: string;
    number: number;
    parentHash: string;
    receiptsRoot: string;
    sha3Uncles: string;
    size: number;
    stateRoot: string;
    timestamp: number;
    totalDifficulty?: bigint;
    transactions: string[];
    transactionsRoot: string;
    uncles: string[];
};
export type Action = {
    action: {
        from: string;
        callType: string;
        gas: bigint;
        input: string;
        to: string;
        value: bigint;
    };
    blockHash: string;
    blockNumber: number;
    result: {
        gasUsed: bigint;
        output: string;
    };
    subtraces: number;
    traceAddress: string[];
    transactionHash: string;
    transactionPosition: number;
    type: string;
};
export type Log = {
    address: string;
    topics: string[];
    data: string;
    blockNumber: number;
    transactionHash: string;
    transactionIndex: number;
    blockHash: string;
    logIndex: number;
    removed: boolean;
};
export type TxInfo = {
    blockHash: string;
    blockNumber: number;
    hash: string;
    accessList?: AccessList;
    transactionIndex: number;
    type: number;
    nonce: bigint;
    input: string;
    r: bigint;
    s: bigint;
    chainId: bigint;
    v: bigint;
    yParity?: string;
    gas: bigint;
    maxPriorityFeePerGas?: bigint;
    from: string;
    to: string;
    maxFeePerGas?: bigint;
    value: bigint;
    gasPrice: bigint;
    maxFeePerBlobGas?: bigint;
    blobVersionedHashes?: string[];
};
export type TxReceipt = {
    transactionHash: string;
    blockHash: string;
    blockNumber: number;
    logsBloom: string;
    gasUsed: bigint;
    contractAddress: string | null;
    cumulativeGasUsed: bigint;
    transactionIndex: number;
    from: string;
    to: string;
    type: number;
    effectiveGasPrice: bigint;
    logs: Log[];
    status: number;
    blobGasPrice?: bigint;
    blobGasUsed?: bigint;
};
export type Unspent = {
    symbol: 'ETH';
    decimals: number;
    balance: bigint;
    nonce: number;
    active: boolean;
};
type ERC20Token = {
    abi: 'ERC20';
    name?: string;
    symbol?: string;
    decimals?: number;
    totalSupply: bigint;
};
type ERC721Token = {
    abi: 'ERC721';
    name?: string;
    symbol?: string;
    totalSupply?: bigint;
    enumerable?: boolean;
    metadata?: boolean;
};
type ERC1155Token = {
    abi: 'ERC1155';
};
export type TokenInfo = {
    contract: string;
} & (ERC20Token | ERC721Token | ERC1155Token);
type TokenError = {
    contract: string;
    error: string;
};
type TokenBalanceSingle = Map<bigint, bigint>;
export type TokenBalances = Record<string, TokenBalanceSingle | TokenError>;
export type Topics = (string | null | (string | null)[])[];
export type Transfer = {
    from: string;
    to?: string;
    value: bigint;
};
export type TokenTransfer = TokenInfo & {
    from: string;
    to: string;
    tokens: Map<bigint, bigint>;
};
export type TxTransfers = {
    hash: string;
    timestamp?: number;
    block?: number;
    transfers: Transfer[];
    tokenTransfers: TokenTransfer[];
    reverted: boolean;
    info: {
        type: keyof typeof TxVersions;
        info: TxInfo;
        receipt: TxReceipt;
        raw?: string;
        block: BlockInfo;
        actions: Action[];
    };
};
/**
 * Callbacks are needed, because we want to call getTx / getBlock / getTokenInfo
 * requests as fast as possible, to reduce amount of sequential execution.
 * If we retrieve 10 pages of transactions, we can call per tx
 * callbacks for transaction from first page before all other pages fetched.
 *
 * Ensure caching: they can be called multiple times for same tx / block.
 */
export type Callbacks = {
    txCallback?: (txHash: string) => void;
    blockCallback?: (blockNum: number) => void;
    contractCallback?: (contrct: string) => void;
};
export type Pagination = {
    fromBlock?: number;
    toBlock?: number;
};
export type TraceOpts = Callbacks & Pagination & {
    perRequest?: number;
    limitTrace?: number;
};
export type LogOpts = Callbacks & (Pagination | {
    fromBlock: number;
    toBlock: number;
    limitLogs: number;
});
export type Balances = {
    balances: Record<string, bigint>;
    tokenBalances: Record<string, Record<string, bigint>>;
};
export type TxInfoOpts = Callbacks & {
    ignoreTxRebuildErrors?: boolean;
};
export type TxAllowances = Record<string, Record<string, bigint>>;
export type JsonrpcInterface = {
    call: (method: string, ...args: any[]) => Promise<any>;
};
/**
 * Transaction-related code around Web3Provider.
 * High-level methods are `height`, `unspent`, `transfers`, `allowances` and `tokenBalances`.
 *
 * Low-level methods are `blockInfo`, `internalTransactions`, `ethLogs`, `tokenTransfers`, `wethTransfers`,
 * `tokenInfo` and `txInfo`.
 */
export declare class Web3Provider implements IWeb3Provider {
    private rpc;
    constructor(rpc: JsonrpcInterface);
    call(method: string, ...args: any[]): Promise<any>;
    ethCall(args: Web3CallArgs, tag?: string): Promise<any>;
    estimateGas(args: Web3CallArgs, tag?: string): Promise<bigint>;
    blockInfo(block: number): Promise<BlockInfo>;
    unspent(address: string): Promise<Unspent>;
    height(): Promise<number>;
    traceFilterSingle(address: string, opts?: TraceOpts): Promise<any>;
    internalTransactions(address: string, opts?: TraceOpts): Promise<any[]>;
    contractCapabilities(address: string, capabilities?: typeof CONTRACT_CAPABILITIES): Promise<{
        [k: string]: boolean;
    }>;
    ethLogsSingle(topics: Topics, opts: LogOpts): Promise<Log[]>;
    ethLogs(topics: Topics, opts?: LogOpts): Promise<Log[]>;
    tokenTransfers(address: string, opts?: LogOpts): Promise<[Log[], Log[]]>;
    wethTransfers(address: string, opts?: LogOpts): Promise<[Log[]]>;
    erc1155Transfers(address: string, opts?: LogOpts): Promise<[Log[], Log[], Log[], Log[]]>;
    txInfo(txHash: string, opts?: TxInfoOpts): Promise<{
        type: 'legacy' | 'eip2930' | 'eip1559' | 'eip4844' | 'eip7702';
        info: any;
        receipt: any;
        raw: string | undefined;
    }>;
    tokenInfo(contract: string): Promise<TokenInfo | TokenError>;
    private tokenBalanceSingle;
    tokenURI(token: TokenInfo | TokenError | string, tokenId: bigint): Promise<string | TokenError>;
    tokenBalances(address: string, tokens: string[], tokenIds?: Record<string, Set<bigint>>): Promise<TokenBalances>;
    private decodeTokenTransfer;
    transfers(address: string, opts?: TraceOpts & LogOpts): Promise<TxTransfers[]>;
    allowances(address: string, opts?: LogOpts): Promise<TxAllowances>;
}
/**
 * Calculates balances at specific point in time after tx.
 * Also, useful as a sanity check in case we've missed something.
 * Info from multiple addresses can be merged (sort everything first).
 */
export declare function calcTransfersDiff(transfers: TxTransfers[]): (TxTransfers & Balances)[];
export {};
//# sourceMappingURL=archive.d.ts.map