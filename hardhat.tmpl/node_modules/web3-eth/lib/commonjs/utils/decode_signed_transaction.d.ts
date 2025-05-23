import { DataFormat, HexStringBytes, SignedTransactionInfoAPI } from 'web3-types';
import { type CustomTransactionSchema } from '../types.js';
/**
 * Decodes an [RLP](https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/#top) encoded transaction.
 *
 * @param encodedSignedTransaction The RLP encoded transaction.
 * @param returnFormat ({@link DataFormat} Specifies how the return data should be formatted.
 * @returns {@link SignedTransactionInfoAPI}, an object containing the RLP encoded signed transaction (accessed via the `raw` property) and the signed transaction object (accessed via the `tx` property).
 */
export declare function decodeSignedTransaction<ReturnFormat extends DataFormat>(encodedSignedTransaction: HexStringBytes, returnFormat: ReturnFormat, options?: {
    fillInputAndData?: boolean;
    transactionSchema?: CustomTransactionSchema;
}): SignedTransactionInfoAPI;
