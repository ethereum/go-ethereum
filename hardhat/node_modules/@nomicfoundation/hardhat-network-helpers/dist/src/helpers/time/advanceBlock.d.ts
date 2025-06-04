import type { NumberLike } from "../../types";
/**
 * Mines `numberOfBlocks` new blocks.
 *
 * @param numberOfBlocks Must be greater than 0
 * @returns number of the latest block mined
 *
 * @deprecated Use `helpers.mine` instead.
 */
export declare function advanceBlock(numberOfBlocks?: NumberLike): Promise<number>;
//# sourceMappingURL=advanceBlock.d.ts.map