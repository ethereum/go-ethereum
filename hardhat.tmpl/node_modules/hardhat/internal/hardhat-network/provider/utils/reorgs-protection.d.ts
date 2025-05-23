/**
 * This function returns a number that should be safe to consider as the
 * largest possible reorg in a network.
 *
 * If there's not such a number, or we aren't aware of it, this function
 * returns undefined.
 */
export declare function getLargestPossibleReorg(networkId: number): bigint | undefined;
export declare const FALLBACK_MAX_REORG = 128n;
//# sourceMappingURL=reorgs-protection.d.ts.map