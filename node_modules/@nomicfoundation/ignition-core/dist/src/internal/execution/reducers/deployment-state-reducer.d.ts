import { DeploymentState } from "../types/deployment-state";
import { JournalMessage } from "../types/messages";
/**
 * The root level reducer for the overall deployment state.
 *
 * @param state - the deployment state
 * @param action - a message that can be journaled
 * @returns a copy of the deployment state with the message applied
 */
export declare function deploymentStateReducer(state?: DeploymentState, action?: JournalMessage): DeploymentState;
//# sourceMappingURL=deployment-state-reducer.d.ts.map