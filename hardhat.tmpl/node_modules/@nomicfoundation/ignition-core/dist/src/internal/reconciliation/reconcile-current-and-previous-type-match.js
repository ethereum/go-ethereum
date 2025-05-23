"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileCurrentAndPreviousTypeMatch = void 0;
const module_1 = require("../../types/module");
const utils_1 = require("./utils");
function reconcileCurrentAndPreviousTypeMatch(future, executionState, _context) {
    if (executionState.futureType === future.type) {
        return { success: true };
    }
    return (0, utils_1.fail)(future, `Future with id ${future.id} has changed from ${module_1.FutureType[executionState.futureType]} to ${module_1.FutureType[future.type]}`);
}
exports.reconcileCurrentAndPreviousTypeMatch = reconcileCurrentAndPreviousTypeMatch;
//# sourceMappingURL=reconcile-current-and-previous-type-match.js.map