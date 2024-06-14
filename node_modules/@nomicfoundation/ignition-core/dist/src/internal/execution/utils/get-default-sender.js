"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getDefaultSender = void 0;
/**
 * Returns the default sender to be used as `from` in futures, transactions
 * and static calls.
 *
 * @param accounts The accounts provided by the integrator of ignition.
 * @returns The default sender.
 */
function getDefaultSender(accounts) {
    return accounts[0];
}
exports.getDefaultSender = getDefaultSender;
//# sourceMappingURL=get-default-sender.js.map