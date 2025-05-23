import { BaseTransaction } from './baseTransaction.js';
import type { Common } from '../common/common.js';
import type { JsonTx, TxData, TxOptions, TxValuesArray } from './types.js';
/**
 * An Ethereum non-typed (legacy) transaction
 */
export declare class Transaction extends BaseTransaction<Transaction> {
    readonly gasPrice: bigint;
    readonly common: Common;
    /**
     * Instantiate a transaction from a data dictionary.
     *
     * Format: { nonce, gasPrice, gasLimit, to, value, data, v, r, s }
     *
     * Notes:
     * - All parameters are optional and have some basic default values
     */
    static fromTxData(txData: TxData, opts?: TxOptions): Transaction;
    /**
     * Instantiate a transaction from the serialized tx.
     *
     * Format: `rlp([nonce, gasPrice, gasLimit, to, value, data, v, r, s])`
     */
    static fromSerializedTx(serialized: Uint8Array, opts?: TxOptions): Transaction;
    /**
     * Create a transaction from a values array.
     *
     * Format: `[nonce, gasPrice, gasLimit, to, value, data, v, r, s]`
     */
    static fromValuesArray(values: TxValuesArray, opts?: TxOptions): Transaction;
    /**
     * This constructor takes the values, validates them, assigns them and freezes the object.
     *
     * It is not recommended to use this constructor directly. Instead use
     * the static factory methods to assist in creating a Transaction object from
     * varying data types.
     */
    constructor(txData: TxData, opts?: TxOptions);
    /**
     * Returns a Uint8Array Array of the raw Uint8Arrays of the legacy transaction, in order.
     *
     * Format: `[nonce, gasPrice, gasLimit, to, value, data, v, r, s]`
     *
     * For legacy txs this is also the correct format to add transactions
     * to a block with {@link Block.fromValuesArray} (use the `serialize()` method
     * for typed txs).
     *
     * For an unsigned tx this method returns the empty Uint8Array values
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
    private _getMessageToSign;
    /**
     * Returns the unsigned tx (hashed or raw), which can be used
     * to sign the transaction (e.g. for sending to a hardware wallet).
     *
     * Note: the raw message message format for the legacy tx is not RLP encoded
     * and you might need to do yourself with:
     *
     * ```javascript
     * import { bufArrToArr } from '../util'
     * import { RLP } from '../rlp'
     * const message = tx.getMessageToSign(false)
     * const serializedMessage = RLP.encode(message) // use this for the HW wallet input
     * ```
     *
     * @param hashMessage - Return hashed message if set to true (default: true)
     */
    getMessageToSign(hashMessage: false): Uint8Array[];
    getMessageToSign(hashMessage?: true): Uint8Array;
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
    getSenderPublicKey(): Uint8Array;
    /**
     * Process the v, r, s values from the `sign` method of the base transaction.
     */
    protected _processSignature(_v: bigint, r: Uint8Array, s: Uint8Array): Transaction;
    /**
     * Returns an object with the JSON representation of the transaction.
     */
    toJSON(): JsonTx;
    /**
     * Validates tx's `v` value
     */
    private _validateTxV;
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
