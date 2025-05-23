"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.watchTransactionForConfirmations = watchTransactionForConfirmations;
const web3_utils_1 = require("web3-utils");
const web3_validator_1 = require("web3-validator");
const web3_errors_1 = require("web3-errors");
const schemas_js_1 = require("../schemas.js");
const watch_transaction_by_polling_js_1 = require("./watch_transaction_by_polling.js");
const watch_transaction_by_subscription_js_1 = require("./watch_transaction_by_subscription.js");
function watchTransactionForConfirmations(web3Context, transactionPromiEvent, transactionReceipt, transactionHash, returnFormat, customTransactionReceiptSchema) {
    if ((0, web3_validator_1.isNullish)(transactionReceipt) || (0, web3_validator_1.isNullish)(transactionReceipt.blockHash))
        throw new web3_errors_1.TransactionMissingReceiptOrBlockHashError({
            receipt: transactionReceipt,
            blockHash: (0, web3_utils_1.format)({ format: 'bytes32' }, transactionReceipt === null || transactionReceipt === void 0 ? void 0 : transactionReceipt.blockHash, returnFormat),
            transactionHash: (0, web3_utils_1.format)({ format: 'bytes32' }, transactionHash, returnFormat),
        });
    if (!transactionReceipt.blockNumber)
        throw new web3_errors_1.TransactionReceiptMissingBlockNumberError({ receipt: transactionReceipt });
    // As we have the receipt, it's the first confirmation that tx is accepted.
    transactionPromiEvent.emit('confirmation', {
        confirmations: (0, web3_utils_1.format)({ format: 'uint' }, 1, returnFormat),
        receipt: (0, web3_utils_1.format)(customTransactionReceiptSchema !== null && customTransactionReceiptSchema !== void 0 ? customTransactionReceiptSchema : schemas_js_1.transactionReceiptSchema, transactionReceipt, returnFormat),
        latestBlockHash: (0, web3_utils_1.format)({ format: 'bytes32' }, transactionReceipt.blockHash, returnFormat),
    });
    // so a subscription for newBlockHeaders can be made instead of polling
    const provider = web3Context.requestManager.provider;
    if (provider && 'supportsSubscriptions' in provider && provider.supportsSubscriptions()) {
        (0, watch_transaction_by_subscription_js_1.watchTransactionBySubscription)({
            web3Context,
            transactionReceipt,
            transactionPromiEvent,
            customTransactionReceiptSchema,
            returnFormat,
        });
    }
    else {
        (0, watch_transaction_by_polling_js_1.watchTransactionByPolling)({
            web3Context,
            transactionReceipt,
            transactionPromiEvent,
            customTransactionReceiptSchema,
            returnFormat,
        });
    }
}
//# sourceMappingURL=watch_transaction_for_confirmations.js.map