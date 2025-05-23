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
import { DEFAULT_RETURN_FORMAT } from 'web3-types';
import { isNullish } from 'web3-validator';
import { mergeDeep, format, bytesToHex, toHex } from 'web3-utils';
import { TransactionDataAndInputError } from 'web3-errors';
import { transactionInfoSchema } from '../schemas.js';
export function formatTransaction(transaction, returnFormat = DEFAULT_RETURN_FORMAT, options = {
    transactionSchema: transactionInfoSchema,
    fillInputAndData: false,
}) {
    var _a, _b;
    let formattedTransaction = mergeDeep({}, transaction);
    if (!isNullish(transaction === null || transaction === void 0 ? void 0 : transaction.common)) {
        formattedTransaction.common = Object.assign({}, transaction.common);
        if (!isNullish((_a = transaction.common) === null || _a === void 0 ? void 0 : _a.customChain))
            formattedTransaction.common.customChain = Object.assign({}, transaction.common.customChain);
    }
    formattedTransaction = format((_b = options.transactionSchema) !== null && _b !== void 0 ? _b : transactionInfoSchema, formattedTransaction, returnFormat);
    if (!isNullish(formattedTransaction.data) &&
        !isNullish(formattedTransaction.input) &&
        // Converting toHex is accounting for data and input being Uint8Arrays
        // since comparing Uint8Array is not as straightforward as comparing strings
        toHex(formattedTransaction.data) !== toHex(formattedTransaction.input))
        throw new TransactionDataAndInputError({
            data: bytesToHex(formattedTransaction.data),
            input: bytesToHex(formattedTransaction.input),
        });
    if (options.fillInputAndData) {
        if (!isNullish(formattedTransaction.data)) {
            formattedTransaction.input = formattedTransaction.data;
        }
        else if (!isNullish(formattedTransaction.input)) {
            formattedTransaction.data = formattedTransaction.input;
        }
    }
    if (!isNullish(formattedTransaction.gasLimit)) {
        formattedTransaction.gas = formattedTransaction.gasLimit;
        delete formattedTransaction.gasLimit;
    }
    return formattedTransaction;
}
//# sourceMappingURL=format_transaction.js.map