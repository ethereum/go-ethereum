/**
 * This class is used to track the time that we have been waiting for
 * a transaction to confirm since it was either sent, or since Ignition started
 * and it was already sent.
 *
 * Note: This class doesn't have a method to clear the timer for a transaction
 * but it shouldn't be problematic.
 */
export declare class TransactionTrackingTimer {
    private readonly _defaultStart;
    private readonly _transactionTrackingTimes;
    /**
     * Adds a new transaction to track.
     */
    addTransaction(txHash: string): void;
    /**
     * Returns the time that we have been waiting for a transaction to confirm
     */
    getTransactionTrackingTime(txHash: string): number;
}
//# sourceMappingURL=transaction-tracking-timer.d.ts.map