"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileNamedStaticCall = void 0;
const compare_1 = require("../helpers/compare");
const reconcile_arguments_1 = require("../helpers/reconcile-arguments");
const reconcile_contract_1 = require("../helpers/reconcile-contract");
const reconcile_from_1 = require("../helpers/reconcile-from");
const reconcile_function_name_1 = require("../helpers/reconcile-function-name");
const reconcile_strategy_1 = require("../helpers/reconcile-strategy");
function reconcileNamedStaticCall(future, executionState, context) {
    let result = (0, reconcile_contract_1.reconcileContract)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_function_name_1.reconcileFunctionName)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_arguments_1.reconcileArguments)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_from_1.reconcileFrom)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_strategy_1.reconcileStrategy)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, compare_1.compare)(future, "Argument name or index", executionState.nameOrIndex, future.nameOrIndex);
    if (result !== undefined) {
        return result;
    }
    return { success: true };
}
exports.reconcileNamedStaticCall = reconcileNamedStaticCall;
//# sourceMappingURL=reconcileNamedStaticCall.js.map