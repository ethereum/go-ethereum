"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.days = void 0;
const hours_1 = require("./hours");
/**
 * Converts days into seconds
 */
function days(n) {
    return (0, hours_1.hours)(n) * 24;
}
exports.days = days;
//# sourceMappingURL=days.js.map