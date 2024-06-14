import type { IgnitionModule, IgnitionModuleResult } from "../types/module";
import { ArtifactResolver } from "../types/artifact";
import { DeployConfig, DeploymentParameters, DeploymentResult } from "../types/deploy";
import { ExecutionEventListener } from "../types/execution-events";
import { DeploymentLoader } from "./deployment-loader/types";
import { JsonRpcClient } from "./execution/jsonrpc-client";
import { ExecutionStrategy } from "./execution/types/execution-strategy";
/**
 * Run an Igntition deployment.
 */
export declare class Deployer {
    private readonly _config;
    private readonly _deploymentDir;
    private readonly _executionStrategy;
    private readonly _jsonRpcClient;
    private readonly _artifactResolver;
    private readonly _deploymentLoader;
    private readonly _executionEventListener?;
    constructor(_config: DeployConfig, _deploymentDir: string | undefined, _executionStrategy: ExecutionStrategy, _jsonRpcClient: JsonRpcClient, _artifactResolver: ArtifactResolver, _deploymentLoader: DeploymentLoader, _executionEventListener?: ExecutionEventListener | undefined);
    deploy<ModuleIdT extends string, ContractNameT extends string, IgnitionModuleResultsT extends IgnitionModuleResult<ContractNameT>>(ignitionModule: IgnitionModule<ModuleIdT, ContractNameT, IgnitionModuleResultsT>, deploymentParameters: DeploymentParameters, accounts: string[], defaultSender: string): Promise<DeploymentResult>;
    private _getDeploymentResult;
    /**
     * Fetches the existing deployment state or initializes a new one.
     *
     * @returns An object with the deployment state and a boolean indicating
     * if the deployment is being resumed (i.e. the deployment state is not
     * new).
     */
    private _getOrInitializeDeploymentState;
    private _emitDeploymentStartEvent;
    private _emitReconciliationWarningsEvent;
    private _emitDeploymentBatchEvent;
    private _emitRunStartEvent;
    private _emitDeploymentCompleteEvent;
    private _isSuccessful;
    private _getExecutionErrorResult;
    /**
     * Determine if an execution run is necessary.
     *
     * @param batches - the batches to be executed
     * @returns if there are batches to be executed
     */
    private _hasBatchesToExecute;
}
//# sourceMappingURL=deployer.d.ts.map