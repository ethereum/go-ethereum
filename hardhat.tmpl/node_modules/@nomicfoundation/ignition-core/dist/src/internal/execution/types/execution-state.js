"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ExecutionStateType = exports.ExecutionStatus = void 0;
/**
 * The different status that the execution can be at.
 */
var ExecutionStatus;
(function (ExecutionStatus) {
    ExecutionStatus["STARTED"] = "STARATED";
    ExecutionStatus["TIMEOUT"] = "TIMEOUT";
    ExecutionStatus["SUCCESS"] = "SUCCESS";
    ExecutionStatus["HELD"] = "HELD";
    ExecutionStatus["FAILED"] = "FAILED";
})(ExecutionStatus = exports.ExecutionStatus || (exports.ExecutionStatus = {}));
/**
 * The different kinds of execution states.
 */
var ExecutionStateType;
(function (ExecutionStateType) {
    ExecutionStateType["DEPLOYMENT_EXECUTION_STATE"] = "DEPLOYMENT_EXECUTION_STATE";
    ExecutionStateType["CALL_EXECUTION_STATE"] = "CALL_EXECUTION_STATE";
    ExecutionStateType["STATIC_CALL_EXECUTION_STATE"] = "STATIC_CALL_EXECUTION_STATE";
    ExecutionStateType["ENCODE_FUNCTION_CALL_EXECUTION_STATE"] = "ENCODE_FUNCTION_CALL_EXECUTION_STATE";
    ExecutionStateType["CONTRACT_AT_EXECUTION_STATE"] = "CONTRACT_AT_EXECUTION_STATE";
    ExecutionStateType["READ_EVENT_ARGUMENT_EXECUTION_STATE"] = "READ_EVENT_ARGUMENT_EXECUTION_STATE";
    ExecutionStateType["SEND_DATA_EXECUTION_STATE"] = "SEND_DATA_EXECUTION_STATE";
})(ExecutionStateType = exports.ExecutionStateType || (exports.ExecutionStateType = {}));
//# sourceMappingURL=execution-state.js.map