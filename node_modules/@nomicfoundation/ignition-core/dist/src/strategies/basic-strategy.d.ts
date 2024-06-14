import { DeploymentLoader } from "../internal/deployment-loader/types";
import { JsonRpcClient } from "../internal/execution/jsonrpc-client";
import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState, StaticCallExecutionState } from "../internal/execution/types/execution-state";
import { CallStrategyGenerator, DeploymentStrategyGenerator, ExecutionStrategy, SendDataStrategyGenerator, StaticCallStrategyGenerator } from "../internal/execution/types/execution-strategy";
/**
 * The basic execution strategy, which sends a single transaction
 * for each contract deployment, call, and send data, and a single static call
 * for each static call execution.
 *
 * @private
 */
export declare class BasicStrategy implements ExecutionStrategy {
    readonly name: string;
    readonly config: Record<PropertyKey, never>;
    private _deploymentLoader;
    constructor();
    init(deploymentLoader: DeploymentLoader, _jsonRpcClient: JsonRpcClient): Promise<void>;
    executeDeployment(executionState: DeploymentExecutionState): DeploymentStrategyGenerator;
    executeCall(executionState: CallExecutionState): CallStrategyGenerator;
    executeSendData(executionState: SendDataExecutionState): SendDataStrategyGenerator;
    executeStaticCall(executionState: StaticCallExecutionState): StaticCallStrategyGenerator;
}
//# sourceMappingURL=basic-strategy.d.ts.map