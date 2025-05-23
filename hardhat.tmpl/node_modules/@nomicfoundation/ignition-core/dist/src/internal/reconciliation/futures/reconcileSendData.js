"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileSendData = void 0;
const future_resolvers_1 = require("../../execution/future-processor/helpers/future-resolvers");
const compare_1 = require("../helpers/compare");
const reconcile_data_1 = require("../helpers/reconcile-data");
const reconcile_from_1 = require("../helpers/reconcile-from");
const reconcile_strategy_1 = require("../helpers/reconcile-strategy");
const reconcile_value_1 = require("../helpers/reconcile-value");
function reconcileSendData(future, executionState, context) {
    const resolvedAddress = (0, future_resolvers_1.resolveSendToAddress)(future.to, context.deploymentState, context.deploymentParameters, context.accounts);
    let result = (0, compare_1.compare)(future, 'Address "to"', executionState.to, resolvedAddress);
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
    result = (0, reconcile_data_1.reconcileData)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_strategy_1.reconcileStrategy)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    return { success: true };
}
exports.reconcileSendData = reconcileSendData;
//# sourceMappingURL=reconcileSendData.js.map