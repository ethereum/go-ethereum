import { getHardhatProvider, assertValidAddress } from "../utils";

/**
 * Allows Hardhat Network to sign transactions as the given address
 *
 * @param address The address to impersonate
 */
export async function impersonateAccount(address: string): Promise<void> {
  const provider = await getHardhatProvider();

  assertValidAddress(address);

  await provider.request({
    method: "hardhat_impersonateAccount",
    params: [address],
  });
}
