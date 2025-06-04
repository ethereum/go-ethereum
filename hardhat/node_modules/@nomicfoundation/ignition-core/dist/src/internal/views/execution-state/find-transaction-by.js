"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.findTransactionBy = void 0;
const assertions_1 = require("../../utils/assertions");
const find_onchain_interaction_by_1 = require("./find-onchain-interaction-by");
function findTransactionBy(executionState, networkInteractionId, hash) {
    const onchainInteraction = (0, find_onchain_interaction_by_1.findOnchainInteractionBy)(executionState, networkInteractionId);
    const transaction = onchainInteraction.transactions.find((tx) => tx.hash === hash);
    (0, assertions_1.assertIgnitionInvariant)(transaction !== undefined, `Expected transaction ${executionState.id}/${networkInteractionId}/${hash} to exist, but it did not`);
    return transaction;
}
exports.findTransactionBy = findTransactionBy;
//# sourceMappingURL=find-transaction-by.js.map