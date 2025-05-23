"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileNamedContractCall = void 0;
const reconcile_arguments_1 = require("../helpers/reconcile-arguments");
const reconcile_contract_1 = require("../helpers/reconcile-contract");
const reconcile_from_1 = require("../helpers/reconcile-from");
const reconcile_function_name_1 = require("../helpers/reconcile-function-name");
const reconcile_strategy_1 = require("../helpers/reconcile-strategy");
const reconcile_value_1 = require("../helpers/reconcile-value");
function reconcileNamedContractCall(future, executionState, context) {
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
    result = (0, reconcile_value_1.reconcileValue)(future, executionState, context);
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
    return { success: true };
}
exports.reconcileNamedContractCall = reconcileNamedContractCall;
//# sourceMappingURL=reconcileNamedContractCall.js.map