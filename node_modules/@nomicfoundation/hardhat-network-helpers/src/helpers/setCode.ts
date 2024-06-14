import {
  getHardhatProvider,
  assertValidAddress,
  assertHexString,
} from "../utils";

/**
 * Modifies the bytecode stored at an account's address
 *
 * @param address The address where the given code should be stored
 * @param code The code to store
 */
export async function setCode(address: string, code: string): Promise<void> {
  const provider = await getHardhatProvider();

  assertValidAddress(address);
  assertHexString(code);

  await provider.request({
    method: "hardhat_setCode",
    params: [address, code],
  });
}
