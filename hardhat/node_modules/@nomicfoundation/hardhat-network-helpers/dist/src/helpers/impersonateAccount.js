"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.impersonateAccount = void 0;
const utils_1 = require("../utils");
/**
 * Allows Hardhat Network to sign transactions as the given address
 *
 * @param address The address to impersonate
 */
async function impersonateAccount(address) {
    const provider = await (0, utils_1.getHardhatProvider)();
    (0, utils_1.assertValidAddress)(address);
    await provider.request({
        method: "hardhat_impersonateAccount",
        params: [address],
    });
}
exports.impersonateAccount = impersonateAccount;
//# sourceMappingURL=impersonateAccount.js.map