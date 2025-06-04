"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.hours = void 0;
const minutes_1 = require("./minutes");
/**
 * Converts hours into seconds
 */
function hours(n) {
    return (0, minutes_1.minutes)(n) * 60;
}
exports.hours = hours;
//# sourceMappingURL=hours.js.map