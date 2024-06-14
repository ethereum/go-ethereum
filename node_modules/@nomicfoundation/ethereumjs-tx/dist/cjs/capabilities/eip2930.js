"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getDataFee = void 0;
const util_js_1 = require("../util.js");
const Legacy = require("./legacy.js");
/**
 * The amount of gas paid for the data in this tx
 */
function getDataFee(tx) {
    return Legacy.getDataFee(tx, BigInt(util_js_1.AccessLists.getDataFeeEIP2930(tx.accessList, tx.common)));
}
exports.getDataFee = getDataFee;
//# sourceMappingURL=eip2930.js.map