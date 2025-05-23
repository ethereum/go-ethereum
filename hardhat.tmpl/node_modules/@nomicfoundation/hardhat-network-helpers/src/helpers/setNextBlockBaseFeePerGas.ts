import type { NumberLike } from "../types";
import { getHardhatProvider, toRpcQuantity } from "../utils";

/**
 * Sets the base fee of the next block.
 *
 * @param baseFeePerGas The new base fee to use.
 */
export async function setNextBlockBaseFeePerGas(
  baseFeePerGas: NumberLike
): Promise<void> {
  const provider = await getHardhatProvider();

  const baseFeePerGasHex = toRpcQuantity(baseFeePerGas);

  await provider.request({
    method: "hardhat_setNextBlockBaseFeePerGas",
    params: [baseFeePerGasHex],
  });
}
