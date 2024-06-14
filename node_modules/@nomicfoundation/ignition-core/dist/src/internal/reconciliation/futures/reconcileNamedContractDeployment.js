"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileNamedContractDeployment = void 0;
const reconcile_arguments_1 = require("../helpers/reconcile-arguments");
const reconcile_artifacts_1 = require("../helpers/reconcile-artifacts");
const reconcile_contract_name_1 = require("../helpers/reconcile-contract-name");
const reconcile_from_1 = require("../helpers/reconcile-from");
const reconcile_libraries_1 = require("../helpers/reconcile-libraries");
const reconcile_strategy_1 = require("../helpers/reconcile-strategy");
const reconcile_value_1 = require("../helpers/reconcile-value");
async function reconcileNamedContractDeployment(future, executionState, context) {
    let result = (0, reconcile_contract_name_1.reconcileContractName)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = await (0, reconcile_artifacts_1.reconcileArtifacts)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_arguments_1.reconcileArguments)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_libraries_1.reconcileLibraries)(future, executionState, context);
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
    return {
        success: true,
    };
}
exports.reconcileNamedContractDeployment = reconcileNamedContractDeployment;
//# sourceMappingURL=reconcileNamedContractDeployment.js.map