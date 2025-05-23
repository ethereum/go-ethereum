"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.setStorageAt = void 0;
const utils_1 = require("../utils");
/**
 * Writes a single position of an account's storage
 *
 * @param address The address where the code should be stored
 * @param index The index in storage
 * @param value The value to store
 */
async function setStorageAt(address, index, value) {
    const provider = await (0, utils_1.getHardhatProvider)();
    (0, utils_1.assertValidAddress)(address);
    const indexParam = (0, utils_1.toRpcQuantity)(index);
    const codeParam = (0, utils_1.toPaddedRpcQuantity)(value, 32);
    await provider.request({
        method: "hardhat_setStorageAt",
        params: [address, indexParam, codeParam],
    });
}
exports.setStorageAt = setStorageAt;
//# sourceMappingURL=setStorageAt.js.map