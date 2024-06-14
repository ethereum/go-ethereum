"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getStorageAt = void 0;
const utils_1 = require("../utils");
/**
 * Retrieves the data located at the given address, index, and block number
 *
 * @param address The address to retrieve storage from
 * @param index The position in storage
 * @param block The block number, or one of `"latest"`, `"earliest"`, or `"pending"`. Defaults to `"latest"`.
 * @returns string containing the hexadecimal code retrieved
 */
async function getStorageAt(address, index, block = "latest") {
    const provider = await (0, utils_1.getHardhatProvider)();
    (0, utils_1.assertValidAddress)(address);
    const indexParam = (0, utils_1.toPaddedRpcQuantity)(index, 32);
    let blockParam;
    switch (block) {
        case "latest":
        case "earliest":
        case "pending":
            blockParam = block;
            break;
        default:
            blockParam = (0, utils_1.toRpcQuantity)(block);
    }
    const data = await provider.request({
        method: "eth_getStorageAt",
        params: [address, indexParam, blockParam],
    });
    return data;
}
exports.getStorageAt = getStorageAt;
//# sourceMappingURL=getStorageAt.js.map