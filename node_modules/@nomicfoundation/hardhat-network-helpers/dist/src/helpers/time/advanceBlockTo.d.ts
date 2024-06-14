import type { NumberLike } from "../../types";
/**
 * Mines new blocks until the latest block number is `blockNumber`
 *
 * @param blockNumber Must be greater than the latest block's number
 * @deprecated Use `helpers.mineUpTo` instead.
 */
export declare function advanceBlockTo(blockNumber: NumberLike): Promise<void>;
//# sourceMappingURL=advanceBlockTo.d.ts.map