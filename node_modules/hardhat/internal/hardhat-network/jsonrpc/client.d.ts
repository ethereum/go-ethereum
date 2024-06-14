/// <reference types="node" />
import { Address } from "@nomicfoundation/ethereumjs-util";
import { RpcBlock, RpcBlockWithTransactions } from "../../core/jsonrpc/types/output/block";
import { HttpProvider } from "../../core/providers/http";
export declare class JsonRpcClient {
    private _httpProvider;
    private _networkId;
    private _latestBlockNumberOnCreation;
    private _maxReorg;
    private _forkCachePath?;
    private _cache;
    constructor(_httpProvider: HttpProvider, _networkId: number, _latestBlockNumberOnCreation: bigint, _maxReorg: bigint, _forkCachePath?: string | undefined);
    getNetworkId(): number;
    getDebugTraceTransaction(transactionHash: Buffer): Promise<any>;
    getStorageAt(address: Address, position: bigint, blockNumber: bigint): Promise<Buffer>;
    getBlockByNumber(blockNumber: bigint, includeTransactions?: false): Promise<RpcBlock | null>;
    getBlockByNumber(blockNumber: bigint, includeTransactions: true): Promise<RpcBlockWithTransactions | null>;
    getBlockByHash(blockHash: Buffer, includeTransactions?: false): Promise<RpcBlock | null>;
    getBlockByHash(blockHash: Buffer, includeTransactions: true): Promise<RpcBlockWithTransactions | null>;
    getTransactionByHash(transactionHash: Buffer): Promise<{
        blockHash: Buffer | null;
        blockNumber: bigint | null;
        from: Buffer;
        gas: bigint;
        gasPrice: bigint;
        hash: Buffer;
        input: Buffer;
        nonce: bigint;
        to: Buffer | null | undefined;
        transactionIndex: bigint | null;
        value: bigint;
        v: bigint;
        r: bigint;
        s: bigint;
        type: bigint | undefined;
        chainId: bigint | null | undefined;
        accessList: {
            address: Buffer;
            storageKeys: Buffer[] | null;
        }[] | undefined;
        maxFeePerGas: bigint | undefined;
        maxPriorityFeePerGas: bigint | undefined;
    } | null>;
    getTransactionCount(address: Uint8Array, blockNumber: bigint): Promise<bigint>;
    getTransactionReceipt(transactionHash: Buffer): Promise<{
        transactionHash: Buffer;
        transactionIndex: bigint;
        blockHash: Buffer;
        blockNumber: bigint;
        from: Buffer;
        to: Buffer | null;
        cumulativeGasUsed: bigint;
        gasUsed: bigint;
        contractAddress: Buffer | null;
        logs: {
            logIndex: bigint | null;
            transactionIndex: bigint | null;
            transactionHash: Buffer | null;
            blockHash: Buffer | null;
            blockNumber: bigint | null;
            address: Buffer;
            data: Buffer;
            topics: Buffer[];
        }[];
        logsBloom: Buffer;
        status: bigint | null | undefined;
        root: Buffer | undefined;
        type: bigint | undefined;
        effectiveGasPrice: bigint | undefined;
    } | null>;
    getLogs(options: {
        fromBlock: bigint;
        toBlock: bigint;
        address?: Uint8Array | Uint8Array[];
        topics?: Array<Array<Uint8Array | null> | null>;
    }): Promise<{
        logIndex: bigint | null;
        transactionIndex: bigint | null;
        transactionHash: Buffer | null;
        blockHash: Buffer | null;
        blockNumber: bigint | null;
        address: Buffer;
        data: Buffer;
        topics: Buffer[];
    }[]>;
    getAccountData(address: Address, blockNumber: bigint): Promise<{
        code: Buffer;
        transactionCount: bigint;
        balance: bigint;
    }>;
    getLatestBlockNumber(): Promise<bigint>;
    private _perform;
    private _performBatch;
    private _send;
    private _sendBatch;
    private _shouldRetry;
    private _getCacheKey;
    private _getBatchCacheKey;
    private _getFromCache;
    private _storeInCache;
    private _getFromDiskCache;
    private _getBatchFromDiskCache;
    private _getRawFromDiskCache;
    private _storeInDiskCache;
    private _getDiskCachePathForKey;
    private _canBeCached;
    private _canBeReorgedOut;
}
//# sourceMappingURL=client.d.ts.map