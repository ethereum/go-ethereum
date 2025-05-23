"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Deployer = void 0;
const errors_1 = require("../errors");
const type_guards_1 = require("../type-guards");
const deploy_1 = require("../types/deploy");
const execution_events_1 = require("../types/execution-events");
const batcher_1 = require("./batcher");
const errors_list_1 = require("./errors-list");
const deployment_state_helpers_1 = require("./execution/deployment-state-helpers");
const execution_engine_1 = require("./execution/execution-engine");
const execution_state_1 = require("./execution/types/execution-state");
const reconciler_1 = require("./reconciliation/reconciler");
const assertions_1 = require("./utils/assertions");
const get_futures_from_module_1 = require("./utils/get-futures-from-module");
const find_deployed_contracts_1 = require("./views/find-deployed-contracts");
const find_status_1 = require("./views/find-status");
/**
 * Run an Igntition deployment.
 */
class Deployer {
    _config;
    _deploymentDir;
    _executionStrategy;
    _jsonRpcClient;
    _artifactResolver;
    _deploymentLoader;
    _executionEventListener;
    constructor(_config, _deploymentDir, _executionStrategy, _jsonRpcClient, _artifactResolver, _deploymentLoader, _executionEventListener) {
        this._config = _config;
        this._deploymentDir = _deploymentDir;
        this._executionStrategy = _executionStrategy;
        this._jsonRpcClient = _jsonRpcClient;
        this._artifactResolver = _artifactResolver;
        this._deploymentLoader = _deploymentLoader;
        this._executionEventListener = _executionEventListener;
        (0, assertions_1.assertIgnitionInvariant)(this._config.requiredConfirmations >= 1, `Configured value 'requiredConfirmations' cannot be less than 1. Value given: '${this._config.requiredConfirmations}'`);
    }
    async deploy(ignitionModule, deploymentParameters, accounts, defaultSender) {
        const deployment = await this._getOrInitializeDeploymentState();
        const isResumed = deployment.isResumed;
        let deploymentState = deployment.deploymentState;
        this._emitDeploymentStartEvent(ignitionModule.id, this._deploymentDir, isResumed, this._config.maxFeeBumps, this._config.disableFeeBumping);
        const contracts = (0, get_futures_from_module_1.getFuturesFromModule)(ignitionModule).filter(type_guards_1.isContractFuture);
        const contractStates = contracts
            .map((contract) => deploymentState?.executionStates[contract.id])
            .filter((v) => v !== undefined);
        // realistically this should be impossible to fail.
        // just need it here for the type inference
        (0, assertions_1.assertIgnitionInvariant)(contractStates.every((exState) => exState.type === execution_state_1.ExecutionStateType.DEPLOYMENT_EXECUTION_STATE ||
            exState.type === execution_state_1.ExecutionStateType.CONTRACT_AT_EXECUTION_STATE), "Invalid state map");
        const reconciliationResult = await reconciler_1.Reconciler.reconcile(ignitionModule, deploymentState, deploymentParameters, accounts, this._deploymentLoader, this._artifactResolver, defaultSender, this._executionStrategy.name, this._executionStrategy.config);
        if (reconciliationResult.reconciliationFailures.length > 0) {
            const errors = {};
            for (const { futureId, failure, } of reconciliationResult.reconciliationFailures) {
                if (errors[futureId] === undefined) {
                    errors[futureId] = [];
                }
                errors[futureId].push(failure);
            }
            const reconciliationErrorResult = {
                type: deploy_1.DeploymentResultType.RECONCILIATION_ERROR,
                errors,
            };
            this._emitDeploymentCompleteEvent(reconciliationErrorResult);
            return reconciliationErrorResult;
        }
        const previousRunErrors = reconciler_1.Reconciler.checkForPreviousRunErrors(deploymentState);
        if (previousRunErrors.length > 0) {
            const errors = {};
            for (const { futureId, failure } of previousRunErrors) {
                if (errors[futureId] === undefined) {
                    errors[futureId] = [];
                }
                errors[futureId].push(failure);
            }
            const previousRunErrorResult = {
                type: deploy_1.DeploymentResultType.PREVIOUS_RUN_ERROR,
                errors,
            };
            this._emitDeploymentCompleteEvent(previousRunErrorResult);
            return previousRunErrorResult;
        }
        if (reconciliationResult.missingExecutedFutures.length > 0) {
            this._emitReconciliationWarningsEvent(reconciliationResult.missingExecutedFutures);
        }
        const batches = batcher_1.Batcher.batch(ignitionModule, deploymentState);
        this._emitDeploymentBatchEvent(batches);
        if (this._hasBatchesToExecute(batches)) {
            this._emitRunStartEvent();
            const executionEngine = new execution_engine_1.ExecutionEngine(this._deploymentLoader, this._artifactResolver, this._executionStrategy, this._jsonRpcClient, this._executionEventListener, this._config.requiredConfirmations, this._config.timeBeforeBumpingFees, this._config.maxFeeBumps, this._config.blockPollingInterval, this._config.disableFeeBumping);
            deploymentState = await executionEngine.executeModule(deploymentState, ignitionModule, batches, accounts, deploymentParameters, defaultSender);
        }
        const result = await this._getDeploymentResult(deploymentState, ignitionModule);
        this._emitDeploymentCompleteEvent(result);
        return result;
    }
    async _getDeploymentResult(deploymentState, _module) {
        if (!this._isSuccessful(deploymentState)) {
            return this._getExecutionErrorResult(deploymentState);
        }
        const deployedContracts = (0, find_deployed_contracts_1.findDeployedContracts)(deploymentState);
        return {
            type: deploy_1.DeploymentResultType.SUCCESSFUL_DEPLOYMENT,
            contracts: deployedContracts,
        };
    }
    /**
     * Fetches the existing deployment state or initializes a new one.
     *
     * @returns An object with the deployment state and a boolean indicating
     * if the deployment is being resumed (i.e. the deployment state is not
     * new).
     */
    async _getOrInitializeDeploymentState() {
        const chainId = await this._jsonRpcClient.getChainId();
        const deploymentState = await (0, deployment_state_helpers_1.loadDeploymentState)(this._deploymentLoader);
        if (deploymentState === undefined) {
            const newState = await (0, deployment_state_helpers_1.initializeDeploymentState)(chainId, this._deploymentLoader);
            return { deploymentState: newState, isResumed: false };
        }
        // TODO: this should be moved out, it is not obvious that a significant
        // check is being done in an init method
        if (deploymentState.chainId !== chainId) {
            throw new errors_1.IgnitionError(errors_list_1.ERRORS.DEPLOY.CHANGED_CHAINID, {
                previousChainId: deploymentState.chainId,
                currentChainId: chainId,
            });
        }
        return { deploymentState, isResumed: true };
    }
    _emitDeploymentStartEvent(moduleId, deploymentDir, isResumed, maxFeeBumps, disableFeeBumping) {
        if (this._executionEventListener === undefined) {
            return;
        }
        this._executionEventListener.deploymentStart({
            type: execution_events_1.ExecutionEventType.DEPLOYMENT_START,
            moduleName: moduleId,
            deploymentDir: deploymentDir ?? undefined,
            isResumed,
            maxFeeBumps,
            disableFeeBumping,
        });
    }
    _emitReconciliationWarningsEvent(warnings) {
        if (this._executionEventListener === undefined) {
            return;
        }
        this._executionEventListener.reconciliationWarnings({
            type: execution_events_1.ExecutionEventType.RECONCILIATION_WARNINGS,
            warnings,
        });
    }
    _emitDeploymentBatchEvent(batches) {
        if (this._executionEventListener === undefined) {
            return;
        }
        this._executionEventListener.batchInitialize({
            type: execution_events_1.ExecutionEventType.BATCH_INITIALIZE,
            batches,
        });
    }
    _emitRunStartEvent() {
        if (this._executionEventListener === undefined) {
            return;
        }
        this._executionEventListener.runStart({
            type: execution_events_1.ExecutionEventType.RUN_START,
        });
    }
    _emitDeploymentCompleteEvent(result) {
        if (this._executionEventListener === undefined) {
            return;
        }
        this._executionEventListener.deploymentComplete({
            type: execution_events_1.ExecutionEventType.DEPLOYMENT_COMPLETE,
            result,
        });
    }
    _isSuccessful(deploymentState) {
        return Object.values(deploymentState.executionStates).every((ex) => ex.status === execution_state_1.ExecutionStatus.SUCCESS);
    }
    _getExecutionErrorResult(deploymentState) {
        const status = (0, find_status_1.findStatus)(deploymentState);
        return {
            type: deploy_1.DeploymentResultType.EXECUTION_ERROR,
            ...status,
        };
    }
    /**
     * Determine if an execution run is necessary.
     *
     * @param batches - the batches to be executed
     * @returns if there are batches to be executed
     */
    _hasBatchesToExecute(batches) {
        return batches.length > 0;
    }
}
exports.Deployer = Deployer;
//# sourceMappingURL=deployer.js.map