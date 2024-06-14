import type { NumberLike } from "../types";
import {
  getHardhatProvider,
  assertValidAddress,
  toRpcQuantity,
  toPaddedRpcQuantity,
} from "../utils";

/**
 * Writes a single position of an account's storage
 *
 * @param address The address where the code should be stored
 * @param index The index in storage
 * @param value The value to store
 */
export async function setStorageAt(
  address: string,
  index: NumberLike,
  value: NumberLike
): Promise<void> {
  const provider = await getHardhatProvider();

  assertValidAddress(address);
  const indexParam = toRpcQuantity(index);
  const codeParam = toPaddedRpcQuantity(value, 32);

  await provider.request({
    method: "hardhat_setStorageAt",
    params: [address, indexParam, codeParam],
  });
}
