"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.queryStaticCall = void 0;
const assertions_1 = require("../../../utils/assertions");
const messages_1 = require("../../types/messages");
const network_interaction_1 = require("../../types/network-interaction");
const network_interaction_execution_1 = require("../helpers/network-interaction-execution");
/**
 * Runs a static call and returns a message indicating its completion.
 *
 * SIDE EFFECTS: This function doesn't have any side effects.
 *
 * @param exState The execution state that requested the static call.
 * @param jsonRpcClient The JSON RPC client to use for the static call.
 * @returns A message indicating the completion of the static call.
 */
async function queryStaticCall(exState, jsonRpcClient) {
    const lastNetworkInteraction = exState.networkInteractions.at(-1);
    (0, assertions_1.assertIgnitionInvariant)(lastNetworkInteraction !== undefined, `Network interaction not found for ExecutionState ${exState.id} when trying to run a StaticCall`);
    (0, assertions_1.assertIgnitionInvariant)(lastNetworkInteraction.type === network_interaction_1.NetworkInteractionType.STATIC_CALL, `Transaction found as last network interaction of ExecutionState ${exState.id} when trying to run a StaticCall`);
    (0, assertions_1.assertIgnitionInvariant)(lastNetworkInteraction.result === undefined, `Resolved StaticCall found in ${exState.id}/${lastNetworkInteraction.id} when trying to run it`);
    const result = await (0, network_interaction_execution_1.runStaticCall)(jsonRpcClient, lastNetworkInteraction);
    return {
        type: messages_1.JournalMessageType.STATIC_CALL_COMPLETE,
        futureId: exState.id,
        networkInteractionId: lastNetworkInteraction.id,
        result,
    };
}
exports.queryStaticCall = queryStaticCall;
//# sourceMappingURL=query-static-call.js.map