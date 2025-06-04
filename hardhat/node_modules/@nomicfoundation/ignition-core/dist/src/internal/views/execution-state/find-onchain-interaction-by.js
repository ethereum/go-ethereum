"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.findOnchainInteractionBy = void 0;
const network_interaction_1 = require("../../execution/types/network-interaction");
const assertions_1 = require("../../utils/assertions");
function findOnchainInteractionBy(executionState, networkInteractionId) {
    const onchainInteraction = executionState.networkInteractions.find((interaction) => interaction.id === networkInteractionId);
    (0, assertions_1.assertIgnitionInvariant)(onchainInteraction !== undefined, `Expected network interaction ${executionState.id}/${networkInteractionId} to exist, but it did not`);
    (0, assertions_1.assertIgnitionInvariant)(onchainInteraction.type === network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION, `Expected network interaction ${executionState.id}/${networkInteractionId} to be an onchain interaction, but instead it was ${onchainInteraction.type}`);
    return onchainInteraction;
}
exports.findOnchainInteractionBy = findOnchainInteractionBy;
//# sourceMappingURL=find-onchain-interaction-by.js.map