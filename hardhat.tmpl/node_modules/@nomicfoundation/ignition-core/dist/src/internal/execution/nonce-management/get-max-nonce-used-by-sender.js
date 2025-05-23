"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getMaxNonceUsedBySender = void 0;
const get_pending_nonce_and_sender_1 = require("../../views/execution-state/get-pending-nonce-and-sender");
function getMaxNonceUsedBySender(deploymentState) {
    const nonces = {};
    for (const executionState of Object.values(deploymentState.executionStates)) {
        const pendingNonceAndSender = (0, get_pending_nonce_and_sender_1.getPendingNonceAndSender)(executionState);
        if (pendingNonceAndSender === undefined) {
            continue;
        }
        const { sender, nonce } = pendingNonceAndSender;
        if (nonces[sender] === undefined) {
            nonces[sender] = nonce;
        }
        else {
            nonces[sender] = Math.max(nonces[sender], nonce);
        }
    }
    return nonces;
}
exports.getMaxNonceUsedBySender = getMaxNonceUsedBySender;
//# sourceMappingURL=get-max-nonce-used-by-sender.js.map