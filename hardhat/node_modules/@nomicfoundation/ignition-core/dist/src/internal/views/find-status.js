"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.findStatus = void 0;
const execution_result_1 = require("../execution/types/execution-result");
const execution_state_1 = require("../execution/types/execution-state");
const formatters_1 = require("../formatters");
const assertions_1 = require("../utils/assertions");
function findStatus(deploymentState) {
    const executionStates = Object.values(deploymentState.executionStates);
    return {
        started: executionStates
            .filter((ex) => ex.status === execution_state_1.ExecutionStatus.STARTED)
            .map((ex) => ex.id),
        successful: executionStates
            .filter((ex) => ex.status === execution_state_1.ExecutionStatus.SUCCESS)
            .map((ex) => ex.id),
        held: executionStates
            .filter(canFail)
            .filter((ex) => ex.status === execution_state_1.ExecutionStatus.HELD)
            .map((ex) => {
            (0, assertions_1.assertIgnitionInvariant)(ex.result !== undefined, `Execution state ${ex.id} is marked as held but has no result`);
            (0, assertions_1.assertIgnitionInvariant)(ex.result.type === execution_result_1.ExecutionResultType.STRATEGY_HELD, `Execution state ${ex.id} is marked as held but has ${ex.result.type} instead of a held result`);
            return {
                futureId: ex.id,
                heldId: ex.result.heldId,
                reason: ex.result.reason,
            };
        }),
        timedOut: executionStates
            .filter(canTimeout)
            .filter((ex) => ex.status === execution_state_1.ExecutionStatus.TIMEOUT)
            .map((ex) => ({
            futureId: ex.id,
            networkInteractionId: ex.networkInteractions.at(-1).id,
        })),
        failed: executionStates
            .filter(canFail)
            .filter((ex) => ex.status === execution_state_1.ExecutionStatus.FAILED)
            .map((ex) => {
            (0, assertions_1.assertIgnitionInvariant)(ex.result !== undefined &&
                ex.result.type !== execution_result_1.ExecutionResultType.SUCCESS &&
                ex.result.type !== execution_result_1.ExecutionResultType.STRATEGY_HELD, `Execution state ${ex.id} is marked as failed but has no error result`);
            return {
                futureId: ex.id,
                networkInteractionId: ex.networkInteractions.at(-1).id,
                error: (0, formatters_1.formatExecutionError)(ex.result),
            };
        }),
    };
}
exports.findStatus = findStatus;
// TODO: Does this exist anywhere else? It's in fact just checking if it sends txs
function canTimeout(exState) {
    return (exState.type === execution_state_1.ExecutionStateType.DEPLOYMENT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.CALL_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.SEND_DATA_EXECUTION_STATE);
}
// TODO: Does this exist anywhere else? It's in fact just checking if has network interactions
function canFail(exState) {
    return (exState.type === execution_state_1.ExecutionStateType.DEPLOYMENT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.CALL_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.SEND_DATA_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.STATIC_CALL_EXECUTION_STATE);
}
//# sourceMappingURL=find-status.js.map