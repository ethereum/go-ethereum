"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.hasExecutionSucceeded = void 0;
const execution_state_1 = require("../execution/types/execution-state");
/**
 * Returns true if the execution of the given future has succeeded.
 *
 * @param future The future.
 * @param deploymentState The deployment state to check against.
 * @returns true if it succeeded.
 */
function hasExecutionSucceeded(future, deploymentState) {
    const exState = deploymentState.executionStates[future.id];
    if (exState === undefined) {
        return false;
    }
    return exState.status === execution_state_1.ExecutionStatus.SUCCESS;
}
exports.hasExecutionSucceeded = hasExecutionSucceeded;
//# sourceMappingURL=has-execution-succeeded.js.map