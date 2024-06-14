"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.mineUpTo = void 0;
const utils_1 = require("../utils");
const latestBlock_1 = require("./time/latestBlock");
/**
 * Mines new blocks until the latest block number is `blockNumber`
 *
 * @param blockNumber Must be greater than the latest block's number
 */
async function mineUpTo(blockNumber) {
    const provider = await (0, utils_1.getHardhatProvider)();
    const normalizedBlockNumber = (0, utils_1.toBigInt)(blockNumber);
    const latestHeight = BigInt(await (0, latestBlock_1.latestBlock)());
    (0, utils_1.assertLargerThan)(normalizedBlockNumber, latestHeight, "block number");
    const blockParam = normalizedBlockNumber - latestHeight;
    await provider.request({
        method: "hardhat_mine",
        params: [(0, utils_1.toRpcQuantity)(blockParam)],
    });
}
exports.mineUpTo = mineUpTo;
//# sourceMappingURL=mineUpTo.js.map