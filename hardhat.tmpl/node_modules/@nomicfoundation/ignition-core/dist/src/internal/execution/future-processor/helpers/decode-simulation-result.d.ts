import { SimulationErrorExecutionResult, StrategySimulationErrorExecutionResult } from "../../types/execution-result";
import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState } from "../../types/execution-state";
import { CallStrategyGenerator, DeploymentStrategyGenerator } from "../../types/execution-strategy";
import { RawStaticCallResult } from "../../types/jsonrpc";
export declare function decodeSimulationResult(strategyGenerator: DeploymentStrategyGenerator | CallStrategyGenerator, exState: DeploymentExecutionState | CallExecutionState | SendDataExecutionState): (simulationResult: RawStaticCallResult) => Promise<SimulationErrorExecutionResult | StrategySimulationErrorExecutionResult | undefined>;
//# sourceMappingURL=decode-simulation-result.d.ts.map