import { Numbers } from 'web3-types';
import type { TypedTransaction } from '../types.js';
import type { TxData, TxOptions } from './types.js';
import { BaseTransaction } from './baseTransaction.js';
export declare class TransactionFactory {
    private constructor();
    static typeToInt(txType: Numbers): number;
    static registerTransactionType<NewTxTypeClass extends typeof BaseTransaction<unknown>>(type: Numbers, txClass: NewTxTypeClass): void;
    /**
     * Create a transaction from a `txData` object
     *
     * @param txData - The transaction data. The `type` field will determine which transaction type is returned (if undefined, creates a legacy transaction)
     * @param txOptions - Options to pass on to the constructor of the transaction
     */
    static fromTxData(txData: TxData | TypedTransaction, txOptions?: TxOptions): TypedTransaction;
    /**
     * This method tries to decode serialized data.
     *
     * @param data - The data Uint8Array
     * @param txOptions - The transaction options
     */
    static fromSerializedData(data: Uint8Array, txOptions?: TxOptions): TypedTransaction;
    /**
     * When decoding a BlockBody, in the transactions field, a field is either:
     * A Uint8Array (a TypedTransaction - encoded as TransactionType || rlp(TransactionPayload))
     * A Uint8Array[] (Legacy Transaction)
     * This method returns the right transaction.
     *
     * @param data - A Uint8Array or Uint8Array[]
     * @param txOptions - The transaction options
     */
    static fromBlockBodyData(data: Uint8Array | Uint8Array[], txOptions?: TxOptions): TypedTransaction;
}
