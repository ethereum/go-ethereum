import { DeploymentState } from "../../types/deployment-state";
import { WipeExecutionStateMessage } from "../../types/messages";
/**
 * Removes an existing execution state from the deployment state.
 *
 * @param state - The deployment state.
 * @param message - The message containing the info of the execution state to remove.
 * @returns - a copy of the deployment state with the execution state removed.
 */
export declare function wipeExecutionState(deploymentState: DeploymentState, message: WipeExecutionStateMessage): DeploymentState;
//# sourceMappingURL=deployment-state-helpers.d.ts.map