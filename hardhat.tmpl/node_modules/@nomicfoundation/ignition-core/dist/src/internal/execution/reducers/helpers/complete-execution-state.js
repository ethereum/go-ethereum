"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.completeExecutionState = void 0;
const immer_1 = require("immer");
const execution_result_1 = require("../../types/execution-result");
const execution_state_1 = require("../../types/execution-state");
/**
 * Update the execution state for a future to complete.
 *
 * This can be done generically currently because all execution states
 * excluding contractAt and readEventArg have a result property, and
 * contractAt and readEventArg are initialized completed.
 *
 * @param state - the execution state that will be completed
 * @param message - the execution state specific completion message
 * @returns - a copy of the execution state with the result and status updated
 */
function completeExecutionState(state, message) {
    return (0, immer_1.produce)(state, (draft) => {
        draft.status = _mapResultTypeToStatus(message.result);
        draft.result = message.result;
    });
}
exports.completeExecutionState = completeExecutionState;
function _mapResultTypeToStatus(result) {
    switch (result.type) {
        case execution_result_1.ExecutionResultType.SUCCESS:
            return execution_state_1.ExecutionStatus.SUCCESS;
        case execution_result_1.ExecutionResultType.SIMULATION_ERROR:
            return execution_state_1.ExecutionStatus.FAILED;
        case execution_result_1.ExecutionResultType.STRATEGY_SIMULATION_ERROR:
            return execution_state_1.ExecutionStatus.FAILED;
        case execution_result_1.ExecutionResultType.REVERTED_TRANSACTION:
            return execution_state_1.ExecutionStatus.FAILED;
        case execution_result_1.ExecutionResultType.STATIC_CALL_ERROR:
            return execution_state_1.ExecutionStatus.FAILED;
        case execution_result_1.ExecutionResultType.STRATEGY_ERROR:
            return execution_state_1.ExecutionStatus.FAILED;
        case execution_result_1.ExecutionResultType.STRATEGY_HELD:
            return execution_state_1.ExecutionStatus.HELD;
    }
}
//# sourceMappingURL=complete-execution-state.js.map