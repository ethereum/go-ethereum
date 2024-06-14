import { ExecutionState } from "../execution/types/execution-state";
/**
 * Determine if an execution state has reached completion, either
 * completing successfully or failing or timing out.
 *
 * @param exState - the execution state
 * @returns true if the execution state is complete, false if it does
 * not exist or is not complete
 */
export declare function isExecutionStateComplete(exState: ExecutionState): boolean;
//# sourceMappingURL=is-execution-state-complete.d.ts.map