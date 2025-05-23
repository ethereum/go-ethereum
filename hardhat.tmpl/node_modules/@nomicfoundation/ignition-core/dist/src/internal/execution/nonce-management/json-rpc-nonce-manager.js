"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.JsonRpcNonceManager = void 0;
const errors_1 = require("../../../errors");
const errors_list_1 = require("../../errors-list");
/**
 * An implementation of NonceManager that validates the nonces using
 * the _maxUsedNonce params and a JsonRpcClient.
 */
class JsonRpcNonceManager {
    _jsonRpcClient;
    _maxUsedNonce;
    constructor(_jsonRpcClient, _maxUsedNonce) {
        this._jsonRpcClient = _jsonRpcClient;
        this._maxUsedNonce = _maxUsedNonce;
    }
    async getNextNonce(sender) {
        const pendingCount = await this._jsonRpcClient.getTransactionCount(sender, "pending");
        const expectedNonce = this._maxUsedNonce[sender] !== undefined
            ? this._maxUsedNonce[sender] + 1
            : pendingCount;
        if (expectedNonce !== pendingCount) {
            throw new errors_1.IgnitionError(errors_list_1.ERRORS.EXECUTION.INVALID_NONCE, {
                sender,
                expectedNonce,
                pendingCount,
            });
        }
        // The nonce hasn't been used yet, but we update as
        // it will be immediately used.
        this._maxUsedNonce[sender] = expectedNonce;
        return expectedNonce;
    }
    revertNonce(sender) {
        this._maxUsedNonce[sender] -= 1;
    }
}
exports.JsonRpcNonceManager = JsonRpcNonceManager;
//# sourceMappingURL=json-rpc-nonce-manager.js.map