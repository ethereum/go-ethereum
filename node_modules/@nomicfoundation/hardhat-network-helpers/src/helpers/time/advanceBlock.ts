import type { NumberLike } from "../../types";

import { mine } from "../mine";

import { latestBlock } from "./latestBlock";

/**
 * Mines `numberOfBlocks` new blocks.
 *
 * @param numberOfBlocks Must be greater than 0
 * @returns number of the latest block mined
 *
 * @deprecated Use `helpers.mine` instead.
 */
export async function advanceBlock(
  numberOfBlocks: NumberLike = 1
): Promise<number> {
  await mine(numberOfBlocks);

  return latestBlock();
}
