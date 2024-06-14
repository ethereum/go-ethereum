import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState, StaticCallExecutionState } from "../../types/execution-state";
import { ExecutionStrategy } from "../../types/execution-strategy";
import { CallExecutionStateCompleteMessage, DeploymentExecutionStateCompleteMessage, NetworkInteractionRequestMessage, SendDataExecutionStateCompleteMessage, StaticCallExecutionStateCompleteMessage } from "../../types/messages";
/**
 * Runs the strategy for the execution state, and returns a message that can be
 * a network interaction request, or an execution state complete message.
 *
 * Execution state complete messages can be a result of running the strategy,
 * or of the transaction executing the latest network interaction having reverted.
 *
 * SIDE EFFECTS: This function doesn't have any side effects.
 *
 * @param exState The execution state that requires the strategy to be run.
 * @param executionStrategy The execution strategy to use.
 * @returns A message indicating the result of running the strategy or a reverted tx.
 */
export declare function runStrategy(exState: DeploymentExecutionState | CallExecutionState | SendDataExecutionState | StaticCallExecutionState, executionStrategy: ExecutionStrategy): Promise<NetworkInteractionRequestMessage | DeploymentExecutionStateCompleteMessage | CallExecutionStateCompleteMessage | SendDataExecutionStateCompleteMessage | StaticCallExecutionStateCompleteMessage>;
//# sourceMappingURL=run-strategy.d.ts.map