"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.mine = void 0;
const utils_1 = require("../utils");
/**
 * Mines a specified number of blocks at a given interval
 *
 * @param blocks Number of blocks to mine
 * @param options.interval Configures the interval (in seconds) between the timestamps of each mined block. Defaults to 1.
 */
async function mine(blocks = 1, options = {}) {
    const provider = await (0, utils_1.getHardhatProvider)();
    const interval = options.interval ?? 1;
    const blocksHex = (0, utils_1.toRpcQuantity)(blocks);
    const intervalHex = (0, utils_1.toRpcQuantity)(interval);
    await provider.request({
        method: "hardhat_mine",
        params: [blocksHex, intervalHex],
    });
}
exports.mine = mine;
//# sourceMappingURL=mine.js.map