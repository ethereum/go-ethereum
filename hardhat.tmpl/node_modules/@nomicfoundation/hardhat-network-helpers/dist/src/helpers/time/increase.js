"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.increase = void 0;
const utils_1 = require("../../utils");
const mine_1 = require("../mine");
const latest_1 = require("./latest");
/**
 * Mines a new block whose timestamp is `amountInSeconds` after the latest block's timestamp
 *
 * @param amountInSeconds number of seconds to increase the next block's timestamp by
 * @returns the timestamp of the mined block
 */
async function increase(amountInSeconds) {
    const provider = await (0, utils_1.getHardhatProvider)();
    const normalizedAmount = (0, utils_1.toBigInt)(amountInSeconds);
    (0, utils_1.assertNonNegativeNumber)(normalizedAmount);
    const latestTimestamp = BigInt(await (0, latest_1.latest)());
    const targetTimestamp = latestTimestamp + normalizedAmount;
    await provider.request({
        method: "evm_setNextBlockTimestamp",
        params: [(0, utils_1.toRpcQuantity)(targetTimestamp)],
    });
    await (0, mine_1.mine)();
    return (0, latest_1.latest)();
}
exports.increase = increase;
//# sourceMappingURL=increase.js.map