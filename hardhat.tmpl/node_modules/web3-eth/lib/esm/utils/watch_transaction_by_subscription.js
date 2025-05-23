var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
import { format } from 'web3-utils';
import { transactionReceiptSchema } from '../schemas.js';
import { watchTransactionByPolling } from './watch_transaction_by_polling.js';
/**
 * This function watches a Transaction by subscribing to new heads.
 * It is used by `watchTransactionForConfirmations`, in case the provider supports subscription.
 */
export const watchTransactionBySubscription = ({ web3Context, transactionReceipt, transactionPromiEvent, customTransactionReceiptSchema, returnFormat, }) => {
    // The following variable will stay true except if the data arrived,
    //	or if watching started after an error had occurred.
    let needToWatchLater = true;
    let lastCaughtBlockHash;
    setImmediate(() => {
        var _a;
        (_a = web3Context.subscriptionManager) === null || _a === void 0 ? void 0 : _a.subscribe('newHeads').then((subscription) => {
            subscription.on('data', (newBlockHeader) => __awaiter(void 0, void 0, void 0, function* () {
                var _a;
                needToWatchLater = false;
                if (!(newBlockHeader === null || newBlockHeader === void 0 ? void 0 : newBlockHeader.number) ||
                    // For some cases, the on-data event is fired couple times for the same block!
                    // This needs investigation but seems to be because of multiple `subscription.on('data'...)` even this should not cause that.
                    lastCaughtBlockHash === (newBlockHeader === null || newBlockHeader === void 0 ? void 0 : newBlockHeader.parentHash)) {
                    return;
                }
                lastCaughtBlockHash = newBlockHeader === null || newBlockHeader === void 0 ? void 0 : newBlockHeader.parentHash;
                const confirmations = BigInt(newBlockHeader.number) -
                    BigInt(transactionReceipt.blockNumber) +
                    BigInt(1);
                transactionPromiEvent.emit('confirmation', {
                    confirmations: format({ format: 'uint' }, confirmations, returnFormat),
                    receipt: format(customTransactionReceiptSchema !== null && customTransactionReceiptSchema !== void 0 ? customTransactionReceiptSchema : transactionReceiptSchema, transactionReceipt, returnFormat),
                    latestBlockHash: format({ format: 'bytes32' }, newBlockHeader.parentHash, returnFormat),
                });
                if (confirmations >= web3Context.transactionConfirmationBlocks) {
                    yield ((_a = web3Context.subscriptionManager) === null || _a === void 0 ? void 0 : _a.removeSubscription(subscription));
                }
            }));
            subscription.on('error', () => __awaiter(void 0, void 0, void 0, function* () {
                var _a;
                yield ((_a = web3Context.subscriptionManager) === null || _a === void 0 ? void 0 : _a.removeSubscription(subscription));
                needToWatchLater = false;
                watchTransactionByPolling({
                    web3Context,
                    transactionReceipt,
                    transactionPromiEvent,
                    customTransactionReceiptSchema,
                    returnFormat,
                });
            }));
        }).catch(() => {
            needToWatchLater = false;
            watchTransactionByPolling({
                web3Context,
                transactionReceipt,
                customTransactionReceiptSchema,
                transactionPromiEvent,
                returnFormat,
            });
        });
    });
    // Fallback to polling if tx receipt didn't arrived in "blockHeaderTimeout" [10 seconds]
    setTimeout(() => {
        if (needToWatchLater) {
            watchTransactionByPolling({
                web3Context,
                transactionReceipt,
                transactionPromiEvent,
                returnFormat,
            });
        }
    }, web3Context.blockHeaderTimeout * 1000);
};
//# sourceMappingURL=watch_transaction_by_subscription.js.map