"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileNamedContractAt = void 0;
const reconcile_address_1 = require("../helpers/reconcile-address");
const reconcile_artifacts_1 = require("../helpers/reconcile-artifacts");
const reconcile_contract_name_1 = require("../helpers/reconcile-contract-name");
const reconcile_strategy_1 = require("../helpers/reconcile-strategy");
async function reconcileNamedContractAt(future, executionState, context) {
    let result = (0, reconcile_contract_name_1.reconcileContractName)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = await (0, reconcile_artifacts_1.reconcileArtifacts)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_address_1.reconcileAddress)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_strategy_1.reconcileStrategy)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    return { success: true };
}
exports.reconcileNamedContractAt = reconcileNamedContractAt;
//# sourceMappingURL=reconcileNamedContractAt.js.map