"use strict";
/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
Object.defineProperty(exports, "__esModule", { value: true });
exports.formatTransaction = formatTransaction;
const web3_types_1 = require("web3-types");
const web3_validator_1 = require("web3-validator");
const web3_utils_1 = require("web3-utils");
const web3_errors_1 = require("web3-errors");
const schemas_js_1 = require("../schemas.js");
function formatTransaction(transaction, returnFormat = web3_types_1.DEFAULT_RETURN_FORMAT, options = {
    transactionSchema: schemas_js_1.transactionInfoSchema,
    fillInputAndData: false,
}) {
    var _a, _b;
    let formattedTransaction = (0, web3_utils_1.mergeDeep)({}, transaction);
    if (!(0, web3_validator_1.isNullish)(transaction === null || transaction === void 0 ? void 0 : transaction.common)) {
        formattedTransaction.common = Object.assign({}, transaction.common);
        if (!(0, web3_validator_1.isNullish)((_a = transaction.common) === null || _a === void 0 ? void 0 : _a.customChain))
            formattedTransaction.common.customChain = Object.assign({}, transaction.common.customChain);
    }
    formattedTransaction = (0, web3_utils_1.format)((_b = options.transactionSchema) !== null && _b !== void 0 ? _b : schemas_js_1.transactionInfoSchema, formattedTransaction, returnFormat);
    if (!(0, web3_validator_1.isNullish)(formattedTransaction.data) &&
        !(0, web3_validator_1.isNullish)(formattedTransaction.input) &&
        // Converting toHex is accounting for data and input being Uint8Arrays
        // since comparing Uint8Array is not as straightforward as comparing strings
        (0, web3_utils_1.toHex)(formattedTransaction.data) !== (0, web3_utils_1.toHex)(formattedTransaction.input))
        throw new web3_errors_1.TransactionDataAndInputError({
            data: (0, web3_utils_1.bytesToHex)(formattedTransaction.data),
            input: (0, web3_utils_1.bytesToHex)(formattedTransaction.input),
        });
    if (options.fillInputAndData) {
        if (!(0, web3_validator_1.isNullish)(formattedTransaction.data)) {
            formattedTransaction.input = formattedTransaction.data;
        }
        else if (!(0, web3_validator_1.isNullish)(formattedTransaction.input)) {
            formattedTransaction.data = formattedTransaction.input;
        }
    }
    if (!(0, web3_validator_1.isNullish)(formattedTransaction.gasLimit)) {
        formattedTransaction.gas = formattedTransaction.gasLimit;
        delete formattedTransaction.gasLimit;
    }
    return formattedTransaction;
}
//# sourceMappingURL=format_transaction.js.map