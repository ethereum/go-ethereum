"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.sendTransaction = void 0;
const assertions_1 = require("../../../utils/assertions");
const execution_result_1 = require("../../types/execution-result");
const messages_1 = require("../../types/messages");
const network_interaction_1 = require("../../types/network-interaction");
const decode_simulation_result_1 = require("../helpers/decode-simulation-result");
const messages_helpers_1 = require("../helpers/messages-helpers");
const network_interaction_execution_1 = require("../helpers/network-interaction-execution");
const replay_strategy_1 = require("../helpers/replay-strategy");
/**
 * Sends a transaction for the execution state's latest NetworkInteraction
 * and returns a TransactionSendMessage, or an execution state complete message
 * in case of an error.
 *
 * This function can send the first transaction of an OnchainInteraction, as well
 * as new transactions to bump fees and recovering from dropped transactions.
 *
 * SIDE EFFECTS: This function has side effects, as it sends a transaction. These
 *  include: sending the transaction to the network, allocating a nonce in the
 *  NonceManager if needed, and adding the transaction to the TransactionTrackingTimer.
 *
 * @param exState The execution state that requires a transaction to be sent.
 * @param executionStrategy The execution strategy to use for simulations.
 * @param jsonRpcClient The JSON RPC client to use for the transaction.
 * @param nonceManager The NonceManager to allocate nonces if needed.
 * @param transactionTrackingTimer The TransactionTrackingTimer to add the transaction to.
 * @returns A message indicating the result of trying to send the transaction.
 */
async function sendTransaction(exState, executionStrategy, jsonRpcClient, nonceManager, transactionTrackingTimer) {
    const lastNetworkInteraction = exState.networkInteractions.at(-1);
    (0, assertions_1.assertIgnitionInvariant)(lastNetworkInteraction !== undefined, `No network interaction found for ExecutionState ${exState.id} when trying to send a transaction`);
    (0, assertions_1.assertIgnitionInvariant)(lastNetworkInteraction.type === network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION, `StaticCall found as last network interaction of ExecutionState ${exState.id} when trying to send a transaction`);
    const generator = await (0, replay_strategy_1.replayStrategy)(exState, executionStrategy);
    // This cast is safe because the execution state is of static call type.
    const strategyGenerator = generator;
    const result = await (0, network_interaction_execution_1.sendTransactionForOnchainInteraction)(jsonRpcClient, exState.from, lastNetworkInteraction, nonceManager, (0, decode_simulation_result_1.decodeSimulationResult)(strategyGenerator, exState));
    // If the transaction failed during simulation, we need to revert the nonce allocation
    if (result.type === execution_result_1.ExecutionResultType.STRATEGY_SIMULATION_ERROR ||
        result.type === execution_result_1.ExecutionResultType.SIMULATION_ERROR) {
        nonceManager.revertNonce(exState.from);
    }
    if (result.type === network_interaction_execution_1.TRANSACTION_SENT_TYPE) {
        transactionTrackingTimer.addTransaction(result.transaction.hash);
        return {
            type: messages_1.JournalMessageType.TRANSACTION_SEND,
            futureId: exState.id,
            networkInteractionId: lastNetworkInteraction.id,
            transaction: result.transaction,
            nonce: result.nonce,
        };
    }
    return (0, messages_helpers_1.createExecutionStateCompleteMessageForExecutionsWithOnchainInteractions)(exState, result);
}
exports.sendTransaction = sendTransaction;
//# sourceMappingURL=send-transaction.js.map