"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ExecutionEngine = void 0;
const execution_events_1 = require("../../types/execution-events");
const assertions_1 = require("../utils/assertions");
const get_futures_from_module_1 = require("../utils/get-futures-from-module");
const get_pending_nonce_and_sender_1 = require("../views/execution-state/get-pending-nonce-and-sender");
const has_execution_succeeded_1 = require("../views/has-execution-succeeded");
const is_batch_finished_1 = require("../views/is-batch-finished");
const deployment_state_helpers_1 = require("./deployment-state-helpers");
const future_processor_1 = require("./future-processor/future-processor");
const get_max_nonce_used_by_sender_1 = require("./nonce-management/get-max-nonce-used-by-sender");
const get_nonce_sync_messages_1 = require("./nonce-management/get-nonce-sync-messages");
const json_rpc_nonce_manager_1 = require("./nonce-management/json-rpc-nonce-manager");
const transaction_tracking_timer_1 = require("./transaction-tracking-timer");
/**
 * This class is used to execute a module to completion, returning the new
 * deployment state.
 */
class ExecutionEngine {
    _deploymentLoader;
    _artifactResolver;
    _executionStrategy;
    _jsonRpcClient;
    _executionEventListener;
    _requiredConfirmations;
    _millisecondBeforeBumpingFees;
    _maxFeeBumps;
    _blockPollingInterval;
    constructor(_deploymentLoader, _artifactResolver, _executionStrategy, _jsonRpcClient, _executionEventListener, _requiredConfirmations, _millisecondBeforeBumpingFees, _maxFeeBumps, _blockPollingInterval) {
        this._deploymentLoader = _deploymentLoader;
        this._artifactResolver = _artifactResolver;
        this._executionStrategy = _executionStrategy;
        this._jsonRpcClient = _jsonRpcClient;
        this._executionEventListener = _executionEventListener;
        this._requiredConfirmations = _requiredConfirmations;
        this._millisecondBeforeBumpingFees = _millisecondBeforeBumpingFees;
        this._maxFeeBumps = _maxFeeBumps;
        this._blockPollingInterval = _blockPollingInterval;
    }
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
    async executeModule(deploymentState, module, batches, accounts, deploymentParameters, defaultSender) {
        deploymentState = await this._syncNonces(deploymentState, module, accounts, defaultSender);
        await this._executionStrategy.init(this._deploymentLoader, this._jsonRpcClient);
        const transactionTrackingTimer = new transaction_tracking_timer_1.TransactionTrackingTimer();
        const nonceManager = new json_rpc_nonce_manager_1.JsonRpcNonceManager(this._jsonRpcClient, (0, get_max_nonce_used_by_sender_1.getMaxNonceUsedBySender)(deploymentState));
        const futureProcessor = new future_processor_1.FutureProcessor(this._deploymentLoader, this._artifactResolver, this._executionStrategy, this._jsonRpcClient, transactionTrackingTimer, nonceManager, this._requiredConfirmations, this._millisecondBeforeBumpingFees, this._maxFeeBumps, accounts, deploymentParameters, defaultSender);
        const futures = (0, get_futures_from_module_1.getFuturesFromModule)(module);
        for (const batch of batches) {
            this._emitBeginNextBatchEvent();
            // TODO: consider changing batcher to return futures rather than ids
            const executionBatch = batch.map((futureId) => this._lookupFuture(futures, futureId));
            deploymentState = await this._executeBatch(futureProcessor, executionBatch, deploymentState);
            if (!executionBatch.every((f) => (0, has_execution_succeeded_1.hasExecutionSucceeded)(f, deploymentState))) {
                return deploymentState;
            }
        }
        return deploymentState;
    }
    /**
     * Executes a batch of futures until all of its futures are completed.
     *
     * @param futureProcessor The FutureProcessor to use for executing the futures.
     * @param batch The batch of futures to execute.
     * @param deploymentState The current deployment state.
     * @returns The new deployment state.
     */
    async _executeBatch(futureProcessor, batch, deploymentState) {
        // TODO: Do we really need to sort them here?
        const sortedFutures = this._getBatchSortedByHighesPendingNonce(batch, deploymentState);
        let block = await this._jsonRpcClient.getLatestBlock();
        while (true) {
            for (const future of sortedFutures) {
                const { newState } = await futureProcessor.processFuture(future, deploymentState);
                deploymentState = newState;
            }
            if ((0, is_batch_finished_1.isBatchFinished)(deploymentState, sortedFutures.map((f) => f.id))) {
                break;
            }
            block = await this._waitForNextBlock(block);
        }
        return deploymentState;
    }
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
    async _waitForNextBlock(previousBlock) {
        while (true) {
            await new Promise((resolve) => setTimeout(resolve, this._blockPollingInterval));
            const newBlock = await this._jsonRpcClient.getLatestBlock();
            if (newBlock.number > previousBlock.number) {
                return newBlock;
            }
        }
    }
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
    async _syncNonces(deploymentState, ignitionModule, accounts, defaultSender) {
        const nonceSyncMessages = await (0, get_nonce_sync_messages_1.getNonceSyncMessages)(this._jsonRpcClient, deploymentState, ignitionModule, accounts, defaultSender, this._requiredConfirmations);
        for (const message of nonceSyncMessages) {
            deploymentState = await (0, deployment_state_helpers_1.applyNewMessage)(message, deploymentState, this._deploymentLoader);
        }
        return deploymentState;
    }
    /**
     * Returns a future by its id.
     */
    _lookupFuture(futures, futureId) {
        const future = futures.find((f) => f.id === futureId);
        (0, assertions_1.assertIgnitionInvariant)(future !== undefined, `Future ${futureId} not found`);
        return future;
    }
    /**
     * Returns the batch sorted by the highest the pending nonce of each future
     * and sender.
     *
     * Futures without any pending nonce come last.
     */
    _getBatchSortedByHighesPendingNonce(batch, deploymentState) {
        const batchWithNonces = batch.map((f) => {
            const NO_PENDING_RESULT = {
                future: f,
                nonce: Number.MAX_SAFE_INTEGER,
                from: "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
            };
            const exState = deploymentState.executionStates[f.id];
            if (exState === undefined) {
                return NO_PENDING_RESULT;
            }
            const pendingNonceAndSender = (0, get_pending_nonce_and_sender_1.getPendingNonceAndSender)(exState);
            if (pendingNonceAndSender === undefined) {
                return NO_PENDING_RESULT;
            }
            return {
                future: f,
                nonce: pendingNonceAndSender.nonce,
                from: pendingNonceAndSender.sender,
            };
        });
        const sortBy = require("lodash/sortBy");
        const sortedBatch = sortBy(batchWithNonces, ["from", "nonce", "future.id"]);
        return sortedBatch.map((f) => f.future);
    }
    /**
     * Emits an execution event signaling that execution of the next batch has begun.
     */
    _emitBeginNextBatchEvent() {
        if (this._executionEventListener !== undefined) {
            this._executionEventListener.beginNextBatch({
                type: execution_events_1.ExecutionEventType.BEGIN_NEXT_BATCH,
            });
        }
    }
}
exports.ExecutionEngine = ExecutionEngine;
//# sourceMappingURL=execution-engine.js.map