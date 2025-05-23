"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isExecutionStateComplete = void 0;
const execution_state_1 = require("../execution/types/execution-state");
/**
 * Determine if an execution state has reached completion, either
 * completing successfully or failing or timing out.
 *
 * @param exState - the execution state
 * @returns true if the execution state is complete, false if it does
 * not exist or is not complete
 */
function isExecutionStateComplete(exState) {
    return exState.status !== execution_state_1.ExecutionStatus.STARTED;
}
exports.isExecutionStateComplete = isExecutionStateComplete;
//# sourceMappingURL=is-execution-state-complete.js.map