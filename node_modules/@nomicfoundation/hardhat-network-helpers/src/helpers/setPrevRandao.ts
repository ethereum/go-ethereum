import type { NumberLike } from "../types";
import { getHardhatProvider, toPaddedRpcQuantity } from "../utils";

/**
 * Sets the PREVRANDAO value of the next block.
 *
 * @param prevRandao The new PREVRANDAO value to use.
 */
export async function setPrevRandao(prevRandao: NumberLike): Promise<void> {
  const provider = await getHardhatProvider();

  const paddedPrevRandao = toPaddedRpcQuantity(prevRandao, 32);

  await provider.request({
    method: "hardhat_setPrevRandao",
    params: [paddedPrevRandao],
  });
}
