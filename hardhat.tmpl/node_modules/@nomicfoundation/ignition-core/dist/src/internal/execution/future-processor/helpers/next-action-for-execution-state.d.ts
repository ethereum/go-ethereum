import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState, StaticCallExecutionState } from "../../types/execution-state";
/**
 * The next action that the FutureProcessor should take.
 */
export declare enum NextAction {
    /**
     * This action is used when the latest network interaction was completed
     * and the execution strategy should be run again, to understand how to
     * proceed.
     */
    RUN_STRATEGY = "RUN_STRATEGY",
    /**
     * This action is used when the latest network interaction is an OnchainInteraction
     * that requires sending a new transaction.
     */
    SEND_TRANSACTION = "SEND_TRANSACTION",
    /**
     * This action is used when the latest network interaction is a StaticCall that
     * hasn't been run yet.
     */
    QUERY_STATIC_CALL = "QUERY_STATIC_CALL",
    /**
     * This action is used when the latest network interaction is an OnchainInteraction
     * that has one or more in-flight transactions, and we need to monitor them.
     */
    MONITOR_ONCHAIN_INTERACTION = "MONITOR_ONCHAIN_INTERACTION"
}
/**
 * Returns the next action to be run for an execution state.
 */
export declare function nextActionForExecutionState(exState: DeploymentExecutionState | CallExecutionState | SendDataExecutionState | StaticCallExecutionState): NextAction;
//# sourceMappingURL=next-action-for-execution-state.d.ts.map