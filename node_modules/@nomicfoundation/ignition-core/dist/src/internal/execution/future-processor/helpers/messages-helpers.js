"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createExecutionStateCompleteMessageForExecutionsWithOnchainInteractions = exports.createExecutionStateCompleteMessage = void 0;
const execution_state_1 = require("../../types/execution-state");
const messages_1 = require("../../types/messages");
/**
 * Creates a message indicating that an execution state is now complete.
 *
 * IMPORTANT NOTE: This function is NOT type-safe. It's the caller's responsibility
 * to ensure that the result is of the correct type.
 *
 * @param exState The completed execution state.
 * @param result The result of the execution.
 * @returns The completion message.
 */
function createExecutionStateCompleteMessage(exState, result) {
    if (exState.type === execution_state_1.ExecutionSateType.STATIC_CALL_EXECUTION_STATE) {
        return {
            type: messages_1.JournalMessageType.STATIC_CALL_EXECUTION_STATE_COMPLETE,
            futureId: exState.id,
            result: result,
        };
    }
    return createExecutionStateCompleteMessageForExecutionsWithOnchainInteractions(exState, result);
}
exports.createExecutionStateCompleteMessage = createExecutionStateCompleteMessage;
/**
 * Creates a message indicating that an execution state is now complete for
 * execution states that require onchain interactions.
 *
 * IMPORTANT NOTE: This function is NOT type-safe. It's the caller's responsibility
 * to ensure that the result is of the correct type.
 *
 * @param exState The completed execution state.
 * @param result The result of the execution.
 * @returns The completion message.
 */
function createExecutionStateCompleteMessageForExecutionsWithOnchainInteractions(exState, result) {
    switch (exState.type) {
        case execution_state_1.ExecutionSateType.DEPLOYMENT_EXECUTION_STATE:
            return {
                type: messages_1.JournalMessageType.DEPLOYMENT_EXECUTION_STATE_COMPLETE,
                futureId: exState.id,
                result: result,
            };
        case execution_state_1.ExecutionSateType.CALL_EXECUTION_STATE:
            return {
                type: messages_1.JournalMessageType.CALL_EXECUTION_STATE_COMPLETE,
                futureId: exState.id,
                result: result,
            };
        case execution_state_1.ExecutionSateType.SEND_DATA_EXECUTION_STATE:
            return {
                type: messages_1.JournalMessageType.SEND_DATA_EXECUTION_STATE_COMPLETE,
                futureId: exState.id,
                result: result,
            };
    }
}
exports.createExecutionStateCompleteMessageForExecutionsWithOnchainInteractions = createExecutionStateCompleteMessageForExecutionsWithOnchainInteractions;
//# sourceMappingURL=messages-helpers.js.map