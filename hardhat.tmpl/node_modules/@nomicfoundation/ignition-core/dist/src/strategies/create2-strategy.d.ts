import { DeploymentLoader } from "../internal/deployment-loader/types";
import { JsonRpcClient } from "../internal/execution/jsonrpc-client";
import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState, StaticCallExecutionState } from "../internal/execution/types/execution-state";
import { CallStrategyGenerator, DeploymentStrategyGenerator, ExecutionStrategy, SendDataStrategyGenerator, StaticCallStrategyGenerator } from "../internal/execution/types/execution-strategy";
/**
 * The create2 strategy extends the basic strategy, for deployment it replaces
 * a deployment transaction with a call to the CreateX factory contract
 * with a user provided salt.
 *
 * If deploying to the local Hardhat node, the CreateX factory will be
 * deployed if it does not exist. If the CreateX factory is not currently
 * available on the remote network, an error will be thrown halting the
 * deployment.
 *
 * Futures that perform calls or send data remain single transactions, and
 * static calls remain a single static call.
 *
 * The strategy requires a salt is provided in the Hardhat config. The same
 * salt will be used for all calls to CreateX.
 *
 * @example
 * {
 *   ...,
 *   ignition: {
 *     strategyConfig: {
 *       create2: {
 *         salt: "my-salt"
 *       }
 *     }
 *   },
 *   ...
 * }
 *
 * @beta
 */
export declare class Create2Strategy implements ExecutionStrategy {
    readonly name: string;
    readonly config: {
        salt: string;
    };
    private _deploymentLoader;
    private _jsonRpcClient;
    constructor(config: {
        salt: string;
    });
    init(deploymentLoader: DeploymentLoader, jsonRpcClient: JsonRpcClient): Promise<void>;
    executeDeployment(executionState: DeploymentExecutionState): DeploymentStrategyGenerator;
    executeCall(executionState: CallExecutionState): CallStrategyGenerator;
    executeSendData(executionState: SendDataExecutionState): SendDataStrategyGenerator;
    executeStaticCall(executionState: StaticCallExecutionState): StaticCallStrategyGenerator;
    /**
     * Within the context of a local development Hardhat chain, deploy
     * the CreateX factory contract using a presigned transaction.
     */
    private _deployCreateXFactory;
}
//# sourceMappingURL=create2-strategy.d.ts.map