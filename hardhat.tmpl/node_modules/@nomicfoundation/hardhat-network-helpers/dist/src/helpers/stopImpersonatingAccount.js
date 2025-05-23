"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.stopImpersonatingAccount = void 0;
const utils_1 = require("../utils");
/**
 * Stops Hardhat Network from impersonating the given address
 *
 * @param address The address to stop impersonating
 */
async function stopImpersonatingAccount(address) {
    const provider = await (0, utils_1.getHardhatProvider)();
    (0, utils_1.assertValidAddress)(address);
    await provider.request({
        method: "hardhat_stopImpersonatingAccount",
        params: [address],
    });
}
exports.stopImpersonatingAccount = stopImpersonatingAccount;
//# sourceMappingURL=stopImpersonatingAccount.js.map