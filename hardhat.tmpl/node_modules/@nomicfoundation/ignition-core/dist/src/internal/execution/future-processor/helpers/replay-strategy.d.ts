import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState, StaticCallExecutionState } from "../../types/execution-state";
import { CallStrategyGenerator, DeploymentStrategyGenerator, ExecutionStrategy, SendDataStrategyGenerator, StaticCallStrategyGenerator } from "../../types/execution-strategy";
/**
 * This function returns a strategy generator for the executionState that has been replayed
 * up to the request that lead to the last network interaction of the exectionState being
 * created.
 *
 * IMPORTANT: This function is NOT type-safe. It is the responsibility of the caller to
 * interpret the returned generator as the correct type. This is allows us to have a single
 * function replay all the different types of execution states.
 *
 * @param executionState The execution state.
 * @param strategy The strategy to use to create the generator.
 * @returns The replayed strategy generator.
 */
export declare function replayStrategy(executionState: DeploymentExecutionState | CallExecutionState | SendDataExecutionState | StaticCallExecutionState, strategy: ExecutionStrategy): Promise<DeploymentStrategyGenerator | CallStrategyGenerator | SendDataStrategyGenerator | StaticCallStrategyGenerator>;
//# sourceMappingURL=replay-strategy.d.ts.map