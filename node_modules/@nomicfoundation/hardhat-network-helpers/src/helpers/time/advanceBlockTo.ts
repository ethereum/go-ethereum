import type { NumberLike } from "../../types";

import { mineUpTo } from "../mineUpTo";

/**
 * Mines new blocks until the latest block number is `blockNumber`
 *
 * @param blockNumber Must be greater than the latest block's number
 * @deprecated Use `helpers.mineUpTo` instead.
 */
export async function advanceBlockTo(blockNumber: NumberLike): Promise<void> {
  return mineUpTo(blockNumber);
}
