"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.years = void 0;
const days_1 = require("./days");
/**
 * Converts years into seconds
 */
function years(n) {
    return (0, days_1.days)(n) * 365;
}
exports.years = years;
//# sourceMappingURL=years.js.map