import type { NumberLike } from "../types";
import {
  getHardhatProvider,
  assertValidAddress,
  toRpcQuantity,
} from "../utils";

/**
 * Sets the balance for the given address.
 *
 * @param address The address whose balance will be edited.
 * @param balance The new balance to set for the given address, in wei.
 */
export async function setBalance(
  address: string,
  balance: NumberLike
): Promise<void> {
  const provider = await getHardhatProvider();

  assertValidAddress(address);
  const balanceHex = toRpcQuantity(balance);

  await provider.request({
    method: "hardhat_setBalance",
    params: [address, balanceHex],
  });
}
