"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.findResultForFutureById = void 0;
const execution_result_1 = require("../execution/types/execution-result");
const execution_state_1 = require("../execution/types/execution-state");
const assertions_1 = require("../utils/assertions");
function findResultForFutureById(deploymentState, futureId) {
    const exState = deploymentState.executionStates[futureId];
    (0, assertions_1.assertIgnitionInvariant)(exState !== undefined, `Expected execution state for ${futureId} to exist, but it did not`);
    (0, assertions_1.assertIgnitionInvariant)(exState.type === execution_state_1.ExecutionStateType.DEPLOYMENT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.STATIC_CALL_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.CONTRACT_AT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.READ_EVENT_ARGUMENT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.ENCODE_FUNCTION_CALL_EXECUTION_STATE, `Expected execution state for ${futureId} to be support result lookup, but instead it was ${exState.type}`);
    if (exState.type === execution_state_1.ExecutionStateType.CONTRACT_AT_EXECUTION_STATE) {
        return exState.contractAddress;
    }
    (0, assertions_1.assertIgnitionInvariant)(exState.result !== undefined, `Expected execution state for ${futureId} to have a result, but it did not`);
    if (exState.type === execution_state_1.ExecutionStateType.READ_EVENT_ARGUMENT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionStateType.ENCODE_FUNCTION_CALL_EXECUTION_STATE) {
        return exState.result;
    }
    (0, assertions_1.assertIgnitionInvariant)(exState.result.type === execution_result_1.ExecutionResultType.SUCCESS, `Cannot access the result of ${futureId}, it was not a deployment success or static call success`);
    switch (exState.type) {
        case execution_state_1.ExecutionStateType.DEPLOYMENT_EXECUTION_STATE:
            return exState.result.address;
        case execution_state_1.ExecutionStateType.STATIC_CALL_EXECUTION_STATE: {
            return exState.result.value;
        }
    }
}
exports.findResultForFutureById = findResultForFutureById;
//# sourceMappingURL=find-result-for-future-by-id.js.map