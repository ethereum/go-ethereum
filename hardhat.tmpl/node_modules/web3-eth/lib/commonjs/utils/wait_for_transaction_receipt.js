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
exports.waitForTransactionReceipt = waitForTransactionReceipt;
const web3_errors_1 = require("web3-errors");
// eslint-disable-next-line import/no-cycle
const web3_utils_1 = require("web3-utils");
// eslint-disable-next-line import/no-cycle
const reject_if_block_timeout_js_1 = require("./reject_if_block_timeout.js");
// eslint-disable-next-line import/no-cycle
const rpc_method_wrappers_js_1 = require("../rpc_method_wrappers.js");
function waitForTransactionReceipt(web3Context, transactionHash, returnFormat, customGetTransactionReceipt) {
    return __awaiter(this, void 0, void 0, function* () {
        var _a;
        const pollingInterval = (_a = web3Context.transactionReceiptPollingInterval) !== null && _a !== void 0 ? _a : web3Context.transactionPollingInterval;
        const [awaitableTransactionReceipt, IntervalId] = (0, web3_utils_1.pollTillDefinedAndReturnIntervalId)(() => __awaiter(this, void 0, void 0, function* () {
            try {
                return (customGetTransactionReceipt !== null && customGetTransactionReceipt !== void 0 ? customGetTransactionReceipt : rpc_method_wrappers_js_1.getTransactionReceipt)(web3Context, transactionHash, returnFormat);
            }
            catch (error) {
                console.warn('An error happen while trying to get the transaction receipt', error);
                return undefined;
            }
        }), pollingInterval);
        const [timeoutId, rejectOnTimeout] = (0, web3_utils_1.rejectIfTimeout)(web3Context.transactionPollingTimeout, new web3_errors_1.TransactionPollingTimeoutError({
            numberOfSeconds: web3Context.transactionPollingTimeout / 1000,
            transactionHash,
        }));
        const [rejectOnBlockTimeout, blockTimeoutResourceCleaner] = yield (0, reject_if_block_timeout_js_1.rejectIfBlockTimeout)(web3Context, transactionHash);
        try {
            // If an error happened here, do not catch it, just clear the resources before raising it to the caller function.
            return yield Promise.race([
                awaitableTransactionReceipt,
                rejectOnTimeout, // this will throw an error on Transaction Polling Timeout
                rejectOnBlockTimeout, // this will throw an error on Transaction Block Timeout
            ]);
        }
        finally {
            if (timeoutId)
                clearTimeout(timeoutId);
            if (IntervalId)
                clearInterval(IntervalId);
            blockTimeoutResourceCleaner.clean();
        }
    });
}
//# sourceMappingURL=wait_for_transaction_receipt.js.map