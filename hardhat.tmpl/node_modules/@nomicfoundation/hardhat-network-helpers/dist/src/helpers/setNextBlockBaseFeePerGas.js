"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.setNextBlockBaseFeePerGas = void 0;
const utils_1 = require("../utils");
/**
 * Sets the base fee of the next block.
 *
 * @param baseFeePerGas The new base fee to use.
 */
async function setNextBlockBaseFeePerGas(baseFeePerGas) {
    const provider = await (0, utils_1.getHardhatProvider)();
    const baseFeePerGasHex = (0, utils_1.toRpcQuantity)(baseFeePerGas);
    await provider.request({
        method: "hardhat_setNextBlockBaseFeePerGas",
        params: [baseFeePerGasHex],
    });
}
exports.setNextBlockBaseFeePerGas = setNextBlockBaseFeePerGas;
//# sourceMappingURL=setNextBlockBaseFeePerGas.js.map