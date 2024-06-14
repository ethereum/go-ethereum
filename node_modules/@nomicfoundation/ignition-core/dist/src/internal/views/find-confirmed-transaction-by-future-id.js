"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.findConfirmedTransactionByFutureId = void 0;
const execution_state_1 = require("../execution/types/execution-state");
const network_interaction_1 = require("../execution/types/network-interaction");
const assertions_1 = require("../utils/assertions");
function findConfirmedTransactionByFutureId(deploymentState, futureId) {
    const exState = deploymentState.executionStates[futureId];
    (0, assertions_1.assertIgnitionInvariant)(exState !== undefined, `Cannot resolve tx hash, no execution state for ${futureId}`);
    (0, assertions_1.assertIgnitionInvariant)(exState.type === execution_state_1.ExecutionSateType.DEPLOYMENT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionSateType.SEND_DATA_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionSateType.CALL_EXECUTION_STATE, `Tx hash resolution only supported on execution states with network interactions, ${futureId} is ${exState.type}`);
    const lastNetworkInteraction = exState.networkInteractions.at(-1);
    (0, assertions_1.assertIgnitionInvariant)(lastNetworkInteraction !== undefined, `Tx hash resolution unable to find a network interaction for ${futureId}`);
    (0, assertions_1.assertIgnitionInvariant)(lastNetworkInteraction.type === network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION, "Tx hash resolution only supported onchain interaction");
    // On confirmation only one transaction is preserverd
    const transaction = lastNetworkInteraction.transactions[0];
    (0, assertions_1.assertIgnitionInvariant)(transaction !== undefined && transaction.receipt !== undefined, `Tx hash resolution unable to find confirmed transaction for ${futureId}`);
    return { ...transaction, receipt: transaction.receipt };
}
exports.findConfirmedTransactionByFutureId = findConfirmedTransactionByFutureId;
//# sourceMappingURL=find-confirmed-transaction-by-future-id.js.map