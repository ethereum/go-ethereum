import type { FeeMarketEIP1559Transaction } from './eip1559Transaction.js';
import type { AccessListEIP2930Transaction } from './eip2930Transaction.js';
import type { BlobEIP4844Transaction } from './eip4844Transaction.js';
import type { LegacyTransaction } from './legacyTransaction.js';
import type { AccessList, AccessListBytes, Common, Hardfork } from '@nomicfoundation/ethereumjs-common';
import type { Address, AddressLike, BigIntLike, BytesLike } from '@nomicfoundation/ethereumjs-util';
export type { AccessList, AccessListBytes, AccessListBytesItem, AccessListItem, } from '@nomicfoundation/ethereumjs-common';
/**
 * Can be used in conjunction with {@link Transaction[TransactionType].supports}
 * to query on tx capabilities
 */
export declare enum Capability {
    /**
     * Tx supports EIP-155 replay protection
     * See: [155](https://eips.ethereum.org/EIPS/eip-155) Replay Attack Protection EIP
     */
    EIP155ReplayProtection = 155,
    /**
     * Tx supports EIP-1559 gas fee market mechanism
     * See: [1559](https://eips.ethereum.org/EIPS/eip-1559) Fee Market EIP
     */
    EIP1559FeeMarket = 1559,
    /**
     * Tx is a typed transaction as defined in EIP-2718
     * See: [2718](https://eips.ethereum.org/EIPS/eip-2718) Transaction Type EIP
     */
    EIP2718TypedTransaction = 2718,
    /**
     * Tx supports access list generation as defined in EIP-2930
     * See: [2930](https://eips.ethereum.org/EIPS/eip-2930) Access Lists EIP
     */
    EIP2930AccessLists = 2930
}
/**
 * The options for initializing a {@link Transaction}.
 */
export interface TxOptions {
    /**
     * A {@link Common} object defining the chain and hardfork for the transaction.
     *
     * Object will be internally copied so that tx behavior don't incidentally
     * change on future HF changes.
     *
     * Default: {@link Common} object set to `mainnet` and the default hardfork as defined in the {@link Common} class.
     *
     * Current default hardfork: `istanbul`
     */
    common?: Common;
    /**
     * A transaction object by default gets frozen along initialization. This gives you
     * strong additional security guarantees on the consistency of the tx parameters.
     * It also enables tx hash caching when the `hash()` method is called multiple times.
     *
     * If you need to deactivate the tx freeze - e.g. because you want to subclass tx and
     * add additional properties - it is strongly encouraged that you do the freeze yourself
     * within your code instead.
     *
     * Default: true
     */
    freeze?: boolean;
    /**
     * Allows unlimited contract code-size init while debugging. This (partially) disables EIP-3860.
     * Gas cost for initcode size analysis will still be charged. Use with caution.
     */
    allowUnlimitedInitCodeSize?: boolean;
}
export declare function isAccessListBytes(input: AccessListBytes | AccessList): input is AccessListBytes;
export declare function isAccessList(input: AccessListBytes | AccessList): input is AccessList;
export interface TransactionCache {
    hash?: Uint8Array;
    dataFee?: {
        value: bigint;
        hardfork: string | Hardfork;
    };
    senderPubKey?: Uint8Array;
    senderAddress?: Address;
}
/**
 * Encompassing type for all transaction types.
 */
export declare enum TransactionType {
    Legacy = 0,
    AccessListEIP2930 = 1,
    FeeMarketEIP1559 = 2,
    BlobEIP4844 = 3
}
export interface Transaction {
    [TransactionType.Legacy]: LegacyTransaction;
    [TransactionType.FeeMarketEIP1559]: FeeMarketEIP1559Transaction;
    [TransactionType.AccessListEIP2930]: AccessListEIP2930Transaction;
    [TransactionType.BlobEIP4844]: BlobEIP4844Transaction;
}
export declare type TypedTransaction = Transaction[TransactionType];
export declare function isLegacyTx(tx: TypedTransaction): tx is LegacyTransaction;
export declare function isAccessListEIP2930Tx(tx: TypedTransaction): tx is AccessListEIP2930Transaction;
export declare function isFeeMarketEIP1559Tx(tx: TypedTransaction): tx is FeeMarketEIP1559Transaction;
export declare function isBlobEIP4844Tx(tx: TypedTransaction): tx is BlobEIP4844Transaction;
export interface TransactionInterface<T extends TransactionType = TransactionType> {
    readonly common: Common;
    readonly nonce: bigint;
    readonly gasLimit: bigint;
    readonly to?: Address;
    readonly value: bigint;
    readonly data: Uint8Array;
    readonly v?: bigint;
    readonly r?: bigint;
    readonly s?: bigint;
    readonly cache: TransactionCache;
    supports(capability: Capability): boolean;
    type: TransactionType;
    getBaseFee(): bigint;
    getDataFee(): bigint;
    getUpfrontCost(): bigint;
    toCreationAddress(): boolean;
    raw(): TxValuesArray[T];
    serialize(): Uint8Array;
    getMessageToSign(): Uint8Array | Uint8Array[];
    getHashedMessageToSign(): Uint8Array;
    hash(): Uint8Array;
    getMessageToVerifySignature(): Uint8Array;
    getValidationErrors(): string[];
    isSigned(): boolean;
    isValid(): boolean;
    verifySignature(): boolean;
    getSenderAddress(): Address;
    getSenderPublicKey(): Uint8Array;
    sign(privateKey: Uint8Array): Transaction[T];
    toJSON(): JsonTx;
    errorStr(): string;
}
export interface LegacyTxInterface<T extends TransactionType = TransactionType> extends TransactionInterface<T> {
}
export interface EIP2718CompatibleTx<T extends TransactionType = TransactionType> extends TransactionInterface<T> {
    readonly chainId: bigint;
    getMessageToSign(): Uint8Array;
}
export interface EIP2930CompatibleTx<T extends TransactionType = TransactionType> extends EIP2718CompatibleTx<T> {
    readonly accessList: AccessListBytes;
    readonly AccessListJSON: AccessList;
}
export interface EIP1559CompatibleTx<T extends TransactionType = TransactionType> extends EIP2930CompatibleTx<T> {
    readonly maxPriorityFeePerGas: bigint;
    readonly maxFeePerGas: bigint;
}
export interface EIP4844CompatibleTx<T extends TransactionType = TransactionType> extends EIP1559CompatibleTx<T> {
    readonly maxFeePerBlobGas: bigint;
    blobVersionedHashes: Uint8Array[];
    blobs?: Uint8Array[];
    kzgCommitments?: Uint8Array[];
    kzgProofs?: Uint8Array[];
    serializeNetworkWrapper(): Uint8Array;
    numBlobs(): number;
}
export interface TxData {
    [TransactionType.Legacy]: LegacyTxData;
    [TransactionType.AccessListEIP2930]: AccessListEIP2930TxData;
    [TransactionType.FeeMarketEIP1559]: FeeMarketEIP1559TxData;
    [TransactionType.BlobEIP4844]: BlobEIP4844TxData;
}
export declare type TypedTxData = TxData[TransactionType];
export declare function isLegacyTxData(txData: TypedTxData): txData is LegacyTxData;
export declare function isAccessListEIP2930TxData(txData: TypedTxData): txData is AccessListEIP2930TxData;
export declare function isFeeMarketEIP1559TxData(txData: TypedTxData): txData is FeeMarketEIP1559TxData;
export declare function isBlobEIP4844TxData(txData: TypedTxData): txData is BlobEIP4844TxData;
/**
 * Legacy {@link Transaction} Data
 */
export declare type LegacyTxData = {
    /**
     * The transaction's nonce.
     */
    nonce?: BigIntLike;
    /**
     * The transaction's gas price.
     */
    gasPrice?: BigIntLike | null;
    /**
     * The transaction's gas limit.
     */
    gasLimit?: BigIntLike;
    /**
     * The transaction's the address is sent to.
     */
    to?: AddressLike;
    /**
     * The amount of Ether sent.
     */
    value?: BigIntLike;
    /**
     * This will contain the data of the message or the init of a contract.
     */
    data?: BytesLike;
    /**
     * EC recovery ID.
     */
    v?: BigIntLike;
    /**
     * EC signature parameter.
     */
    r?: BigIntLike;
    /**
     * EC signature parameter.
     */
    s?: BigIntLike;
    /**
     * The transaction type
     */
    type?: BigIntLike;
};
/**
 * {@link AccessListEIP2930Transaction} data.
 */
export interface AccessListEIP2930TxData extends LegacyTxData {
    /**
     * The transaction's chain ID
     */
    chainId?: BigIntLike;
    /**
     * The access list which contains the addresses/storage slots which the transaction wishes to access
     */
    accessList?: AccessListBytes | AccessList | null;
}
/**
 * {@link FeeMarketEIP1559Transaction} data.
 */
export interface FeeMarketEIP1559TxData extends AccessListEIP2930TxData {
    /**
     * The transaction's gas price, inherited from {@link Transaction}.  This property is not used for EIP1559
     * transactions and should always be undefined for this specific transaction type.
     */
    gasPrice?: never | null;
    /**
     * The maximum inclusion fee per gas (this fee is given to the miner)
     */
    maxPriorityFeePerGas?: BigIntLike;
    /**
     * The maximum total fee
     */
    maxFeePerGas?: BigIntLike;
}
/**
 * {@link BlobEIP4844Transaction} data.
 */
export interface BlobEIP4844TxData extends FeeMarketEIP1559TxData {
    /**
     * The versioned hashes used to validate the blobs attached to a transaction
     */
    blobVersionedHashes?: BytesLike[];
    /**
     * The maximum fee per blob gas paid for the transaction
     */
    maxFeePerBlobGas?: BigIntLike;
    /**
     * The blobs associated with a transaction
     */
    blobs?: BytesLike[];
    /**
     * The KZG commitments corresponding to the versioned hashes for each blob
     */
    kzgCommitments?: BytesLike[];
    /**
     * The KZG proofs associated with the transaction
     */
    kzgProofs?: BytesLike[];
    /**
     * An array of arbitrary strings that blobs are to be constructed from
     */
    blobsData?: string[];
}
export interface TxValuesArray {
    [TransactionType.Legacy]: LegacyTxValuesArray;
    [TransactionType.AccessListEIP2930]: AccessListEIP2930TxValuesArray;
    [TransactionType.FeeMarketEIP1559]: FeeMarketEIP1559TxValuesArray;
    [TransactionType.BlobEIP4844]: BlobEIP4844TxValuesArray;
}
/**
 * Bytes values array for a legacy {@link Transaction}
 */
declare type LegacyTxValuesArray = Uint8Array[];
/**
 * Bytes values array for an {@link AccessListEIP2930Transaction}
 */
declare type AccessListEIP2930TxValuesArray = [
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    AccessListBytes,
    Uint8Array?,
    Uint8Array?,
    Uint8Array?
];
/**
 * Bytes values array for a {@link FeeMarketEIP1559Transaction}
 */
declare type FeeMarketEIP1559TxValuesArray = [
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    AccessListBytes,
    Uint8Array?,
    Uint8Array?,
    Uint8Array?
];
/**
 * Bytes values array for a {@link BlobEIP4844Transaction}
 */
declare type BlobEIP4844TxValuesArray = [
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    AccessListBytes,
    Uint8Array,
    Uint8Array[],
    Uint8Array?,
    Uint8Array?,
    Uint8Array?
];
export declare type BlobEIP4844NetworkValuesArray = [
    BlobEIP4844TxValuesArray,
    Uint8Array[],
    Uint8Array[],
    Uint8Array[]
];
declare type JsonAccessListItem = {
    address: string;
    storageKeys: string[];
};
/**
 * Generic interface for all tx types with a
 * JSON representation of a transaction.
 *
 * Note that all values are marked as optional
 * and not all the values are present on all tx types
 * (an EIP1559 tx e.g. lacks a `gasPrice`).
 */
export interface JsonTx {
    nonce?: string;
    gasPrice?: string;
    gasLimit?: string;
    to?: string;
    data?: string;
    v?: string;
    r?: string;
    s?: string;
    value?: string;
    chainId?: string;
    accessList?: JsonAccessListItem[];
    type?: string;
    maxPriorityFeePerGas?: string;
    maxFeePerGas?: string;
    maxFeePerBlobGas?: string;
    blobVersionedHashes?: string[];
}
export interface JsonRpcTx {
    blockHash: string | null;
    blockNumber: string | null;
    from: string;
    gas: string;
    gasPrice: string;
    maxFeePerGas?: string;
    maxPriorityFeePerGas?: string;
    type: string;
    accessList?: JsonTx['accessList'];
    chainId?: string;
    hash: string;
    input: string;
    nonce: string;
    to: string | null;
    transactionIndex: string | null;
    value: string;
    v: string;
    r: string;
    s: string;
    maxFeePerBlobGas?: string;
    blobVersionedHashes?: string[];
}
//# sourceMappingURL=types.d.ts.map