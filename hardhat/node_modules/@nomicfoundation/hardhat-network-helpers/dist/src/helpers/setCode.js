"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.setCode = void 0;
const utils_1 = require("../utils");
/**
 * Modifies the bytecode stored at an account's address
 *
 * @param address The address where the given code should be stored
 * @param code The code to store
 */
async function setCode(address, code) {
    const provider = await (0, utils_1.getHardhatProvider)();
    (0, utils_1.assertValidAddress)(address);
    (0, utils_1.assertHexString)(code);
    await provider.request({
        method: "hardhat_setCode",
        params: [address, code],
    });
}
exports.setCode = setCode;
//# sourceMappingURL=setCode.js.map