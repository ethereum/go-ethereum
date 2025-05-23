"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.dropTransaction = void 0;
const utils_1 = require("../utils");
/**
 * Removes the given transaction from the mempool, if it exists.
 *
 * @param txHash Transaction hash to be removed from the mempool.
 * @returns `true` if successful, otherwise `false`.
 * @throws if the transaction was already mined.
 */
async function dropTransaction(txHash) {
    const provider = await (0, utils_1.getHardhatProvider)();
    (0, utils_1.assertTxHash)(txHash);
    return (await provider.request({
        method: "hardhat_dropTransaction",
        params: [txHash],
    }));
}
exports.dropTransaction = dropTransaction;
//# sourceMappingURL=dropTransaction.js.map