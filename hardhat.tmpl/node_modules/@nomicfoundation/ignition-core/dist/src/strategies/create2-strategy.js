"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Create2Strategy = void 0;
const ethers_1 = require("ethers");
const errors_1 = require("../errors");
const execution_strategy_helpers_1 = require("../internal/execution/execution-strategy-helpers");
const createx_artifact_1 = require("../internal/execution/strategy/createx-artifact");
const execution_result_1 = require("../internal/execution/types/execution-result");
const execution_strategy_1 = require("../internal/execution/types/execution-strategy");
const network_interaction_1 = require("../internal/execution/types/network-interaction");
const assertions_1 = require("../internal/utils/assertions");
// v0.1.0
const CREATE_X_ADDRESS = "0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed";
const CREATE_X_DEPLOYED_BYTECODE_HASH = "0xbd8a7ea8cfca7b4e5f5041d7d4b17bc317c5ce42cfbc42066a00cf26b43eb53f";
const CREATE_X_PRESIGNED_DEPLOYER_ADDRESS = "0xeD456e05CaAb11d66C4c797dD6c1D6f9A7F352b5";
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
class Create2Strategy {
    name = "create2";
    config;
    _deploymentLoader;
    _jsonRpcClient;
    constructor(config) {
        this.config = config;
    }
    async init(deploymentLoader, jsonRpcClient) {
        this._deploymentLoader = deploymentLoader;
        this._jsonRpcClient = jsonRpcClient;
        // Check if CreateX is deployed on the current network
        const result = await this._jsonRpcClient.getCode(CREATE_X_ADDRESS);
        // If CreateX factory is deployed (and bytecode matches) then nothing to do
        if (result !== "0x") {
            (0, assertions_1.assertIgnitionInvariant)(ethers_1.ethers.keccak256(result) === CREATE_X_DEPLOYED_BYTECODE_HASH, "Deployed CreateX bytecode does not match expected bytecode");
            return;
        }
        const chainId = await this._jsonRpcClient.getChainId();
        // Otherwise if we're not on a local hardhat node, throw an error
        if (chainId !== 31337) {
            throw new errors_1.NomicIgnitionPluginError("create2", `CreateX not deployed on current network ${chainId}`);
        }
        // On a local hardhat node, deploy the CreateX factory
        await this._deployCreateXFactory(this._jsonRpcClient);
    }
    async *executeDeployment(executionState) {
        (0, assertions_1.assertIgnitionInvariant)(this._deploymentLoader !== undefined && this._jsonRpcClient !== undefined, `Strategy ${this.name} not initialized`);
        const artifact = await this._deploymentLoader.loadArtifact(executionState.artifactId);
        const bytecodeToDeploy = (0, execution_strategy_helpers_1.encodeArtifactDeploymentData)(artifact, executionState.constructorArgs, executionState.libraries);
        const transactionOrResult = yield* (0, execution_strategy_helpers_1.executeOnchainInteractionRequest)(executionState.id, {
            id: 1,
            type: network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION,
            to: CREATE_X_ADDRESS,
            data: (0, execution_strategy_helpers_1.encodeArtifactFunctionCall)(createx_artifact_1.createxArtifact, "deployCreate2(bytes32,bytes)", [this.config.salt, bytecodeToDeploy]),
            value: executionState.value,
        }, (returnData) => (0, execution_strategy_helpers_1.decodeArtifactFunctionCallResult)(createx_artifact_1.createxArtifact, "deployCreate2(bytes32,bytes)", returnData), (returnData) => (0, execution_strategy_helpers_1.decodeArtifactCustomError)(createx_artifact_1.createxArtifact, returnData));
        if (transactionOrResult.type !==
            execution_strategy_1.OnchainInteractionResponseType.SUCCESSFUL_TRANSACTION) {
            return transactionOrResult;
        }
        const deployedAddress = (0, execution_strategy_helpers_1.getEventArgumentFromReceipt)(transactionOrResult.transaction.receipt, createx_artifact_1.createxArtifact, CREATE_X_ADDRESS, "ContractCreation", 0, "newContract");
        (0, assertions_1.assertIgnitionInvariant)(typeof deployedAddress === "string", "Deployed event should return a string addr property");
        return {
            type: execution_result_1.ExecutionResultType.SUCCESS,
            address: deployedAddress,
        };
    }
    async *executeCall(executionState) {
        (0, assertions_1.assertIgnitionInvariant)(this._deploymentLoader !== undefined && this._jsonRpcClient !== undefined, `Strategy ${this.name} not initialized`);
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
        (0, assertions_1.assertIgnitionInvariant)(this._deploymentLoader !== undefined && this._jsonRpcClient !== undefined, `Strategy ${this.name} not initialized`);
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
    /**
     * Within the context of a local development Hardhat chain, deploy
     * the CreateX factory contract using a presigned transaction.
     */
    async _deployCreateXFactory(client) {
        // The account that will deploy the CreateX factory needs to be funded
        // first
        await client.setBalance(CREATE_X_PRESIGNED_DEPLOYER_ADDRESS, 400000000000000000n);
        const txHash = await client.sendRawTransaction(createx_artifact_1.presignedTx);
        (0, assertions_1.assertIgnitionInvariant)(txHash !== "0x", "CreateX deployment failed");
        while (true) {
            const receipt = await client.getTransactionReceipt(txHash);
            if (receipt !== undefined) {
                (0, assertions_1.assertIgnitionInvariant)(receipt?.contractAddress !== undefined, "CreateX deployment should have an address");
                (0, assertions_1.assertIgnitionInvariant)(receipt.contractAddress === CREATE_X_ADDRESS, `CreateX deployment should have the expected address ${CREATE_X_ADDRESS}, instead it is ${receipt.contractAddress}`);
                return;
            }
            await new Promise((res) => setTimeout(res, 200));
        }
    }
}
exports.Create2Strategy = Create2Strategy;
//# sourceMappingURL=create2-strategy.js.map