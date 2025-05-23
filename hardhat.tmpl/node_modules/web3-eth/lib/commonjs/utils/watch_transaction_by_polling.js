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
exports.watchTransactionByPolling = void 0;
const web3_utils_1 = require("web3-utils");
const web3_rpc_methods_1 = require("web3-rpc-methods");
const schemas_js_1 = require("../schemas.js");
/**
 * This function watches a Transaction by subscribing to new heads.
 * It is used by `watchTransactionForConfirmations`, in case the provider does not support subscription.
 * And it is also used by `watchTransactionBySubscription`, as a fallback, if the subscription failed for any reason.
 */
const watchTransactionByPolling = ({ web3Context, transactionReceipt, transactionPromiEvent, customTransactionReceiptSchema, returnFormat, }) => {
    var _a;
    // Having a transactionReceipt means that the transaction has already been included
    // in at least one block, so we start with 1
    let confirmations = 1;
    const intervalId = setInterval(() => {
        (() => __awaiter(void 0, void 0, void 0, function* () {
            if (confirmations >= web3Context.transactionConfirmationBlocks) {
                clearInterval(intervalId);
                return;
            }
            const nextBlock = yield web3_rpc_methods_1.ethRpcMethods.getBlockByNumber(web3Context.requestManager, (0, web3_utils_1.numberToHex)(BigInt(transactionReceipt.blockNumber) + BigInt(confirmations)), false);
            if (nextBlock === null || nextBlock === void 0 ? void 0 : nextBlock.hash) {
                confirmations += 1;
                transactionPromiEvent.emit('confirmation', {
                    confirmations: (0, web3_utils_1.format)({ format: 'uint' }, confirmations, returnFormat),
                    receipt: (0, web3_utils_1.format)(customTransactionReceiptSchema !== null && customTransactionReceiptSchema !== void 0 ? customTransactionReceiptSchema : schemas_js_1.transactionReceiptSchema, transactionReceipt, returnFormat),
                    latestBlockHash: (0, web3_utils_1.format)({ format: 'bytes32' }, nextBlock.hash, returnFormat),
                });
            }
        }))();
    }, (_a = web3Context.transactionReceiptPollingInterval) !== null && _a !== void 0 ? _a : web3Context.transactionPollingInterval);
};
exports.watchTransactionByPolling = watchTransactionByPolling;
//# sourceMappingURL=watch_transaction_by_polling.js.map