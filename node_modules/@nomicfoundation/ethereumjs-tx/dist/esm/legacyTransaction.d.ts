import { BaseTransaction } from './baseTransaction.js';
import { TransactionType } from './types.js';
import type { TxData as AllTypesTxData, TxValuesArray as AllTypesTxValuesArray, JsonTx, TxOptions } from './types.js';
import type { Common } from '@nomicfoundation/ethereumjs-common';
declare type TxData = AllTypesTxData[TransactionType.Legacy];
declare type TxValuesArray = AllTypesTxValuesArray[TransactionType.Legacy];
/**
 * An Ethereum non-typed (legacy) transaction
 */
export declare class LegacyTransaction extends BaseTransaction<TransactionType.Legacy> {
    readonly gasPrice: bigint;
    readonly common: Common;
    private keccakFunction;
    /**
     * Instantiate a transaction from a data dictionary.
     *
     * Format: { nonce, gasPrice, gasLimit, to, value, data, v, r, s }
     *
     * Notes:
     * - All parameters are optional and have some basic default values
     */
    static fromTxData(txData: TxData, opts?: TxOptions): LegacyTransaction;
    /**
     * Instantiate a transaction from the serialized tx.
     *
     * Format: `rlp([nonce, gasPrice, gasLimit, to, value, data, v, r, s])`
     */
    static fromSerializedTx(serialized: Uint8Array, opts?: TxOptions): LegacyTransaction;
    /**
     * Create a transaction from a values array.
     *
     * Format: `[nonce, gasPrice, gasLimit, to, value, data, v, r, s]`
     */
    static fromValuesArray(values: TxValuesArray, opts?: TxOptions): LegacyTransaction;
    /**
     * This constructor takes the values, validates them, assigns them and freezes the object.
     *
     * It is not recommended to use this constructor directly. Instead use
     * the static factory methods to assist in creating a Transaction object from
     * varying data types.
     */
    constructor(txData: TxData, opts?: TxOptions);
    /**
     * Returns a Uint8Array Array of the raw Bytes of the legacy transaction, in order.
     *
     * Format: `[nonce, gasPrice, gasLimit, to, value, data, v, r, s]`
     *
     * For legacy txs this is also the correct format to add transactions
     * to a block with {@link Block.fromValuesArray} (use the `serialize()` method
     * for typed txs).
     *
     * For an unsigned tx this method returns the empty Bytes values
     * for the signature parameters `v`, `r` and `s`. For an EIP-155 compliant
     * representation have a look at {@link Transaction.getMessageToSign}.
     */
    raw(): TxValuesArray;
    /**
     * Returns the serialized encoding of the legacy transaction.
     *
     * Format: `rlp([nonce, gasPrice, gasLimit, to, value, data, v, r, s])`
     *
     * For an unsigned tx this method uses the empty Uint8Array values for the
     * signature parameters `v`, `r` and `s` for encoding. For an EIP-155 compliant
     * representation for external signing use {@link Transaction.getMessageToSign}.
     */
    serialize(): Uint8Array;
    /**
     * Returns the raw unsigned tx, which can be used
     * to sign the transaction (e.g. for sending to a hardware wallet).
     *
     * Note: the raw message message format for the legacy tx is not RLP encoded
     * and you might need to do yourself with:
     *
     * ```javascript
     * import { RLP } from '@nomicfoundation/ethereumjs-rlp'
     * const message = tx.getMessageToSign()
     * const serializedMessage = RLP.encode(message)) // use this for the HW wallet input
     * ```
     */
    getMessageToSign(): Uint8Array[];
    /**
     * Returns the hashed serialized unsigned tx, which can be used
     * to sign the transaction (e.g. for sending to a hardware wallet).
     */
    getHashedMessageToSign(): Uint8Array;
    /**
     * The amount of gas paid for the data in this tx
     */
    getDataFee(): bigint;
    /**
     * The up front amount that an account must have for this transaction to be valid
     */
    getUpfrontCost(): bigint;
    /**
     * Computes a sha3-256 hash of the serialized tx.
     *
     * This method can only be used for signed txs (it throws otherwise).
     * Use {@link Transaction.getMessageToSign} to get a tx hash for the purpose of signing.
     */
    hash(): Uint8Array;
    /**
     * Computes a sha3-256 hash which can be used to verify the signature
     */
    getMessageToVerifySignature(): Uint8Array;
    /**
     * Returns the public key of the sender
     */
    _getSenderPublicKey(): Uint8Array;
    /**
     * Process the v, r, s values from the `sign` method of the base transaction.
     */
    protected _processSignature(v: bigint, r: Uint8Array, s: Uint8Array): LegacyTransaction;
    /**
     * Returns an object with the JSON representation of the transaction.
     */
    toJSON(): JsonTx;
    /**
     * Validates tx's `v` value
     */
    protected _validateTxV(_v?: bigint, common?: Common): Common;
    /**
     * Return a compact error string representation of the object
     */
    errorStr(): string;
    /**
     * Internal helper function to create an annotated error message
     *
     * @param msg Base error message
     * @hidden
     */
    protected _errorMsg(msg: string): string;
}
export {};
//# sourceMappingURL=legacyTransaction.d.ts.map