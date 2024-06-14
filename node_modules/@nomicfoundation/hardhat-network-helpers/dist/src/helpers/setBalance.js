"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.setBalance = void 0;
const utils_1 = require("../utils");
/**
 * Sets the balance for the given address.
 *
 * @param address The address whose balance will be edited.
 * @param balance The new balance to set for the given address, in wei.
 */
async function setBalance(address, balance) {
    const provider = await (0, utils_1.getHardhatProvider)();
    (0, utils_1.assertValidAddress)(address);
    const balanceHex = (0, utils_1.toRpcQuantity)(balance);
    await provider.request({
        method: "hardhat_setBalance",
        params: [address, balanceHex],
    });
}
exports.setBalance = setBalance;
//# sourceMappingURL=setBalance.js.map