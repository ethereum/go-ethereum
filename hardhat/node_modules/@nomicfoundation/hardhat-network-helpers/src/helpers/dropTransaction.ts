import { getHardhatProvider, assertTxHash } from "../utils";

/**
 * Removes the given transaction from the mempool, if it exists.
 *
 * @param txHash Transaction hash to be removed from the mempool.
 * @returns `true` if successful, otherwise `false`.
 * @throws if the transaction was already mined.
 */
export async function dropTransaction(txHash: string): Promise<boolean> {
  const provider = await getHardhatProvider();

  assertTxHash(txHash);

  return (await provider.request({
    method: "hardhat_dropTransaction",
    params: [txHash],
  })) as boolean;
}
