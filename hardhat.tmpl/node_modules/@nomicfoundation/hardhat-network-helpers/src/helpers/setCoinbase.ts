import { getHardhatProvider, assertValidAddress } from "../utils";

/**
 * Sets the coinbase address to be used in new blocks
 *
 * @param address The new coinbase address
 */
export async function setCoinbase(address: string): Promise<void> {
  const provider = await getHardhatProvider();

  assertValidAddress(address);

  await provider.request({
    method: "hardhat_setCoinbase",
    params: [address],
  });
}
