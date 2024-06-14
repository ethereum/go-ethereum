import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState, StaticCallExecutionState } from "../../types/execution-state";
import { CallExecutionStateCompleteMessage, DeploymentExecutionStateCompleteMessage, SendDataExecutionStateCompleteMessage, StaticCallExecutionStateCompleteMessage } from "../../types/messages";
/**
 * Update the execution state for a future to complete.
 *
 * This can be done generically currently because all execution states
 * excluding contractAt and readEventArg have a result property, and
 * contractAt and readEventArg are initialized completed.
 *
 * @param state - the execution state that will be completed
 * @param message - the execution state specific completion message
 * @returns - a copy of the execution state with the result and status updated
 */
export declare function completeExecutionState<ExState extends StaticCallExecutionState | SendDataExecutionState | CallExecutionState | DeploymentExecutionState>(state: ExState, message: StaticCallExecutionStateCompleteMessage | SendDataExecutionStateCompleteMessage | CallExecutionStateCompleteMessage | DeploymentExecutionStateCompleteMessage): ExState;
//# sourceMappingURL=complete-execution-state.d.ts.map