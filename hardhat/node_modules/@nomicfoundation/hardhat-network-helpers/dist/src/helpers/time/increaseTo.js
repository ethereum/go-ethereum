"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.increaseTo = void 0;
const utils_1 = require("../../utils");
const mine_1 = require("../mine");
const duration_1 = require("./duration");
/**
 * Mines a new block whose timestamp is `timestamp`
 *
 * @param timestamp Can be `Date` or Epoch seconds. Must be bigger than the latest block's timestamp
 */
async function increaseTo(timestamp) {
    const provider = await (0, utils_1.getHardhatProvider)();
    const normalizedTimestamp = (0, utils_1.toBigInt)(timestamp instanceof Date ? (0, duration_1.millis)(timestamp.valueOf()) : timestamp);
    await provider.request({
        method: "evm_setNextBlockTimestamp",
        params: [(0, utils_1.toRpcQuantity)(normalizedTimestamp)],
    });
    await (0, mine_1.mine)();
}
exports.increaseTo = increaseTo;
//# sourceMappingURL=increaseTo.js.map