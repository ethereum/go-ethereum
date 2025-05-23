"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileFrom = void 0;
const future_resolvers_1 = require("../../execution/future-processor/helpers/future-resolvers");
const compare_1 = require("./compare");
function reconcileFrom(future, exState, context) {
    if (future.from === undefined && context.accounts.includes(exState.from)) {
        return undefined;
    }
    const resolvedFrom = (0, future_resolvers_1.resolveFutureFrom)(future.from, context.accounts, context.defaultSender);
    return (0, compare_1.compare)(future, "From account", exState.from, resolvedFrom);
}
exports.reconcileFrom = reconcileFrom;
//# sourceMappingURL=reconcile-from.js.map