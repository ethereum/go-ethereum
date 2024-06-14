import type { NumberLike } from "../types";

import {
  getHardhatProvider,
  toRpcQuantity,
  assertLargerThan,
  toBigInt,
} from "../utils";

import { latestBlock } from "./time/latestBlock";

/**
 * Mines new blocks until the latest block number is `blockNumber`
 *
 * @param blockNumber Must be greater than the latest block's number
 */
export async function mineUpTo(blockNumber: NumberLike): Promise<void> {
  const provider = await getHardhatProvider();

  const normalizedBlockNumber = toBigInt(blockNumber);
  const latestHeight = BigInt(await latestBlock());

  assertLargerThan(normalizedBlockNumber, latestHeight, "block number");

  const blockParam = normalizedBlockNumber - latestHeight;

  await provider.request({
    method: "hardhat_mine",
    params: [toRpcQuantity(blockParam)],
  });
}
