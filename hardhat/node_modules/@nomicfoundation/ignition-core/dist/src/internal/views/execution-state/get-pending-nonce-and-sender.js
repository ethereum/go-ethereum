"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getPendingNonceAndSender = void 0;
const execution_state_1 = require("../../execution/types/execution-state");
const get_pending_onchain_interaction_1 = require("./get-pending-onchain-interaction");
/**
 * Returns the nonce and sender of a pending transaction of the execution state,
 * if any.
 *
 * @param exState The execution state to check.
 * @returns Returns the nonce and sender of the last (and only) pending tx
 *  of the execution state, if any.
 */
function getPendingNonceAndSender(exState) {
    if (exState.type === execution_state_1.ExecutionStateType.READ_EVENT_ARGUMENT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.CONTRACT_AT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.ENCODE_FUNCTION_CALL_EXECUTION_STATE) {
        return undefined;
    }
    const interaction = (0, get_pending_onchain_interaction_1.getPendingOnchainInteraction)(exState);
    if (interaction === undefined || interaction.nonce === undefined) {
        return undefined;
    }
    return { nonce: interaction.nonce, sender: exState.from };
}
exports.getPendingNonceAndSender = getPendingNonceAndSender;
//# sourceMappingURL=get-pending-nonce-and-sender.js.map