"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deploymentStateReducer = void 0;
const messages_1 = require("../types/messages");
const execution_state_reducer_1 = require("./execution-state-reducer");
const deployment_state_helpers_1 = require("./helpers/deployment-state-helpers");
/**
 * The root level reducer for the overall deployment state.
 *
 * @param state - the deployment state
 * @param action - a message that can be journaled
 * @returns a copy of the deployment state with the message applied
 */
function deploymentStateReducer(state, action) {
    if (state === undefined) {
        state = {
            chainId: -1,
            executionStates: {},
        };
    }
    if (action === undefined) {
        return state;
    }
    if (action.type === messages_1.JournalMessageType.DEPLOYMENT_INITIALIZE) {
        return {
            ...state,
            chainId: action.chainId,
        };
    }
    if (action.type === messages_1.JournalMessageType.WIPE_APPLY) {
        return (0, deployment_state_helpers_1.wipeExecutionState)(state, action);
    }
    const previousExState = state.executionStates[action.futureId];
    return {
        ...state,
        executionStates: {
            ...state.executionStates,
            [action.futureId]: (0, execution_state_reducer_1.executionStateReducer)(previousExState, action),
        },
    };
}
exports.deploymentStateReducer = deploymentStateReducer;
//# sourceMappingURL=deployment-state-reducer.js.map