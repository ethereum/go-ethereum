"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.formatBigDecimal = exports.GWEI_TO_WEI = void 0;
const constants_js_1 = require("./constants.js");
/** Easy conversion from Gwei to wei */
exports.GWEI_TO_WEI = BigInt(1000000000);
function formatBigDecimal(numerator, denominator, maxDecimalFactor) {
    if (denominator === constants_js_1.BIGINT_0) {
        denominator = constants_js_1.BIGINT_1;
    }
    const full = numerator / denominator;
    const fraction = ((numerator - full * denominator) * maxDecimalFactor) / denominator;
    // zeros to be added post decimal are number of zeros in maxDecimalFactor - number of digits in fraction
    const zerosPostDecimal = String(maxDecimalFactor).length - 1 - String(fraction).length;
    return `${full}.${'0'.repeat(zerosPostDecimal)}${fraction}`;
}
exports.formatBigDecimal = formatBigDecimal;
//# sourceMappingURL=units.js.map