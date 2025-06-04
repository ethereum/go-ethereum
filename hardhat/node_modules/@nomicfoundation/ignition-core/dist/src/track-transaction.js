"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.trackTransaction = void 0;
const fs_extra_1 = require("fs-extra");
const errors_1 = require("./errors");
const file_deployment_loader_1 = require("./internal/deployment-loader/file-deployment-loader");
const errors_list_1 = require("./internal/errors-list");
const deployment_state_helpers_1 = require("./internal/execution/deployment-state-helpers");
const jsonrpc_client_1 = require("./internal/execution/jsonrpc-client");
const messages_1 = require("./internal/execution/types/messages");
const defaultConfig_1 = require("./internal/defaultConfig");
const assertions_1 = require("./internal/utils/assertions");
const network_interaction_1 = require("./internal/execution/types/network-interaction");
const get_network_execution_states_1 = require("./internal/views/execution-state/get-network-execution-states");
/**
 * Tracks a transaction associated with a given deployment.
 *
 * @param deploymentDir - the directory of the deployment the transaction belongs to
 * @param txHash - the hash of the transaction to track
 * @param provider - a JSON RPC provider to retrieve transaction information from
 * @param requiredConfirmations - the number of confirmations required for the transaction to be considered confirmed
 * @param applyNewMessageFn - only used for ease of testing this function and should not be used otherwise
 *
 * @beta
 */
async function trackTransaction(deploymentDir, txHash, provider, requiredConfirmations = defaultConfig_1.defaultConfig.requiredConfirmations, applyNewMessageFn = deployment_state_helpers_1.applyNewMessage) {
    if (!(await (0, fs_extra_1.pathExists)(deploymentDir))) {
        throw new errors_1.IgnitionError(errors_list_1.ERRORS.TRACK_TRANSACTION.DEPLOYMENT_DIR_NOT_FOUND, {
            deploymentDir,
        });
    }
    const deploymentLoader = new file_deployment_loader_1.FileDeploymentLoader(deploymentDir);
    const deploymentState = await (0, deployment_state_helpers_1.loadDeploymentState)(deploymentLoader);
    if (deploymentState === undefined) {
        throw new errors_1.IgnitionError(errors_list_1.ERRORS.TRACK_TRANSACTION.UNINITIALIZED_DEPLOYMENT, {
            deploymentDir,
        });
    }
    const jsonRpcClient = new jsonrpc_client_1.EIP1193JsonRpcClient(provider);
    const transaction = await jsonRpcClient.getFullTransaction(txHash);
    if (transaction === undefined) {
        throw new errors_1.IgnitionError(errors_list_1.ERRORS.TRACK_TRANSACTION.TRANSACTION_NOT_FOUND, {
            txHash,
        });
    }
    const exStates = (0, get_network_execution_states_1.getNetworkExecutionStates)(deploymentState);
    /**
     * Cases to consider:
     * 1. (happy case) given txhash matches a nonce we prepared but didn't record sending
     * 2. (user replaced with different tx) given txhash matches a nonce we prepared but didn't record sending,
     *     but the tx details are different
     * 3. (user sent known txhash) given txhash matches a nonce we recorded sending with the same txhash
     * 4. (user sent unknown txhash) given txhash matches a nonce we recorded sending but with a different txhash
     * 5. (user sent unrelated txhash) given txhash doesn't match any nonce we've allocated
     */
    for (const exState of exStates) {
        for (const networkInteraction of exState.networkInteractions) {
            if (networkInteraction.type ===
                network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION &&
                exState.from.toLowerCase() === transaction.from.toLowerCase() &&
                networkInteraction.nonce === transaction.nonce) {
                if (networkInteraction.transactions.length === 0) {
                    // case 1: the txHash matches a transaction we appear to have sent
                    if (networkInteraction.to?.toLowerCase() ===
                        transaction.to?.toLowerCase() &&
                        networkInteraction.data === transaction.data &&
                        networkInteraction.value === transaction.value) {
                        let fees;
                        if ("maxFeePerGas" in transaction &&
                            "maxPriorityFeePerGas" in transaction &&
                            transaction.maxFeePerGas !== undefined &&
                            transaction.maxPriorityFeePerGas !== undefined) {
                            fees = {
                                maxFeePerGas: transaction.maxFeePerGas,
                                maxPriorityFeePerGas: transaction.maxPriorityFeePerGas,
                            };
                        }
                        else {
                            (0, assertions_1.assertIgnitionInvariant)("gasPrice" in transaction && transaction.gasPrice !== undefined, "Transaction fees are missing");
                            fees = {
                                gasPrice: transaction.gasPrice,
                            };
                        }
                        const transactionSendMessage = {
                            futureId: exState.id,
                            networkInteractionId: networkInteraction.id,
                            nonce: networkInteraction.nonce,
                            type: messages_1.JournalMessageType.TRANSACTION_SEND,
                            transaction: {
                                hash: transaction.hash,
                                fees,
                            },
                        };
                        await applyNewMessageFn(transactionSendMessage, deploymentState, deploymentLoader);
                        return;
                    }
                    // case 2: the user sent a different transaction that replaced ours
                    // so we check their transaction for the required number of confirmations
                    else {
                        return checkConfirmations(exState, networkInteraction, transaction, requiredConfirmations, jsonRpcClient, deploymentState, deploymentLoader, applyNewMessageFn);
                    }
                }
                // case: the user gave us a transaction that matches a nonce we've already recorded sending from
                else {
                    // case 3: the txHash matches the one we have saved in the journal for the same nonce
                    if (networkInteraction.transactions[0].hash === transaction.hash) {
                        throw new errors_1.IgnitionError(errors_list_1.ERRORS.TRACK_TRANSACTION.KNOWN_TRANSACTION);
                    }
                    // case 4: the user sent a different transaction that replaced ours
                    // so we check their transaction for the required number of confirmations
                    return checkConfirmations(exState, networkInteraction, transaction, requiredConfirmations, jsonRpcClient, deploymentState, deploymentLoader, applyNewMessageFn);
                }
            }
        }
    }
    // case 5: the txHash doesn't match any nonce we've allocated
    throw new errors_1.IgnitionError(errors_list_1.ERRORS.TRACK_TRANSACTION.MATCHING_NONCE_NOT_FOUND);
}
exports.trackTransaction = trackTransaction;
async function checkConfirmations(exState, networkInteraction, transaction, requiredConfirmations, jsonRpcClient, deploymentState, deploymentLoader, applyNewMessageFn) {
    const [block, receipt] = await Promise.all([
        jsonRpcClient.getLatestBlock(),
        jsonRpcClient.getTransactionReceipt(transaction.hash),
    ]);
    (0, assertions_1.assertIgnitionInvariant)(receipt !== undefined, "Unable to retrieve transaction receipt");
    const confirmations = block.number - receipt.blockNumber + 1;
    if (confirmations >= requiredConfirmations) {
        const transactionReplacedMessage = {
            futureId: exState.id,
            networkInteractionId: networkInteraction.id,
            type: messages_1.JournalMessageType.ONCHAIN_INTERACTION_REPLACED_BY_USER,
        };
        await applyNewMessageFn(transactionReplacedMessage, deploymentState, deploymentLoader);
        /**
         * We tell the user specifically what future will be executed upon re-running the deployment
         * in case the replacement transaction sent by the user was the same transaction that we were going to send.
         *
         * i.e., if the broken transaction was for a future sending 100 ETH to an address, and the user decided to just send it
         * themselves after the deployment failed, we tell them that the future sending 100 ETH will be executed upon re-running
         * the deployment. It is not obvious to the user that that is the case, and it could result in a double send if they assume
         * the opposite.
         */
        return `Your deployment has been fixed and will continue with the execution of the "${exState.id}" future.

If this is not the expected behavior, please edit your Hardhat Ignition module accordingly before re-running your deployment.`;
    }
    else {
        throw new errors_1.IgnitionError(errors_list_1.ERRORS.TRACK_TRANSACTION.INSUFFICIENT_CONFIRMATIONS);
    }
}
//# sourceMappingURL=track-transaction.js.map