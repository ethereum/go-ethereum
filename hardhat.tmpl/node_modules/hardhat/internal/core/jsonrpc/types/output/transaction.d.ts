import * as t from "io-ts";
export type RpcTransaction = t.TypeOf<typeof rpcTransaction>;
export declare const rpcTransaction: t.TypeC<{
    blockHash: t.Type<Buffer | null, Buffer | null, unknown>;
    blockNumber: t.Type<bigint | null, bigint | null, unknown>;
    from: t.Type<Buffer, Buffer, unknown>;
    gas: t.Type<bigint, bigint, unknown>;
    gasPrice: t.Type<bigint, bigint, unknown>;
    hash: t.Type<Buffer, Buffer, unknown>;
    input: t.Type<Buffer, Buffer, unknown>;
    nonce: t.Type<bigint, bigint, unknown>;
    to: t.Type<Buffer | null | undefined, Buffer | null | undefined, unknown>;
    transactionIndex: t.Type<bigint | null, bigint | null, unknown>;
    value: t.Type<bigint, bigint, unknown>;
    v: t.Type<bigint, bigint, unknown>;
    r: t.Type<bigint, bigint, unknown>;
    s: t.Type<bigint, bigint, unknown>;
    type: t.Type<bigint | undefined, bigint | undefined, unknown>;
    chainId: t.Type<bigint | null | undefined, bigint | null | undefined, unknown>;
    accessList: t.Type<{
        address: Buffer;
        storageKeys: Buffer[] | null;
    }[] | undefined, {
        address: Buffer;
        storageKeys: Buffer[] | null;
    }[] | undefined, unknown>;
    maxFeePerGas: t.Type<bigint | undefined, bigint | undefined, unknown>;
    maxPriorityFeePerGas: t.Type<bigint | undefined, bigint | undefined, unknown>;
}>;
//# sourceMappingURL=transaction.d.ts.map