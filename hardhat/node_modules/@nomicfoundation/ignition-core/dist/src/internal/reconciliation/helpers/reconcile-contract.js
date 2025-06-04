"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileContract = void 0;
const future_resolvers_1 = require("../../execution/future-processor/helpers/future-resolvers");
const compare_1 = require("./compare");
function reconcileContract(future, exState, context) {
    const resolvedAddress = (0, future_resolvers_1.resolveAddressLike)(future.contract, context.deploymentState, context.deploymentParameters, context.accounts);
    return (0, compare_1.compare)(future, "Contract address", exState.contractAddress, resolvedAddress, ` (future ${future.contract.id})`);
}
exports.reconcileContract = reconcileContract;
//# sourceMappingURL=reconcile-contract.js.map