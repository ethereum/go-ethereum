"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isBatchFinished = void 0;
const execution_state_1 = require("../execution/types/execution-state");
/**
 * Have the futures making up a batch finished executing, as defined by
 * no longer being `STARTED`, so they have succeeded, failed, or timed out.
 *
 * @param deploymentState - the deployment state
 * @param batch - the list of future ids of the futures in the batch
 * @returns true if all futures in the batch have finished executing
 */
function isBatchFinished(deploymentState, batch) {
    return batch
        .map((futureId) => deploymentState.executionStates[futureId])
        .every((exState) => exState !== undefined && exState.status !== execution_state_1.ExecutionStatus.STARTED);
}
exports.isBatchFinished = isBatchFinished;
//# sourceMappingURL=is-batch-finished.js.map