import { bytesToHex, format, hexToBytes, keccak256 } from 'web3-utils';
import { TransactionFactory } from 'web3-eth-accounts';
import { detectRawTransactionType } from './detect_transaction_type.js';
import { formatTransaction } from './format_transaction.js';
/**
 * Decodes an [RLP](https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/#top) encoded transaction.
 *
 * @param encodedSignedTransaction The RLP encoded transaction.
 * @param returnFormat ({@link DataFormat} Specifies how the return data should be formatted.
 * @returns {@link SignedTransactionInfoAPI}, an object containing the RLP encoded signed transaction (accessed via the `raw` property) and the signed transaction object (accessed via the `tx` property).
 */
export function decodeSignedTransaction(encodedSignedTransaction, returnFormat, options = {
    fillInputAndData: false,
}) {
    return {
        raw: format({ format: 'bytes' }, encodedSignedTransaction, returnFormat),
        tx: formatTransaction(Object.assign(Object.assign({}, TransactionFactory.fromSerializedData(hexToBytes(encodedSignedTransaction)).toJSON()), { hash: bytesToHex(keccak256(hexToBytes(encodedSignedTransaction))), type: detectRawTransactionType(hexToBytes(encodedSignedTransaction)) }), returnFormat, {
            fillInputAndData: options.fillInputAndData,
            transactionSchema: options.transactionSchema,
        }),
    };
}
//# sourceMappingURL=decode_signed_transaction.js.map