"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getNetworkExecutionStates = void 0;
const execution_state_1 = require("../../execution/types/execution-state");
function getNetworkExecutionStates(deploymentState) {
    const exStates = [];
    for (const exState of Object.values(deploymentState.executionStates)) {
        if (exState.type === execution_state_1.ExecutionStateType.DEPLOYMENT_EXECUTION_STATE ||
            exState.type === execution_state_1.ExecutionStateType.CALL_EXECUTION_STATE ||
            exState.type === execution_state_1.ExecutionStateType.SEND_DATA_EXECUTION_STATE ||
            exState.type === execution_state_1.ExecutionStateType.STATIC_CALL_EXECUTION_STATE) {
            exStates.push(exState);
        }
    }
    return exStates;
}
exports.getNetworkExecutionStates = getNetworkExecutionStates;
//# sourceMappingURL=get-network-execution-states.js.map