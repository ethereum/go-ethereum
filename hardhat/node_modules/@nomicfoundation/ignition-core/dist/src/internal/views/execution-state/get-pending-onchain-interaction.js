"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getPendingOnchainInteraction = void 0;
const execution_state_1 = require("../../execution/types/execution-state");
const network_interaction_1 = require("../../execution/types/network-interaction");
const assertions_1 = require("../../utils/assertions");
/**
 * Returns the last NetworkInteraction if there's one and it's an
 * OnchainInteraction without a confirmed transaction.
 *
 * @param exState The execution state to check.
 * @returns Returns the pending nonce and sender if the last network interaction
 *  was a transaction, and it hasn't been been confirmed yet.
 */
function getPendingOnchainInteraction(exState) {
    if (exState.type === execution_state_1.ExecutionStateType.STATIC_CALL_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.READ_EVENT_ARGUMENT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.CONTRACT_AT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.ENCODE_FUNCTION_CALL_EXECUTION_STATE) {
        return undefined;
    }
    const interaction = exState.networkInteractions.at(-1);
    (0, assertions_1.assertIgnitionInvariant)(interaction !== undefined, `Unable to find network interaction for ${exState.id} when trying to get pending onchain interaction`);
    if (interaction.type === network_interaction_1.NetworkInteractionType.STATIC_CALL ||
        interaction.transactions.some((tx) => tx.receipt !== undefined)) {
        return undefined;
    }
    return interaction;
}
exports.getPendingOnchainInteraction = getPendingOnchainInteraction;
//# sourceMappingURL=get-pending-onchain-interaction.js.map