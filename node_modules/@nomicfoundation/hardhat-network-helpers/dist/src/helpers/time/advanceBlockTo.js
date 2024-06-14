"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.advanceBlockTo = void 0;
const mineUpTo_1 = require("../mineUpTo");
/**
 * Mines new blocks until the latest block number is `blockNumber`
 *
 * @param blockNumber Must be greater than the latest block's number
 * @deprecated Use `helpers.mineUpTo` instead.
 */
async function advanceBlockTo(blockNumber) {
    return (0, mineUpTo_1.mineUpTo)(blockNumber);
}
exports.advanceBlockTo = advanceBlockTo;
//# sourceMappingURL=advanceBlockTo.js.map