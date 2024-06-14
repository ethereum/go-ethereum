import { Chain, Common } from '@nomicfoundation/ethereumjs-common';
import { Address } from '@nomicfoundation/ethereumjs-util';
import { Capability, TransactionType } from './types.js';
import type { JsonTx, Transaction, TransactionCache, TransactionInterface, TxData, TxOptions, TxValuesArray } from './types.js';
import type { BigIntLike } from '@nomicfoundation/ethereumjs-util';
/**
 * This base class will likely be subject to further
 * refactoring along the introduction of additional tx types
 * on the Ethereum network.
 *
 * It is therefore not recommended to use directly.
 */
export declare abstract class BaseTransaction<T extends TransactionType> implements TransactionInterface<T> {
    protected readonly _type: TransactionType;
    readonly nonce: bigint;
    readonly gasLimit: bigint;
    readonly to?: Address;
    readonly value: bigint;
    readonly data: Uint8Array;
    readonly v?: bigint;
    readonly r?: bigint;
    readonly s?: bigint;
    readonly common: Common;
    cache: TransactionCache;
    protected readonly txOptions: TxOptions;
    /**
     * List of tx type defining EIPs,
     * e.g. 1559 (fee market) and 2930 (access lists)
     * for FeeMarketEIP1559Transaction objects
     */
    protected activeCapabilities: number[];
    /**
     * The default chain the tx falls back to if no Common
     * is provided and if the chain can't be derived from
     * a passed in chainId (only EIP-2718 typed txs) or
     * EIP-155 signature (legacy txs).
     *
     * @hidden
     */
    protected DEFAULT_CHAIN: Chain;
    constructor(txData: TxData[T], opts: TxOptions);
    /**
     * Returns the transaction type.
     *
     * Note: legacy txs will return tx type `0`.
     */
    get type(): TransactionType;
    /**
     * Checks if a tx type defining capability is active
     * on a tx, for example the EIP-1559 fee market mechanism
     * or the EIP-2930 access list feature.
     *
     * Note that this is different from the tx type itself,
     * so EIP-2930 access lists can very well be active
     * on an EIP-1559 tx for example.
     *
     * This method can be useful for feature checks if the
     * tx type is unknown (e.g. when instantiated with
     * the tx factory).
     *
     * See `Capabilities` in the `types` module for a reference
     * on all supported capabilities.
     */
    supports(capability: Capability): boolean;
    /**
     * Validates the transaction signature and minimum gas requirements.
     * @returns {string[]} an array of error strings
     */
    getValidationErrors(): string[];
    /**
     * Validates the transaction signature and minimum gas requirements.
     * @returns {boolean} true if the transaction is valid, false otherwise
     */
    isValid(): boolean;
    /**
     * The minimum amount of gas the tx must have (DataFee + TxFee + Creation Fee)
     */
    getBaseFee(): bigint;
    /**
     * The amount of gas paid for the data in this tx
     */
    getDataFee(): bigint;
    /**
     * The up front amount that an account must have for this transaction to be valid
     */
    abstract getUpfrontCost(): bigint;
    /**
     * If the tx's `to` is to the creation address
     */
    toCreationAddress(): boolean;
    /**
     * Returns a Uint8Array Array of the raw Bytes of this transaction, in order.
     *
     * Use {@link BaseTransaction.serialize} to add a transaction to a block
     * with {@link Block.fromValuesArray}.
     *
     * For an unsigned tx this method uses the empty Bytes values for the
     * signature parameters `v`, `r` and `s` for encoding. For an EIP-155 compliant
     * representation for external signing use {@link BaseTransaction.getMessageToSign}.
     */
    abstract raw(): TxValuesArray[T];
    /**
     * Returns the encoding of the transaction.
     */
    abstract serialize(): Uint8Array;
    abstract getMessageToSign(): Uint8Array | Uint8Array[];
    abstract getHashedMessageToSign(): Uint8Array;
    abstract hash(): Uint8Array;
    abstract getMessageToVerifySignature(): Uint8Array;
    isSigned(): boolean;
    /**
     * Determines if the signature is valid
     */
    verifySignature(): boolean;
    /**
     * Returns the sender's address
     */
    getSenderAddress(): Address;
    /**
     * Returns the public key of the sender
     */
    abstract _getSenderPublicKey(): Uint8Array;
    getSenderPublicKey(): Uint8Array;
    /**
     * Signs a transaction.
     *
     * Note that the signed tx is returned as a new object,
     * use as follows:
     * ```javascript
     * const signedTx = tx.sign(privateKey)
     * ```
     */
    sign(privateKey: Uint8Array): Transaction[T];
    /**
     * Returns an object with the JSON representation of the transaction
     */
    toJSON(): JsonTx;
    protected abstract _processSignature(v: bigint, r: Uint8Array, s: Uint8Array): Transaction[T];
    /**
     * Does chain ID checks on common and returns a common
     * to be used on instantiation
     * @hidden
     *
     * @param common - {@link Common} instance from tx options
     * @param chainId - Chain ID from tx options (typed txs) or signature (legacy tx)
     */
    protected _getCommon(common?: Common, chainId?: BigIntLike): Common;
    /**
     * Validates that an object with BigInt values cannot exceed the specified bit limit.
     * @param values Object containing string keys and BigInt values
     * @param bits Number of bits to check (64 or 256)
     * @param cannotEqual Pass true if the number also cannot equal one less the maximum value
     */
    protected _validateCannotExceedMaxInteger(values: {
        [key: string]: bigint | undefined;
    }, bits?: number, cannotEqual?: boolean): void;
    protected static _validateNotArray(values: {
        [key: string]: any;
    }): void;
    /**
     * Return a compact error string representation of the object
     */
    abstract errorStr(): string;
    /**
     * Internal helper function to create an annotated error message
     *
     * @param msg Base error message
     * @hidden
     */
    protected abstract _errorMsg(msg: string): string;
    /**
     * Returns the shared error postfix part for _error() method
     * tx type implementations.
     */
    protected _getSharedErrorPostfix(): string;
}
//# sourceMappingURL=baseTransaction.d.ts.map