"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.runStrategy = void 0;
const assertions_1 = require("../../../utils/assertions");
const execution_result_1 = require("../../types/execution-result");
const execution_state_1 = require("../../types/execution-state");
const execution_strategy_1 = require("../../types/execution-strategy");
const jsonrpc_1 = require("../../types/jsonrpc");
const messages_1 = require("../../types/messages");
const network_interaction_1 = require("../../types/network-interaction");
const messages_helpers_1 = require("../helpers/messages-helpers");
const replay_strategy_1 = require("../helpers/replay-strategy");
/**
 * Runs the strategy for the execution state, and returns a message that can be
 * a network interaction request, or an execution state complete message.
 *
 * Execution state complete messages can be a result of running the strategy,
 * or of the transaction executing the latest network interaction having reverted.
 *
 * SIDE EFFECTS: This function doesn't have any side effects.
 *
 * @param exState The execution state that requires the strategy to be run.
 * @param executionStrategy The execution strategy to use.
 * @returns A message indicating the result of running the strategy or a reverted tx.
 */
async function runStrategy(exState, executionStrategy) {
    const strategyGenerator = await (0, replay_strategy_1.replayStrategy)(exState, executionStrategy);
    const lastNetworkInteraction = exState.networkInteractions.at(-1);
    let response;
    if (lastNetworkInteraction === undefined) {
        response = await strategyGenerator.next();
    }
    else if (lastNetworkInteraction.type === network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION) {
        (0, assertions_1.assertIgnitionInvariant)(exState.type !== execution_state_1.ExecutionSateType.STATIC_CALL_EXECUTION_STATE, `Unexpected StaticCallExecutionState ${exState.id} with onchain interaction ${lastNetworkInteraction.id} when trying to run a strategy`);
        // We know this is safe because StaticCallExecutionState's can't generate
        // OnchainInteractions.
        const typedGenerator = strategyGenerator;
        const confirmedTx = lastNetworkInteraction.transactions.find((tx) => tx.receipt !== undefined);
        (0, assertions_1.assertIgnitionInvariant)(confirmedTx !== undefined && confirmedTx.receipt !== undefined, "Trying to advance strategy without confirmed tx in the last network interaction");
        if (confirmedTx.receipt.status === jsonrpc_1.TransactionReceiptStatus.FAILURE) {
            const result = {
                type: execution_result_1.ExecutionResultType.REVERTED_TRANSACTION,
                txHash: confirmedTx.hash,
            };
            return (0, messages_helpers_1.createExecutionStateCompleteMessageForExecutionsWithOnchainInteractions)(exState, result);
        }
        response = await typedGenerator.next({
            type: execution_strategy_1.OnchainInteractionResponseType.SUCCESSFUL_TRANSACTION,
            transaction: {
                ...confirmedTx,
                receipt: {
                    ...confirmedTx.receipt,
                    status: jsonrpc_1.TransactionReceiptStatus.SUCCESS,
                },
            },
        });
    }
    else {
        (0, assertions_1.assertIgnitionInvariant)(lastNetworkInteraction.result !== undefined, "Trying to advance strategy without result in the last network interaction");
        response = await strategyGenerator.next(lastNetworkInteraction.result);
    }
    if (response.done !== true) {
        (0, assertions_1.assertIgnitionInvariant)(response.value.type !== execution_strategy_1.SIMULATION_SUCCESS_SIGNAL_TYPE, "Invalid SIMULATION_SUCCESS_SIGNAL received");
        return {
            type: messages_1.JournalMessageType.NETWORK_INTERACTION_REQUEST,
            futureId: exState.id,
            networkInteraction: resolveNetworkInteractionRequest(exState, response.value),
        };
    }
    return (0, messages_helpers_1.createExecutionStateCompleteMessage)(exState, response.value);
}
exports.runStrategy = runStrategy;
function resolveNetworkInteractionRequest(exState, req) {
    if (req.type === network_interaction_1.NetworkInteractionType.STATIC_CALL) {
        return {
            ...req,
            from: req.from ?? exState.from,
        };
    }
    return req;
}
//# sourceMappingURL=run-strategy.js.map