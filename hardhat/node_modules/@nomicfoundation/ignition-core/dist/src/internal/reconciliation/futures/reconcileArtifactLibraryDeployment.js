"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileArtifactLibraryDeployment = void 0;
const reconcile_artifacts_1 = require("../helpers/reconcile-artifacts");
const reconcile_contract_name_1 = require("../helpers/reconcile-contract-name");
const reconcile_from_1 = require("../helpers/reconcile-from");
const reconcile_libraries_1 = require("../helpers/reconcile-libraries");
const reconcile_strategy_1 = require("../helpers/reconcile-strategy");
async function reconcileArtifactLibraryDeployment(future, executionState, context) {
    let result = (0, reconcile_contract_name_1.reconcileContractName)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = await (0, reconcile_artifacts_1.reconcileArtifacts)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_libraries_1.reconcileLibraries)(future, executionState, context);
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
exports.reconcileArtifactLibraryDeployment = reconcileArtifactLibraryDeployment;
//# sourceMappingURL=reconcileArtifactLibraryDeployment.js.map