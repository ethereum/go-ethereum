"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileAddress = void 0;
const future_resolvers_1 = require("../../execution/future-processor/helpers/future-resolvers");
const compare_1 = require("./compare");
function reconcileAddress(future, exState, context) {
    const resolvedAddress = (0, future_resolvers_1.resolveAddressLike)(future.address, context.deploymentState, context.deploymentParameters, context.accounts);
    return (0, compare_1.compare)(future, "Address", exState.contractAddress, resolvedAddress);
}
exports.reconcileAddress = reconcileAddress;
//# sourceMappingURL=reconcile-address.js.map