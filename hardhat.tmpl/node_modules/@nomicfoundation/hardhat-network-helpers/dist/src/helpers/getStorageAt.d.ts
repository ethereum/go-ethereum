import type { NumberLike, BlockTag } from "../types";
/**
 * Retrieves the data located at the given address, index, and block number
 *
 * @param address The address to retrieve storage from
 * @param index The position in storage
 * @param block The block number, or one of `"latest"`, `"earliest"`, or `"pending"`. Defaults to `"latest"`.
 * @returns string containing the hexadecimal code retrieved
 */
export declare function getStorageAt(address: string, index: NumberLike, block?: NumberLike | BlockTag): Promise<string>;
//# sourceMappingURL=getStorageAt.d.ts.map