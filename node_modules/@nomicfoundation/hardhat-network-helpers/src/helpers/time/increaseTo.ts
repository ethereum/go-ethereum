import type { NumberLike } from "../../types";

import { getHardhatProvider, toRpcQuantity, toBigInt } from "../../utils";
import { mine } from "../mine";
import { millis } from "./duration";

/**
 * Mines a new block whose timestamp is `timestamp`
 *
 * @param timestamp Can be `Date` or Epoch seconds. Must be bigger than the latest block's timestamp
 */
export async function increaseTo(timestamp: NumberLike | Date): Promise<void> {
  const provider = await getHardhatProvider();

  const normalizedTimestamp = toBigInt(
    timestamp instanceof Date ? millis(timestamp.valueOf()) : timestamp
  );

  await provider.request({
    method: "evm_setNextBlockTimestamp",
    params: [toRpcQuantity(normalizedTimestamp)],
  });

  await mine();
}
