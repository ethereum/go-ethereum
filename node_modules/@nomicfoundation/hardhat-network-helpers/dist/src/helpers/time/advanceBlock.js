"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.advanceBlock = void 0;
const mine_1 = require("../mine");
const latestBlock_1 = require("./latestBlock");
/**
 * Mines `numberOfBlocks` new blocks.
 *
 * @param numberOfBlocks Must be greater than 0
 * @returns number of the latest block mined
 *
 * @deprecated Use `helpers.mine` instead.
 */
async function advanceBlock(numberOfBlocks = 1) {
    await (0, mine_1.mine)(numberOfBlocks);
    return (0, latestBlock_1.latestBlock)();
}
exports.advanceBlock = advanceBlock;
//# sourceMappingURL=advanceBlock.js.map