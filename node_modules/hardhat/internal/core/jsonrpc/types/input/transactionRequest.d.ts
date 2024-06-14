import * as t from "io-ts";
export declare const rpcTransactionRequest: t.TypeC<{
    from: t.Type<Buffer, Buffer, unknown>;
    to: t.Type<Buffer | undefined, Buffer | undefined, unknown>;
    gas: t.Type<bigint | undefined, bigint | undefined, unknown>;
    gasPrice: t.Type<bigint | undefined, bigint | undefined, unknown>;
    value: t.Type<bigint | undefined, bigint | undefined, unknown>;
    nonce: t.Type<bigint | undefined, bigint | undefined, unknown>;
    data: t.Type<Buffer | undefined, Buffer | undefined, unknown>;
    accessList: t.Type<{
        address: Buffer;
        storageKeys: Buffer[] | null;
    }[] | undefined, {
        address: Buffer;
        storageKeys: Buffer[] | null;
    }[] | undefined, unknown>;
    chainId: t.Type<bigint | undefined, bigint | undefined, unknown>;
    maxFeePerGas: t.Type<bigint | undefined, bigint | undefined, unknown>;
    maxPriorityFeePerGas: t.Type<bigint | undefined, bigint | undefined, unknown>;
    blobs: t.Type<Buffer[] | undefined, Buffer[] | undefined, unknown>;
    blobVersionedHashes: t.Type<Buffer[] | undefined, Buffer[] | undefined, unknown>;
}>;
export interface RpcTransactionRequestInput {
    from: string;
    to?: string;
    gas?: string;
    gasPrice?: string;
    value?: string;
    nonce?: string;
    data?: string;
    accessList?: Array<{
        address: string;
        storageKeys: string[];
    }>;
    maxFeePerGas?: string;
    maxPriorityFeePerGas?: string;
    blobs?: string[];
    blobVersionedHashes?: string[];
}
export type RpcTransactionRequest = t.TypeOf<typeof rpcTransactionRequest>;
//# sourceMappingURL=transactionRequest.d.ts.map