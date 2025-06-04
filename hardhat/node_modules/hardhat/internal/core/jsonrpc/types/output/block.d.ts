import * as t from "io-ts";
export type RpcBlock = t.TypeOf<typeof rpcBlock>;
export declare const rpcBlock: t.TypeC<{
    transactions: t.ArrayC<t.Type<Buffer, Buffer, unknown>>;
    number: t.Type<bigint | null, bigint | null, unknown>;
    hash: t.Type<Buffer | null, Buffer | null, unknown>;
    parentHash: t.Type<Buffer, Buffer, unknown>;
    nonce: t.Type<Buffer | undefined, Buffer | undefined, unknown>;
    sha3Uncles: t.Type<Buffer, Buffer, unknown>;
    logsBloom: t.Type<Buffer, Buffer, unknown>;
    transactionsRoot: t.Type<Buffer, Buffer, unknown>;
    stateRoot: t.Type<Buffer, Buffer, unknown>;
    receiptsRoot: t.Type<Buffer, Buffer, unknown>;
    miner: t.Type<Buffer, Buffer, unknown>;
    difficulty: t.Type<bigint, bigint, unknown>;
    totalDifficulty: t.Type<bigint | undefined, bigint | undefined, unknown>;
    extraData: t.Type<Buffer, Buffer, unknown>;
    size: t.Type<bigint, bigint, unknown>;
    gasLimit: t.Type<bigint, bigint, unknown>;
    gasUsed: t.Type<bigint, bigint, unknown>;
    timestamp: t.Type<bigint, bigint, unknown>;
    uncles: t.ArrayC<t.Type<Buffer, Buffer, unknown>>;
    mixHash: t.Type<Buffer | undefined, Buffer | undefined, unknown>;
    baseFeePerGas: t.Type<bigint | undefined, bigint | undefined, unknown>;
    withdrawals: t.Type<{
        index: bigint;
        validatorIndex: bigint;
        address: Buffer;
        amount: bigint;
    }[] | undefined, {
        index: bigint;
        validatorIndex: bigint;
        address: Buffer;
        amount: bigint;
    }[] | undefined, unknown>;
    withdrawalsRoot: t.Type<Buffer | undefined, Buffer | undefined, unknown>;
    parentBeaconBlockRoot: t.Type<Buffer | undefined, Buffer | undefined, unknown>;
    blobGasUsed: t.Type<bigint | undefined, bigint | undefined, unknown>;
    excessBlobGas: t.Type<bigint | undefined, bigint | undefined, unknown>;
}>;
export type RpcBlockWithTransactions = t.TypeOf<typeof rpcBlockWithTransactions>;
export declare const rpcBlockWithTransactions: t.TypeC<{
    transactions: t.ArrayC<t.TypeC<{
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
    }>>;
    number: t.Type<bigint | null, bigint | null, unknown>;
    hash: t.Type<Buffer | null, Buffer | null, unknown>;
    parentHash: t.Type<Buffer, Buffer, unknown>;
    nonce: t.Type<Buffer | undefined, Buffer | undefined, unknown>;
    sha3Uncles: t.Type<Buffer, Buffer, unknown>;
    logsBloom: t.Type<Buffer, Buffer, unknown>;
    transactionsRoot: t.Type<Buffer, Buffer, unknown>;
    stateRoot: t.Type<Buffer, Buffer, unknown>;
    receiptsRoot: t.Type<Buffer, Buffer, unknown>;
    miner: t.Type<Buffer, Buffer, unknown>;
    difficulty: t.Type<bigint, bigint, unknown>;
    totalDifficulty: t.Type<bigint | undefined, bigint | undefined, unknown>;
    extraData: t.Type<Buffer, Buffer, unknown>;
    size: t.Type<bigint, bigint, unknown>;
    gasLimit: t.Type<bigint, bigint, unknown>;
    gasUsed: t.Type<bigint, bigint, unknown>;
    timestamp: t.Type<bigint, bigint, unknown>;
    uncles: t.ArrayC<t.Type<Buffer, Buffer, unknown>>;
    mixHash: t.Type<Buffer | undefined, Buffer | undefined, unknown>;
    baseFeePerGas: t.Type<bigint | undefined, bigint | undefined, unknown>;
    withdrawals: t.Type<{
        index: bigint;
        validatorIndex: bigint;
        address: Buffer;
        amount: bigint;
    }[] | undefined, {
        index: bigint;
        validatorIndex: bigint;
        address: Buffer;
        amount: bigint;
    }[] | undefined, unknown>;
    withdrawalsRoot: t.Type<Buffer | undefined, Buffer | undefined, unknown>;
    parentBeaconBlockRoot: t.Type<Buffer | undefined, Buffer | undefined, unknown>;
    blobGasUsed: t.Type<bigint | undefined, bigint | undefined, unknown>;
    excessBlobGas: t.Type<bigint | undefined, bigint | undefined, unknown>;
}>;
//# sourceMappingURL=block.d.ts.map