"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.compare = void 0;
const utils_1 = require("../utils");
function compare(future, fieldName, existingValue, newValue, messageSuffix) {
    if (existingValue !== newValue) {
        return (0, utils_1.fail)(future, `${fieldName} has been changed from ${existingValue?.toString() ?? '"undefined"'} to ${newValue?.toString() ?? '"undefined"'}${messageSuffix ?? ""}`);
    }
}
exports.compare = compare;
//# sourceMappingURL=compare.js.map