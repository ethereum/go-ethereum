import { ArtifactResolver } from "../../types/artifact";
import { DeploymentParameters } from "../../types/deploy";
import { ExecutionEventListener } from "../../types/execution-events";
import { IgnitionModule, IgnitionModuleResult } from "../../types/module";
import { DeploymentLoader } from "../deployment-loader/types";
import { JsonRpcClient } from "./jsonrpc-client";
import { DeploymentState } from "./types/deployment-state";
import { ExecutionStrategy } from "./types/execution-strategy";
/**
 * This class is used to execute a module to completion, returning the new
 * deployment state.
 */
export declare class ExecutionEngine {
    private readonly _deploymentLoader;
    private readonly _artifactResolver;
    private readonly _executionStrategy;
    private readonly _jsonRpcClient;
    private readonly _executionEventListener;
    private readonly _requiredConfirmations;
    private readonly _millisecondBeforeBumpingFees;
    private readonly _maxFeeBumps;
    private readonly _blockPollingInterval;
    constructor(_deploymentLoader: DeploymentLoader, _artifactResolver: ArtifactResolver, _executionStrategy: ExecutionStrategy, _jsonRpcClient: JsonRpcClient, _executionEventListener: ExecutionEventListener | undefined, _requiredConfirmations: number, _millisecondBeforeBumpingFees: number, _maxFeeBumps: number, _blockPollingInterval: number);
    /**
     * Executes a module to completion, returning the new deployment state.
     *
     * This functions saves to the journal any created message, and stores
     * artifacts and successful deployment addresses in the deployment folder.
     *
     * @param deploymentState The existing deployment state.
     * @param module The module to execute.
     * @param batches The result of batching the futures of the module.
     * @param accounts The accounts to use for executing the module.
     * @param deploymentParameters The deployment parameters provided by the user.
     * @param defaultSender The default sender to use as `from` of futures, transactions and static calls.
     * @returns The new deployment state.
     */
    executeModule(deploymentState: DeploymentState, module: IgnitionModule<string, string, IgnitionModuleResult<string>>, batches: string[][], accounts: string[], deploymentParameters: DeploymentParameters, defaultSender: string): Promise<DeploymentState>;
    /**
     * Executes a batch of futures until all of its futures are completed.
     *
     * @param futureProcessor The FutureProcessor to use for executing the futures.
     * @param batch The batch of futures to execute.
     * @param deploymentState The current deployment state.
     * @returns The new deployment state.
     */
    private _executeBatch;
    /**
     * Returns a promise that only resolves when the next block is available,
     * and returns it.
     *
     * This function polls the network every `_blockPollingInterval` milliseconds.
     *
     * @param previousBlock The previous block that we know of, to compare from
     *  the one we get grom the network.
     * @returns The new block.
     */
    private _waitForNextBlock;
    /**
     * Syncs the nonces of the deployment state with the blockchain, returning
     * the new deployment state, and throwing if they can't be synced.
     *
     * This method processes dropped and replaced transactions.
     *
     * @param deploymentState The existing deployment state.
     * @param ignitionModule The module that will be executed.
     * @returns The updated deployment state.
     */
    private _syncNonces;
    /**
     * Returns a future by its id.
     */
    private _lookupFuture;
    /**
     * Returns the batch sorted by the highest the pending nonce of each future
     * and sender.
     *
     * Futures without any pending nonce come last.
     */
    private _getBatchSortedByHighesPendingNonce;
    /**
     * Emits an execution event signaling that execution of the next batch has begun.
     */
    private _emitBeginNextBatchEvent;
}
//# sourceMappingURL=execution-engine.d.ts.map