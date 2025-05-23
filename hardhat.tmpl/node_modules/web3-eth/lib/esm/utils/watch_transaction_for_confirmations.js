import { format } from 'web3-utils';
import { isNullish } from 'web3-validator';
import { TransactionMissingReceiptOrBlockHashError, TransactionReceiptMissingBlockNumberError, } from 'web3-errors';
import { transactionReceiptSchema } from '../schemas.js';
import { watchTransactionByPolling, } from './watch_transaction_by_polling.js';
import { watchTransactionBySubscription } from './watch_transaction_by_subscription.js';
export function watchTransactionForConfirmations(web3Context, transactionPromiEvent, transactionReceipt, transactionHash, returnFormat, customTransactionReceiptSchema) {
    if (isNullish(transactionReceipt) || isNullish(transactionReceipt.blockHash))
        throw new TransactionMissingReceiptOrBlockHashError({
            receipt: transactionReceipt,
            blockHash: format({ format: 'bytes32' }, transactionReceipt === null || transactionReceipt === void 0 ? void 0 : transactionReceipt.blockHash, returnFormat),
            transactionHash: format({ format: 'bytes32' }, transactionHash, returnFormat),
        });
    if (!transactionReceipt.blockNumber)
        throw new TransactionReceiptMissingBlockNumberError({ receipt: transactionReceipt });
    // As we have the receipt, it's the first confirmation that tx is accepted.
    transactionPromiEvent.emit('confirmation', {
        confirmations: format({ format: 'uint' }, 1, returnFormat),
        receipt: format(customTransactionReceiptSchema !== null && customTransactionReceiptSchema !== void 0 ? customTransactionReceiptSchema : transactionReceiptSchema, transactionReceipt, returnFormat),
        latestBlockHash: format({ format: 'bytes32' }, transactionReceipt.blockHash, returnFormat),
    });
    // so a subscription for newBlockHeaders can be made instead of polling
    const provider = web3Context.requestManager.provider;
    if (provider && 'supportsSubscriptions' in provider && provider.supportsSubscriptions()) {
        watchTransactionBySubscription({
            web3Context,
            transactionReceipt,
            transactionPromiEvent,
            customTransactionReceiptSchema,
            returnFormat,
        });
    }
    else {
        watchTransactionByPolling({
            web3Context,
            transactionReceipt,
            transactionPromiEvent,
            customTransactionReceiptSchema,
            returnFormat,
        });
    }
}
//# sourceMappingURL=watch_transaction_for_confirmations.js.map