import * as t from "io-ts";
export type RpcTransactionReceipt = t.TypeOf<typeof rpcTransactionReceipt>;
export declare const rpcTransactionReceipt: t.TypeC<{
    transactionHash: t.Type<Buffer, Buffer, unknown>;
    transactionIndex: t.Type<bigint, bigint, unknown>;
    blockHash: t.Type<Buffer, Buffer, unknown>;
    blockNumber: t.Type<bigint, bigint, unknown>;
    from: t.Type<Buffer, Buffer, unknown>;
    to: t.Type<Buffer | null, Buffer | null, unknown>;
    cumulativeGasUsed: t.Type<bigint, bigint, unknown>;
    gasUsed: t.Type<bigint, bigint, unknown>;
    contractAddress: t.Type<Buffer | null, Buffer | null, unknown>;
    logs: t.ArrayC<t.TypeC<{
        logIndex: t.Type<bigint | null, bigint | null, unknown>;
        transactionIndex: t.Type<bigint | null, bigint | null, unknown>;
        transactionHash: t.Type<Buffer | null, Buffer | null, unknown>;
        blockHash: t.Type<Buffer | null, Buffer | null, unknown>;
        blockNumber: t.Type<bigint | null, bigint | null, unknown>;
        address: t.Type<Buffer, Buffer, unknown>;
        data: t.Type<Buffer, Buffer, unknown>;
        topics: t.ArrayC<t.Type<Buffer, Buffer, unknown>>;
    }>>;
    logsBloom: t.Type<Buffer, Buffer, unknown>;
    status: t.Type<bigint | null | undefined, bigint | null | undefined, unknown>;
    root: t.Type<Buffer | undefined, Buffer | undefined, unknown>;
    type: t.Type<bigint | undefined, bigint | undefined, unknown>;
    effectiveGasPrice: t.Type<bigint | undefined, bigint | undefined, unknown>;
}>;
//# sourceMappingURL=receipt.d.ts.map