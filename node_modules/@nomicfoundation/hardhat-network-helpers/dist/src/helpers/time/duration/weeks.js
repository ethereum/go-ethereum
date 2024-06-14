"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.weeks = void 0;
const days_1 = require("./days");
/**
 * Converts weeks into seconds
 */
function weeks(n) {
    return (0, days_1.days)(n) * 7;
}
exports.weeks = weeks;
//# sourceMappingURL=weeks.js.map