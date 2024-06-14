import { DeploymentState } from "../execution/types/deployment-state";
/**
 * Have the futures making up a batch finished executing, as defined by
 * no longer being `STARTED`, so they have succeeded, failed, or timed out.
 *
 * @param deploymentState - the deployment state
 * @param batch - the list of future ids of the futures in the batch
 * @returns true if all futures in the batch have finished executing
 */
export declare function isBatchFinished(deploymentState: DeploymentState, batch: string[]): boolean;
//# sourceMappingURL=is-batch-finished.d.ts.map