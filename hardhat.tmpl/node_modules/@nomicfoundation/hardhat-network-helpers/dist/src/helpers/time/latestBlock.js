"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.latestBlock = void 0;
const utils_1 = require("../../utils");
/**
 * Returns the number of the latest block
 */
async function latestBlock() {
    const provider = await (0, utils_1.getHardhatProvider)();
    const height = (await provider.request({
        method: "eth_blockNumber",
        params: [],
    }));
    return parseInt(height, 16);
}
exports.latestBlock = latestBlock;
//# sourceMappingURL=latestBlock.js.map