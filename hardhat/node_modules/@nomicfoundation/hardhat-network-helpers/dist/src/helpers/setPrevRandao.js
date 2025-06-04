"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.setPrevRandao = void 0;
const utils_1 = require("../utils");
/**
 * Sets the PREVRANDAO value of the next block.
 *
 * @param prevRandao The new PREVRANDAO value to use.
 */
async function setPrevRandao(prevRandao) {
    const provider = await (0, utils_1.getHardhatProvider)();
    const paddedPrevRandao = (0, utils_1.toPaddedRpcQuantity)(prevRandao, 32);
    await provider.request({
        method: "hardhat_setPrevRandao",
        params: [paddedPrevRandao],
    });
}
exports.setPrevRandao = setPrevRandao;
//# sourceMappingURL=setPrevRandao.js.map