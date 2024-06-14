"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.wipeExecutionState = void 0;
const immer_1 = require("immer");
const assertions_1 = require("../../../utils/assertions");
/**
 * Removes an existing execution state from the deployment state.
 *
 * @param state - The deployment state.
 * @param message - The message containing the info of the execution state to remove.
 * @returns - a copy of the deployment state with the execution state removed.
 */
function wipeExecutionState(deploymentState, message) {
    return (0, immer_1.produce)(deploymentState, (draft) => {
        (0, assertions_1.assertIgnitionInvariant)(draft.executionStates[message.futureId] !== undefined, `ExecutionState ${message.futureId} must exist to be wiped.`);
        delete draft.executionStates[message.futureId];
    });
}
exports.wipeExecutionState = wipeExecutionState;
//# sourceMappingURL=deployment-state-helpers.js.map