import { DeploymentExecutionResult, CallExecutionResult, SendDataExecutionResult, StaticCallExecutionResult } from "../../types/execution-result";
import { DeploymentExecutionState, CallExecutionState, SendDataExecutionState, StaticCallExecutionState } from "../../types/execution-state";
import { DeploymentExecutionStateCompleteMessage, CallExecutionStateCompleteMessage, SendDataExecutionStateCompleteMessage, StaticCallExecutionStateCompleteMessage } from "../../types/messages";
/**
 * Creates a message indicating that an execution state is now complete.
 *
 * IMPORTANT NOTE: This function is NOT type-safe. It's the caller's responsibility
 * to ensure that the result is of the correct type.
 *
 * @param exState The completed execution state.
 * @param result The result of the execution.
 * @returns The completion message.
 */
export declare function createExecutionStateCompleteMessage(exState: DeploymentExecutionState | CallExecutionState | SendDataExecutionState | StaticCallExecutionState, result: DeploymentExecutionResult | CallExecutionResult | SendDataExecutionResult | StaticCallExecutionResult): DeploymentExecutionStateCompleteMessage | CallExecutionStateCompleteMessage | SendDataExecutionStateCompleteMessage | StaticCallExecutionStateCompleteMessage;
/**
 * Creates a message indicating that an execution state is now complete for
 * execution states that require onchain interactions.
 *
 * IMPORTANT NOTE: This function is NOT type-safe. It's the caller's responsibility
 * to ensure that the result is of the correct type.
 *
 * @param exState The completed execution state.
 * @param result The result of the execution.
 * @returns The completion message.
 */
export declare function createExecutionStateCompleteMessageForExecutionsWithOnchainInteractions(exState: DeploymentExecutionState | CallExecutionState | SendDataExecutionState, result: DeploymentExecutionResult | CallExecutionResult | SendDataExecutionResult): DeploymentExecutionStateCompleteMessage | CallExecutionStateCompleteMessage | SendDataExecutionStateCompleteMessage;
//# sourceMappingURL=messages-helpers.d.ts.map