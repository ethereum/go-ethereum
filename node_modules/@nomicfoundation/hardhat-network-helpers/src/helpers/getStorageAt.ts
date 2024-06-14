import type { NumberLike, BlockTag } from "../types";
import {
  getHardhatProvider,
  assertValidAddress,
  toPaddedRpcQuantity,
  toRpcQuantity,
} from "../utils";

/**
 * Retrieves the data located at the given address, index, and block number
 *
 * @param address The address to retrieve storage from
 * @param index The position in storage
 * @param block The block number, or one of `"latest"`, `"earliest"`, or `"pending"`. Defaults to `"latest"`.
 * @returns string containing the hexadecimal code retrieved
 */
export async function getStorageAt(
  address: string,
  index: NumberLike,
  block: NumberLike | BlockTag = "latest"
): Promise<string> {
  const provider = await getHardhatProvider();

  assertValidAddress(address);
  const indexParam = toPaddedRpcQuantity(index, 32);

  let blockParam: NumberLike | BlockTag;
  switch (block) {
    case "latest":
    case "earliest":
    case "pending":
      blockParam = block;
      break;
    default:
      blockParam = toRpcQuantity(block);
  }

  const data = await provider.request({
    method: "eth_getStorageAt",
    params: [address, indexParam, blockParam],
  });

  return data as string;
}
