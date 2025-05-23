"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.TransactionTrackingTimer = void 0;
/**
 * This class is used to track the time that we have been waiting for
 * a transaction to confirm since it was either sent, or since Ignition started
 * and it was already sent.
 *
 * Note: This class doesn't have a method to clear the timer for a transaction
 * but it shouldn't be problematic.
 */
class TransactionTrackingTimer {
    _defaultStart = Date.now();
    _transactionTrackingTimes = {};
    /**
     * Adds a new transaction to track.
     */
    addTransaction(txHash) {
        this._transactionTrackingTimes[txHash] = Date.now();
    }
    /**
     * Returns the time that we have been waiting for a transaction to confirm
     */
    getTransactionTrackingTime(txHash) {
        const start = this._transactionTrackingTimes[txHash] ?? this._defaultStart;
        return Date.now() - start;
    }
}
exports.TransactionTrackingTimer = TransactionTrackingTimer;
//# sourceMappingURL=transaction-tracking-timer.js.map