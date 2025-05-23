import * as t from "io-ts";
export type RpcLog = t.TypeOf<typeof rpcLog>;
export declare const rpcLog: t.TypeC<{
    logIndex: t.Type<bigint | null, bigint | null, unknown>;
    transactionIndex: t.Type<bigint | null, bigint | null, unknown>;
    transactionHash: t.Type<Buffer | null, Buffer | null, unknown>;
    blockHash: t.Type<Buffer | null, Buffer | null, unknown>;
    blockNumber: t.Type<bigint | null, bigint | null, unknown>;
    address: t.Type<Buffer, Buffer, unknown>;
    data: t.Type<Buffer, Buffer, unknown>;
    topics: t.ArrayC<t.Type<Buffer, Buffer, unknown>>;
}>;
//# sourceMappingURL=log.d.ts.map