import { DeploymentLoader } from "../deployment-loader/types";
import { DeploymentState } from "./types/deployment-state";
import { JournalMessage } from "./types/messages";
/**
 * Loads a previous deployment state from its existing messages.
 * @param messages An async iterator of journal messages.
 * @returns The deployment state or undefined if no messages were provided.
 */
export declare function loadDeploymentState(deploymentLoader: DeploymentLoader): Promise<DeploymentState | undefined>;
/**
 * Ininitalizes the deployment state and records the run start message to the journal.
 *
 * @param chainId The chain ID.
 * @param deploymentLoader The deployment loader that will be used to record the message.
 * @returns The new DeploymentState.
 */
export declare function initializeDeploymentState(chainId: number, deploymentLoader: DeploymentLoader): Promise<DeploymentState>;
/**
 * This function applies a new message to the deployment state, recording it to the
 * journal if needed.
 *
 * @param message The message to apply.
 * @param deploymentState The original deployment state.
 * @param deploymentLoader The deployment loader that will be used to record the message.
 * @returns The new deployment state.
 */
export declare function applyNewMessage(message: JournalMessage, deploymentState: DeploymentState, deploymentLoader: DeploymentLoader): Promise<DeploymentState>;
/**
 * Returns true if a message should be recorded to the jorunal.
 */
export declare function shouldBeJournaled(message: JournalMessage): boolean;
//# sourceMappingURL=deployment-state-helpers.d.ts.map