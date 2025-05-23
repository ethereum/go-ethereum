import * as t from "io-ts";
export declare const rpcFilterRequest: t.TypeC<{
    fromBlock: t.Type<bigint | "pending" | "latest" | "earliest" | "safe" | "finalized" | undefined, bigint | "pending" | "latest" | "earliest" | "safe" | "finalized" | undefined, unknown>;
    toBlock: t.Type<bigint | "pending" | "latest" | "earliest" | "safe" | "finalized" | undefined, bigint | "pending" | "latest" | "earliest" | "safe" | "finalized" | undefined, unknown>;
    address: t.Type<Buffer | Buffer[] | undefined, Buffer | Buffer[] | undefined, unknown>;
    topics: t.Type<(Buffer | (Buffer | null)[] | null)[] | undefined, (Buffer | (Buffer | null)[] | null)[] | undefined, unknown>;
    blockHash: t.Type<Buffer | undefined, Buffer | undefined, unknown>;
}>;
export type RpcFilterRequest = t.TypeOf<typeof rpcFilterRequest>;
export declare const optionalRpcFilterRequest: t.Type<{
    fromBlock: bigint | "pending" | "latest" | "earliest" | "safe" | "finalized" | undefined;
    toBlock: bigint | "pending" | "latest" | "earliest" | "safe" | "finalized" | undefined;
    address: Buffer | Buffer[] | undefined;
    topics: (Buffer | (Buffer | null)[] | null)[] | undefined;
    blockHash: Buffer | undefined;
} | undefined, {
    fromBlock: bigint | "pending" | "latest" | "earliest" | "safe" | "finalized" | undefined;
    toBlock: bigint | "pending" | "latest" | "earliest" | "safe" | "finalized" | undefined;
    address: Buffer | Buffer[] | undefined;
    topics: (Buffer | (Buffer | null)[] | null)[] | undefined;
    blockHash: Buffer | undefined;
} | undefined, unknown>;
export type OptionalRpcFilterRequest = t.TypeOf<typeof optionalRpcFilterRequest>;
//# sourceMappingURL=filterRequest.d.ts.map