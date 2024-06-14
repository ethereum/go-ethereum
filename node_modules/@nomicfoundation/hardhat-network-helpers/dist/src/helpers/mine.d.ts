import type { NumberLike } from "../types";
/**
 * Mines a specified number of blocks at a given interval
 *
 * @param blocks Number of blocks to mine
 * @param options.interval Configures the interval (in seconds) between the timestamps of each mined block. Defaults to 1.
 */
export declare function mine(blocks?: NumberLike, options?: {
    interval?: NumberLike;
}): Promise<void>;
//# sourceMappingURL=mine.d.ts.map