import { ExecutionState } from "./execution-state";
/**
 * A map of execution states indexed by their id.
 */
export interface ExecutionStateMap {
    [key: string]: ExecutionState;
}
/**
 * The execution state of an entire deployment.
 */
export interface DeploymentState {
    chainId: number;
    executionStates: ExecutionStateMap;
}
//# sourceMappingURL=deployment-state.d.ts.map