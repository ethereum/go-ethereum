"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.setBlockGasLimit = void 0;
const utils_1 = require("../utils");
/**
 * Sets the gas limit for future blocks
 *
 * @param blockGasLimit The gas limit to set for future blocks
 */
async function setBlockGasLimit(blockGasLimit) {
    const provider = await (0, utils_1.getHardhatProvider)();
    const blockGasLimitHex = (0, utils_1.toRpcQuantity)(blockGasLimit);
    await provider.request({
        method: "evm_setBlockGasLimit",
        params: [blockGasLimitHex],
    });
}
exports.setBlockGasLimit = setBlockGasLimit;
//# sourceMappingURL=setBlockGasLimit.js.map