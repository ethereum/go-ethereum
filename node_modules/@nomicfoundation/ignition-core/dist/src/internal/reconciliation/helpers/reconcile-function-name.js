"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileFunctionName = void 0;
const compare_1 = require("./compare");
function reconcileFunctionName(future, exState, _context) {
    return (0, compare_1.compare)(future, "Function name", exState.functionName, future.functionName);
}
exports.reconcileFunctionName = reconcileFunctionName;
//# sourceMappingURL=reconcile-function-name.js.map