"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.findStaticCallBy = void 0;
const network_interaction_1 = require("../../execution/types/network-interaction");
const assertions_1 = require("../../utils/assertions");
function findStaticCallBy(executionState, networkInteractionId) {
    const staticCall = executionState.networkInteractions.find((interaction) => interaction.id === networkInteractionId);
    (0, assertions_1.assertIgnitionInvariant)(staticCall !== undefined, `Expected static call ${executionState.id}/${networkInteractionId} to exist, but it did not`);
    (0, assertions_1.assertIgnitionInvariant)(staticCall.type === network_interaction_1.NetworkInteractionType.STATIC_CALL, `Expected network interaction ${executionState.id}/${networkInteractionId} to be a static call, but instead it was ${staticCall.type}`);
    return staticCall;
}
exports.findStaticCallBy = findStaticCallBy;
//# sourceMappingURL=find-static-call-by.js.map