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
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getTransactionError = getTransactionError;
const web3_errors_1 = require("web3-errors");
// eslint-disable-next-line import/no-cycle
const get_revert_reason_js_1 = require("./get_revert_reason.js");
function getTransactionError(web3Context, transactionFormatted, transactionReceiptFormatted, receivedError, contractAbi, knownReason) {
    return __awaiter(this, void 0, void 0, function* () {
        let _reason = knownReason;
        if (_reason === undefined) {
            if (receivedError !== undefined) {
                _reason = (0, get_revert_reason_js_1.parseTransactionError)(receivedError);
            }
            else if (web3Context.handleRevert && transactionFormatted !== undefined) {
                _reason = yield (0, get_revert_reason_js_1.getRevertReason)(web3Context, transactionFormatted, contractAbi);
            }
        }
        let error;
        if (_reason === undefined) {
            error = new web3_errors_1.TransactionRevertedWithoutReasonError(transactionReceiptFormatted);
        }
        else if (typeof _reason === 'string') {
            error = new web3_errors_1.TransactionRevertInstructionError(_reason, undefined, transactionReceiptFormatted);
        }
        else if (_reason.customErrorName !== undefined &&
            _reason.customErrorDecodedSignature !== undefined &&
            _reason.customErrorArguments !== undefined) {
            const reasonWithCustomError = _reason;
            error = new web3_errors_1.TransactionRevertWithCustomError(reasonWithCustomError.reason, reasonWithCustomError.customErrorName, reasonWithCustomError.customErrorDecodedSignature, reasonWithCustomError.customErrorArguments, reasonWithCustomError.signature, transactionReceiptFormatted, reasonWithCustomError.data);
        }
        else {
            error = new web3_errors_1.TransactionRevertInstructionError(_reason.reason, _reason.signature, transactionReceiptFormatted, _reason.data);
        }
        return error;
    });
}
//# sourceMappingURL=get_transaction_error.js.map