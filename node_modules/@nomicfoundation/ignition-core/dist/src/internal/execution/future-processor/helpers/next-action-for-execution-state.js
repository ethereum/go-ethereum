"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.nextActionForExecutionState = exports.NextAction = void 0;
const network_interaction_1 = require("../../types/network-interaction");
/**
 * The next action that the FutureProcessor should take.
 */
var NextAction;
(function (NextAction) {
    /**
     * This action is used when the latest network interaction was completed
     * and the execution strategy should be run again, to understand how to
     * proceed.
     */
    NextAction["RUN_STRATEGY"] = "RUN_STRATEGY";
    /**
     * This action is used when the latest network interaction is an OnchainInteraction
     * that requires sending a new transaction.
     */
    NextAction["SEND_TRANSACTION"] = "SEND_TRANSACTION";
    /**
     * This action is used when the latest network interaction is a StaticCall that
     * hasn't been run yet.
     */
    NextAction["QUERY_STATIC_CALL"] = "QUERY_STATIC_CALL";
    /**
     * This action is used when the latest network interaction is an OnchainInteraction
     * that has one or more in-flight transactions, and we need to monitor them.
     */
    NextAction["MONITOR_ONCHAIN_INTERACTION"] = "MONITOR_ONCHAIN_INTERACTION";
})(NextAction || (exports.NextAction = NextAction = {}));
/**
 * Returns the next action to be run for an execution state.
 */
function nextActionForExecutionState(exState) {
    const interaction = exState.networkInteractions.at(-1);
    if (interaction === undefined) {
        return NextAction.RUN_STRATEGY;
    }
    switch (interaction.type) {
        case network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION: {
            if (interaction.transactions.length === 0) {
                return NextAction.SEND_TRANSACTION;
            }
            else {
                const receipt = interaction.transactions.find((tx) => tx.receipt);
                if (receipt !== undefined) {
                    // We got a confirmed transaction
                    return NextAction.RUN_STRATEGY;
                }
                if (interaction.shouldBeResent) {
                    return NextAction.SEND_TRANSACTION;
                }
                // Wait for confirmations, drops, or nonce invalidation
                return NextAction.MONITOR_ONCHAIN_INTERACTION;
            }
        }
        case network_interaction_1.NetworkInteractionType.STATIC_CALL: {
            if (interaction.result !== undefined) {
                return NextAction.RUN_STRATEGY;
            }
            return NextAction.QUERY_STATIC_CALL;
        }
    }
}
exports.nextActionForExecutionState = nextActionForExecutionState;
//# sourceMappingURL=next-action-for-execution-state.js.map