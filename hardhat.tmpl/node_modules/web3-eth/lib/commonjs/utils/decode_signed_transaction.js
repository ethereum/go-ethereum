"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.decodeSignedTransaction = decodeSignedTransaction;
const web3_utils_1 = require("web3-utils");
const web3_eth_accounts_1 = require("web3-eth-accounts");
const detect_transaction_type_js_1 = require("./detect_transaction_type.js");
const format_transaction_js_1 = require("./format_transaction.js");
/**
 * Decodes an [RLP](https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/#top) encoded transaction.
 *
 * @param encodedSignedTransaction The RLP encoded transaction.
 * @param returnFormat ({@link DataFormat} Specifies how the return data should be formatted.
 * @returns {@link SignedTransactionInfoAPI}, an object containing the RLP encoded signed transaction (accessed via the `raw` property) and the signed transaction object (accessed via the `tx` property).
 */
function decodeSignedTransaction(encodedSignedTransaction, returnFormat, options = {
    fillInputAndData: false,
}) {
    return {
        raw: (0, web3_utils_1.format)({ format: 'bytes' }, encodedSignedTransaction, returnFormat),
        tx: (0, format_transaction_js_1.formatTransaction)(Object.assign(Object.assign({}, web3_eth_accounts_1.TransactionFactory.fromSerializedData((0, web3_utils_1.hexToBytes)(encodedSignedTransaction)).toJSON()), { hash: (0, web3_utils_1.bytesToHex)((0, web3_utils_1.keccak256)((0, web3_utils_1.hexToBytes)(encodedSignedTransaction))), type: (0, detect_transaction_type_js_1.detectRawTransactionType)((0, web3_utils_1.hexToBytes)(encodedSignedTransaction)) }), returnFormat, {
            fillInputAndData: options.fillInputAndData,
            transactionSchema: options.transactionSchema,
        }),
    };
}
//# sourceMappingURL=decode_signed_transaction.js.map