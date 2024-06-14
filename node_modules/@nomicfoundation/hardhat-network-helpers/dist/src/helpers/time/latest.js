"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.latest = void 0;
const utils_1 = require("../../utils");
/**
 * Returns the timestamp of the latest block
 */
async function latest() {
    const provider = await (0, utils_1.getHardhatProvider)();
    const latestBlock = (await provider.request({
        method: "eth_getBlockByNumber",
        params: ["latest", false],
    }));
    return parseInt(latestBlock.timestamp, 16);
}
exports.latest = latest;
//# sourceMappingURL=latest.js.map