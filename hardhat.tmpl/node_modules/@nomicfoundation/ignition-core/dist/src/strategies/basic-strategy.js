"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.BasicStrategy = void 0;
const execution_strategy_helpers_1 = require("../internal/execution/execution-strategy-helpers");
const execution_result_1 = require("../internal/execution/types/execution-result");
const execution_strategy_1 = require("../internal/execution/types/execution-strategy");
const network_interaction_1 = require("../internal/execution/types/network-interaction");
const assertions_1 = require("../internal/utils/assertions");
/**
 * The basic execution strategy, which sends a single transaction
 * for each contract deployment, call, and send data, and a single static call
 * for each static call execution.
 *
 * @private
 */
class BasicStrategy {
    name = "basic";
    config;
    _deploymentLoader;
    constructor() {
        this.config = {};
    }
    async init(deploymentLoader, _jsonRpcClient) {
        this._deploymentLoader = deploymentLoader;
    }
    async *executeDeployment(executionState) {
        (0, assertions_1.assertIgnitionInvariant)(this._deploymentLoader !== undefined, `Strategy ${this.name} not initialized`);
        const artifact = await this._deploymentLoader.loadArtifact(executionState.artifactId);
        const transactionOrResult = yield* (0, execution_strategy_helpers_1.executeOnchainInteractionRequest)(executionState.id, {
            id: 1,
            type: network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION,
            to: undefined,
            data: (0, execution_strategy_helpers_1.encodeArtifactDeploymentData)(artifact, executionState.constructorArgs, executionState.libraries),
            value: executionState.value,
        }, undefined, (returnData) => (0, execution_strategy_helpers_1.decodeArtifactCustomError)(artifact, returnData));
        if (transactionOrResult.type !==
            execution_strategy_1.OnchainInteractionResponseType.SUCCESSFUL_TRANSACTION) {
            return transactionOrResult;
        }
        const tx = transactionOrResult.transaction;
        const contractAddress = tx.receipt.contractAddress;
        if (contractAddress === undefined) {
            return {
                type: execution_result_1.ExecutionResultType.STRATEGY_ERROR,
                error: `Transaction ${tx.hash} confirmed but it didn't create a contract`,
            };
        }
        return {
            type: execution_result_1.ExecutionResultType.SUCCESS,
            address: contractAddress,
        };
    }
    async *executeCall(executionState) {
        (0, assertions_1.assertIgnitionInvariant)(this._deploymentLoader !== undefined, `Strategy ${this.name} not initialized`);
        const artifact = await this._deploymentLoader.loadArtifact(executionState.artifactId);
        const transactionOrResult = yield* (0, execution_strategy_helpers_1.executeOnchainInteractionRequest)(executionState.id, {
            id: 1,
            type: network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION,
            to: executionState.contractAddress,
            data: (0, execution_strategy_helpers_1.encodeArtifactFunctionCall)(artifact, executionState.functionName, executionState.args),
            value: executionState.value,
        }, (returnData) => (0, execution_strategy_helpers_1.decodeArtifactFunctionCallResult)(artifact, executionState.functionName, returnData), (returnData) => (0, execution_strategy_helpers_1.decodeArtifactCustomError)(artifact, returnData));
        if (transactionOrResult.type !==
            execution_strategy_1.OnchainInteractionResponseType.SUCCESSFUL_TRANSACTION) {
            return transactionOrResult;
        }
        return {
            type: execution_result_1.ExecutionResultType.SUCCESS,
        };
    }
    async *executeSendData(executionState) {
        const transactionOrResult = yield* (0, execution_strategy_helpers_1.executeOnchainInteractionRequest)(executionState.id, {
            id: 1,
            type: network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION,
            to: executionState.to,
            data: executionState.data,
            value: executionState.value,
        });
        if (transactionOrResult.type !==
            execution_strategy_1.OnchainInteractionResponseType.SUCCESSFUL_TRANSACTION) {
            return transactionOrResult;
        }
        return {
            type: execution_result_1.ExecutionResultType.SUCCESS,
        };
    }
    async *executeStaticCall(executionState) {
        (0, assertions_1.assertIgnitionInvariant)(this._deploymentLoader !== undefined, `Strategy ${this.name} not initialized`);
        const artifact = await this._deploymentLoader.loadArtifact(executionState.artifactId);
        const decodedResultOrError = yield* (0, execution_strategy_helpers_1.executeStaticCallRequest)({
            id: 1,
            type: network_interaction_1.NetworkInteractionType.STATIC_CALL,
            to: executionState.contractAddress,
            from: executionState.from,
            data: (0, execution_strategy_helpers_1.encodeArtifactFunctionCall)(artifact, executionState.functionName, executionState.args),
            value: 0n,
        }, (returnData) => (0, execution_strategy_helpers_1.decodeArtifactFunctionCallResult)(artifact, executionState.functionName, returnData), (returnData) => (0, execution_strategy_helpers_1.decodeArtifactCustomError)(artifact, returnData));
        if (decodedResultOrError.type === execution_result_1.ExecutionResultType.STATIC_CALL_ERROR) {
            return decodedResultOrError;
        }
        return {
            type: execution_result_1.ExecutionResultType.SUCCESS,
            value: (0, execution_strategy_helpers_1.getStaticCallExecutionStateResultValue)(executionState, decodedResultOrError),
        };
    }
}
exports.BasicStrategy = BasicStrategy;
//# sourceMappingURL=basic-strategy.js.map