var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
import { rejectIfConditionAtInterval } from 'web3-utils';
import { TransactionBlockTimeoutError } from 'web3-errors';
import { NUMBER_DATA_FORMAT } from '../constants.js';
// eslint-disable-next-line import/no-cycle
import { getBlockNumber } from '../rpc_method_wrappers.js';
function resolveByPolling(web3Context, starterBlockNumber, transactionHash) {
    const pollingInterval = web3Context.transactionPollingInterval;
    const [intervalId, promiseToError] = rejectIfConditionAtInterval(() => __awaiter(this, void 0, void 0, function* () {
        let lastBlockNumber;
        try {
            lastBlockNumber = yield getBlockNumber(web3Context, NUMBER_DATA_FORMAT);
        }
        catch (error) {
            console.warn('An error happen while trying to get the block number', error);
            return undefined;
        }
        const numberOfBlocks = lastBlockNumber - starterBlockNumber;
        if (numberOfBlocks >= web3Context.transactionBlockTimeout) {
            return new TransactionBlockTimeoutError({
                starterBlockNumber,
                numberOfBlocks,
                transactionHash,
            });
        }
        return undefined;
    }), pollingInterval);
    const clean = () => {
        clearInterval(intervalId);
    };
    return [promiseToError, { clean }];
}
function resolveBySubscription(web3Context, starterBlockNumber, transactionHash) {
    return __awaiter(this, void 0, void 0, function* () {
        var _a;
        // The following variable will stay true except if the data arrived,
        //	or if watching started after an error had occurred.
        let needToWatchLater = true;
        let subscription;
        let resourceCleaner;
        // internal helper function
        function revertToPolling(reject, previousError) {
            if (previousError) {
                console.warn('error happened at subscription. So revert to polling...', previousError);
            }
            resourceCleaner.clean();
            needToWatchLater = false;
            const [promiseToError, newResourceCleaner] = resolveByPolling(web3Context, starterBlockNumber, transactionHash);
            resourceCleaner.clean = newResourceCleaner.clean;
            promiseToError.catch(error => reject(error));
        }
        try {
            subscription = (yield ((_a = web3Context.subscriptionManager) === null || _a === void 0 ? void 0 : _a.subscribe('newHeads')));
            resourceCleaner = {
                clean: () => {
                    var _a;
                    // Remove the subscription, if it was not removed somewhere
                    // 	else by calling, for example, subscriptionManager.clear()
                    if (subscription.id) {
                        (_a = web3Context.subscriptionManager) === null || _a === void 0 ? void 0 : _a.removeSubscription(subscription).then(() => {
                            // Subscription ended successfully
                        }).catch(() => {
                            // An error happened while ending subscription. But no need to take any action.
                        });
                    }
                },
            };
        }
        catch (error) {
            return resolveByPolling(web3Context, starterBlockNumber, transactionHash);
        }
        const promiseToError = new Promise((_, reject) => {
            try {
                subscription.on('data', (lastBlockHeader) => {
                    needToWatchLater = false;
                    if (!(lastBlockHeader === null || lastBlockHeader === void 0 ? void 0 : lastBlockHeader.number)) {
                        return;
                    }
                    const numberOfBlocks = Number(BigInt(lastBlockHeader.number) - BigInt(starterBlockNumber));
                    if (numberOfBlocks >= web3Context.transactionBlockTimeout) {
                        // Transaction Block Timeout is known to be reached by subscribing to new heads
                        reject(new TransactionBlockTimeoutError({
                            starterBlockNumber,
                            numberOfBlocks,
                            transactionHash,
                        }));
                    }
                });
                subscription.on('error', error => {
                    revertToPolling(reject, error);
                });
            }
            catch (error) {
                revertToPolling(reject, error);
            }
            // Fallback to polling if tx receipt didn't arrived in "blockHeaderTimeout" [10 seconds]
            setTimeout(() => {
                if (needToWatchLater) {
                    revertToPolling(reject);
                }
            }, web3Context.blockHeaderTimeout * 1000);
        });
        return [promiseToError, resourceCleaner];
    });
}
/* TODO: After merge, there will be constant block mining time (exactly 12 second each block, except slot missed that currently happens in <1% of slots. ) so we can optimize following function
for POS NWs, we can skip checking getBlockNumber(); after interval and calculate only based on time  that certain num of blocked are mined after that for internal double check, can do one getBlockNumber() call and timeout.
*/
export function rejectIfBlockTimeout(web3Context, transactionHash) {
    return __awaiter(this, void 0, void 0, function* () {
        var _a, _b;
        const { provider } = web3Context.requestManager;
        let callingRes;
        const starterBlockNumber = yield getBlockNumber(web3Context, NUMBER_DATA_FORMAT);
        // TODO: once https://github.com/web3/web3.js/issues/5521 is implemented, remove checking for `enableExperimentalFeatures.useSubscriptionWhenCheckingBlockTimeout`
        if (((_b = (_a = provider).supportsSubscriptions) === null || _b === void 0 ? void 0 : _b.call(_a)) &&
            web3Context.enableExperimentalFeatures.useSubscriptionWhenCheckingBlockTimeout) {
            // eslint-disable-next-line @typescript-eslint/no-floating-promises
            callingRes = yield resolveBySubscription(web3Context, starterBlockNumber, transactionHash);
        }
        else {
            // eslint-disable-next-line @typescript-eslint/no-floating-promises
            callingRes = resolveByPolling(web3Context, starterBlockNumber, transactionHash);
        }
        return callingRes;
    });
}
//# sourceMappingURL=reject_if_block_timeout.js.map