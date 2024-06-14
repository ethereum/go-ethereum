"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileArguments = void 0;
const type_guards_1 = require("../../../type-guards");
const future_resolvers_1 = require("../../execution/future-processor/helpers/future-resolvers");
const execution_state_1 = require("../../execution/types/execution-state");
const address_1 = require("../../execution/utils/address");
const utils_1 = require("../utils");
function reconcileArguments(future, exState, context) {
    const unresolvedFutureArgs = (0, type_guards_1.isDeploymentFuture)(future)
        ? future.constructorArgs
        : future.args;
    const futureArgs = (0, future_resolvers_1.resolveArgs)(unresolvedFutureArgs, context.deploymentState, context.deploymentParameters, context.accounts);
    const exStateArgs = exState.type === execution_state_1.ExecutionSateType.DEPLOYMENT_EXECUTION_STATE
        ? exState.constructorArgs
        : exState.args;
    if (futureArgs.length !== exStateArgs.length) {
        return (0, utils_1.fail)(future, `The number of arguments changed from ${exStateArgs.length} to ${futureArgs.length}`);
    }
    const isEqual = require("lodash/isEqual");
    for (const [i, futureArg] of futureArgs.entries()) {
        const exStateArg = exStateArgs[i];
        // if both args are addresses, we need to compare the checksummed versions
        // to ensure case discrepancies are ignored
        if ((0, address_1.isAddress)(futureArg) && (0, address_1.isAddress)(exStateArg)) {
            if (!(0, address_1.equalAddresses)(futureArg, exStateArg)) {
                return (0, utils_1.fail)(future, `Argument at index ${i} has been changed`);
            }
        }
        else if (!isEqual(futureArg, exStateArg)) {
            return (0, utils_1.fail)(future, `Argument at index ${i} has been changed`);
        }
    }
}
exports.reconcileArguments = reconcileArguments;
//# sourceMappingURL=reconcile-arguments.js.map