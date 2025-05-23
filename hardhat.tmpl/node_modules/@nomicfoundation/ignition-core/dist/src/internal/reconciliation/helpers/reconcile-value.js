"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileValue = void 0;
const future_resolvers_1 = require("../../execution/future-processor/helpers/future-resolvers");
const compare_1 = require("./compare");
function reconcileValue(future, exState, context) {
    const resolvedValue = (0, future_resolvers_1.resolveValue)(future.value, context.deploymentParameters, context.deploymentState, context.accounts);
    return (0, compare_1.compare)(future, "Value", exState.value, resolvedValue);
}
exports.reconcileValue = reconcileValue;
//# sourceMappingURL=reconcile-value.js.map