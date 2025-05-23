"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileNamedEncodeFunctionCall = void 0;
const reconcile_arguments_1 = require("../helpers/reconcile-arguments");
const reconcile_function_name_1 = require("../helpers/reconcile-function-name");
const reconcile_strategy_1 = require("../helpers/reconcile-strategy");
function reconcileNamedEncodeFunctionCall(future, executionState, context) {
    let result = (0, reconcile_function_name_1.reconcileFunctionName)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_arguments_1.reconcileArguments)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_strategy_1.reconcileStrategy)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    return { success: true };
}
exports.reconcileNamedEncodeFunctionCall = reconcileNamedEncodeFunctionCall;
//# sourceMappingURL=reconcileNamedEncodeFunctionCall.js.map