/**
 * Removes the given transaction from the mempool, if it exists.
 *
 * @param txHash Transaction hash to be removed from the mempool.
 * @returns `true` if successful, otherwise `false`.
 * @throws if the transaction was already mined.
 */
export declare function dropTransaction(txHash: string): Promise<boolean>;
//# sourceMappingURL=dropTransaction.d.ts.map