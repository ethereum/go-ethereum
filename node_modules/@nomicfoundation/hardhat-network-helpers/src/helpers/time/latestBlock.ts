import { getHardhatProvider } from "../../utils";

/**
 * Returns the number of the latest block
 */
export async function latestBlock(): Promise<number> {
  const provider = await getHardhatProvider();

  const height = (await provider.request({
    method: "eth_blockNumber",
    params: [],
  })) as string;

  return parseInt(height, 16);
}
