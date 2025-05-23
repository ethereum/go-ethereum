"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.setCoinbase = void 0;
const utils_1 = require("../utils");
/**
 * Sets the coinbase address to be used in new blocks
 *
 * @param address The new coinbase address
 */
async function setCoinbase(address) {
    const provider = await (0, utils_1.getHardhatProvider)();
    (0, utils_1.assertValidAddress)(address);
    await provider.request({
        method: "hardhat_setCoinbase",
        params: [address],
    });
}
exports.setCoinbase = setCoinbase;
//# sourceMappingURL=setCoinbase.js.map