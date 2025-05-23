var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
import { format, numberToHex } from 'web3-utils';
import { ethRpcMethods } from 'web3-rpc-methods';
import { transactionReceiptSchema } from '../schemas.js';
/**
 * This function watches a Transaction by subscribing to new heads.
 * It is used by `watchTransactionForConfirmations`, in case the provider does not support subscription.
 * And it is also used by `watchTransactionBySubscription`, as a fallback, if the subscription failed for any reason.
 */
export const watchTransactionByPolling = ({ web3Context, transactionReceipt, transactionPromiEvent, customTransactionReceiptSchema, returnFormat, }) => {
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
            const nextBlock = yield ethRpcMethods.getBlockByNumber(web3Context.requestManager, numberToHex(BigInt(transactionReceipt.blockNumber) + BigInt(confirmations)), false);
            if (nextBlock === null || nextBlock === void 0 ? void 0 : nextBlock.hash) {
                confirmations += 1;
                transactionPromiEvent.emit('confirmation', {
                    confirmations: format({ format: 'uint' }, confirmations, returnFormat),
                    receipt: format(customTransactionReceiptSchema !== null && customTransactionReceiptSchema !== void 0 ? customTransactionReceiptSchema : transactionReceiptSchema, transactionReceipt, returnFormat),
                    latestBlockHash: format({ format: 'bytes32' }, nextBlock.hash, returnFormat),
                });
            }
        }))();
    }, (_a = web3Context.transactionReceiptPollingInterval) !== null && _a !== void 0 ? _a : web3Context.transactionPollingInterval);
};
//# sourceMappingURL=watch_transaction_by_polling.js.map