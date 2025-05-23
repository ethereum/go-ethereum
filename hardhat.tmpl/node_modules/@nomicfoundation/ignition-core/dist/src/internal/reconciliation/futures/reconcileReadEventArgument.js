"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileReadEventArgument = void 0;
const future_resolvers_1 = require("../../execution/future-processor/helpers/future-resolvers");
const compare_1 = require("../helpers/compare");
const reconcile_strategy_1 = require("../helpers/reconcile-strategy");
function reconcileReadEventArgument(future, executionState, context) {
    const resolvedAddress = (0, future_resolvers_1.resolveAddressForContractFuture)(future.emitter, context.deploymentState);
    let result = (0, compare_1.compare)(future, "Emitter", executionState.emitterAddress, resolvedAddress, ` (future ${future.emitter.id})`);
    if (result !== undefined) {
        return result;
    }
    result = (0, compare_1.compare)(future, "Event name", executionState.eventName, future.eventName);
    if (result !== undefined) {
        return result;
    }
    result = (0, compare_1.compare)(future, "Event index", executionState.eventIndex, future.eventIndex);
    if (result !== undefined) {
        return result;
    }
    result = (0, compare_1.compare)(future, "Argument name or index", executionState.nameOrIndex, future.nameOrIndex);
    if (result !== undefined) {
        return result;
    }
    result = (0, reconcile_strategy_1.reconcileStrategy)(future, executionState, context);
    if (result !== undefined) {
        return result;
    }
    return { success: true };
}
exports.reconcileReadEventArgument = reconcileReadEventArgument;
//# sourceMappingURL=reconcileReadEventArgument.js.map