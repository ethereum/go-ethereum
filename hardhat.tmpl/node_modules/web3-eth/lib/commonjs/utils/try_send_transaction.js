"use strict";
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
exports.trySendTransaction = trySendTransaction;
const web3_utils_1 = require("web3-utils");
const web3_errors_1 = require("web3-errors");
// eslint-disable-next-line import/no-cycle
const reject_if_block_timeout_js_1 = require("./reject_if_block_timeout.js");
/**
 * An internal function to send a transaction or throws if sending did not finish during the timeout during the blocks-timeout.
 * @param web3Context - the context to read the configurations from
 * @param sendTransactionFunc - the function that will send the transaction (could be sendTransaction or sendRawTransaction)
 * @param transactionHash - to be used inside the exception message if there will be any exceptions.
 * @returns the Promise<string> returned by the `sendTransactionFunc`.
 */
function trySendTransaction(web3Context, sendTransactionFunc, transactionHash) {
    return __awaiter(this, void 0, void 0, function* () {
        const [timeoutId, rejectOnTimeout] = (0, web3_utils_1.rejectIfTimeout)(web3Context.transactionSendTimeout, new web3_errors_1.TransactionSendTimeoutError({
            numberOfSeconds: web3Context.transactionSendTimeout / 1000,
            transactionHash,
        }));
        const [rejectOnBlockTimeout, blockTimeoutResourceCleaner] = yield (0, reject_if_block_timeout_js_1.rejectIfBlockTimeout)(web3Context, transactionHash);
        try {
            // If an error happened here, do not catch it, just clear the resources before raising it to the caller function.
            return yield Promise.race([
                sendTransactionFunc(), // this is the function that will send the transaction
                rejectOnTimeout, // this will throw an error on Transaction Send Timeout
                rejectOnBlockTimeout, // this will throw an error on Transaction Block Timeout
            ]);
        }
        finally {
            clearTimeout(timeoutId);
            blockTimeoutResourceCleaner.clean();
        }
    });
}
//# sourceMappingURL=try_send_transaction.js.map