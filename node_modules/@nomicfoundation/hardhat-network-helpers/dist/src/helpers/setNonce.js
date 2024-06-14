"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.setNonce = void 0;
const utils_1 = require("../utils");
/**
 * Modifies an account's nonce by overwriting it
 *
 * @param address The address whose nonce is to be changed
 * @param nonce The new nonce
 */
async function setNonce(address, nonce) {
    const provider = await (0, utils_1.getHardhatProvider)();
    (0, utils_1.assertValidAddress)(address);
    const nonceHex = (0, utils_1.toRpcQuantity)(nonce);
    await provider.request({
        method: "hardhat_setNonce",
        params: [address, nonceHex],
    });
}
exports.setNonce = setNonce;
//# sourceMappingURL=setNonce.js.map