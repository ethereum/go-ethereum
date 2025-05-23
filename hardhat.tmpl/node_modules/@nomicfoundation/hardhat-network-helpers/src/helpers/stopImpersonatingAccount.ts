import { getHardhatProvider, assertValidAddress } from "../utils";

/**
 * Stops Hardhat Network from impersonating the given address
 *
 * @param address The address to stop impersonating
 */
export async function stopImpersonatingAccount(address: string): Promise<void> {
  const provider = await getHardhatProvider();

  assertValidAddress(address);

  await provider.request({
    method: "hardhat_stopImpersonatingAccount",
    params: [address],
  });
}
