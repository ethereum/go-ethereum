import { getHardhatProvider } from "../../utils";

/**
 * Returns the timestamp of the latest block
 */
export async function latest(): Promise<number> {
  const provider = await getHardhatProvider();

  const latestBlock = (await provider.request({
    method: "eth_getBlockByNumber",
    params: ["latest", false],
  })) as { timestamp: string };

  return parseInt(latestBlock.timestamp, 16);
}
