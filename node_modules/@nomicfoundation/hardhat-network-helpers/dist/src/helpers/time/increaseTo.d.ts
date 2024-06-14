import type { NumberLike } from "../../types";
/**
 * Mines a new block whose timestamp is `timestamp`
 *
 * @param timestamp Can be `Date` or Epoch seconds. Must be bigger than the latest block's timestamp
 */
export declare function increaseTo(timestamp: NumberLike | Date): Promise<void>;
//# sourceMappingURL=increaseTo.d.ts.map