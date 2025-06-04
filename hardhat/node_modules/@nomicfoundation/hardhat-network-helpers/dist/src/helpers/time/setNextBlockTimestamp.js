"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.setNextBlockTimestamp = void 0;
const utils_1 = require("../../utils");
const duration_1 = require("./duration");
/**
 * Sets the timestamp of the next block but doesn't mine one.
 *
 * @param timestamp Can be `Date` or Epoch seconds. Must be greater than the latest block's timestamp
 */
async function setNextBlockTimestamp(timestamp) {
    const provider = await (0, utils_1.getHardhatProvider)();
    const timestampRpc = (0, utils_1.toRpcQuantity)(timestamp instanceof Date ? (0, duration_1.millis)(timestamp.valueOf()) : timestamp);
    await provider.request({
        method: "evm_setNextBlockTimestamp",
        params: [timestampRpc],
    });
}
exports.setNextBlockTimestamp = setNextBlockTimestamp;
//# sourceMappingURL=setNextBlockTimestamp.js.map