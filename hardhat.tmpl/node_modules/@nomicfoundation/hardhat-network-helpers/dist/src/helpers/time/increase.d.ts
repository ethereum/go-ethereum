import type { NumberLike } from "../../types";
/**
 * Mines a new block whose timestamp is `amountInSeconds` after the latest block's timestamp
 *
 * @param amountInSeconds number of seconds to increase the next block's timestamp by
 * @returns the timestamp of the mined block
 */
export declare function increase(amountInSeconds: NumberLike): Promise<number>;
//# sourceMappingURL=increase.d.ts.map