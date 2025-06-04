import type { NumberLike } from "../../types";
/**
 * Sets the timestamp of the next block but doesn't mine one.
 *
 * @param timestamp Can be `Date` or Epoch seconds. Must be greater than the latest block's timestamp
 */
export declare function setNextBlockTimestamp(timestamp: NumberLike | Date): Promise<void>;
//# sourceMappingURL=setNextBlockTimestamp.d.ts.map