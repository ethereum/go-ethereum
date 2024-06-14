import * as t from "io-ts";
export declare const rpcNewBlockTagObjectWithNumber: t.TypeC<{
    blockNumber: t.Type<bigint, bigint, unknown>;
}>;
export declare const rpcNewBlockTagObjectWithHash: t.TypeC<{
    blockHash: t.Type<Buffer, Buffer, unknown>;
    requireCanonical: t.Type<boolean | undefined, boolean | undefined, unknown>;
}>;
export declare const rpcBlockTagName: t.KeyofC<{
    earliest: null;
    latest: null;
    pending: null;
    safe: null;
    finalized: null;
}>;
export declare const rpcNewBlockTag: t.UnionC<[t.Type<bigint, bigint, unknown>, t.TypeC<{
    blockNumber: t.Type<bigint, bigint, unknown>;
}>, t.TypeC<{
    blockHash: t.Type<Buffer, Buffer, unknown>;
    requireCanonical: t.Type<boolean | undefined, boolean | undefined, unknown>;
}>, t.KeyofC<{
    earliest: null;
    latest: null;
    pending: null;
    safe: null;
    finalized: null;
}>]>;
export type RpcNewBlockTag = t.TypeOf<typeof rpcNewBlockTag>;
export declare const optionalRpcNewBlockTag: t.Type<bigint | "pending" | "latest" | "earliest" | "safe" | "finalized" | {
    blockNumber: bigint;
} | {
    blockHash: Buffer;
    requireCanonical: boolean | undefined;
} | undefined, bigint | "pending" | "latest" | "earliest" | "safe" | "finalized" | {
    blockNumber: bigint;
} | {
    blockHash: Buffer;
    requireCanonical: boolean | undefined;
} | undefined, unknown>;
export type OptionalRpcNewBlockTag = t.TypeOf<typeof optionalRpcNewBlockTag>;
export declare const rpcOldBlockTag: t.UnionC<[t.Type<bigint, bigint, unknown>, t.KeyofC<{
    earliest: null;
    latest: null;
    pending: null;
    safe: null;
    finalized: null;
}>]>;
export type RpcOldBlockTag = t.TypeOf<typeof rpcOldBlockTag>;
export declare const optionalRpcOldBlockTag: t.Type<bigint | "pending" | "latest" | "earliest" | "safe" | "finalized" | undefined, bigint | "pending" | "latest" | "earliest" | "safe" | "finalized" | undefined, unknown>;
export type OptionalRpcOldBlockTag = t.TypeOf<typeof optionalRpcOldBlockTag>;
//# sourceMappingURL=blockTag.d.ts.map