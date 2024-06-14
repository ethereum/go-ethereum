"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileLibraries = void 0;
const future_resolvers_1 = require("../../execution/future-processor/helpers/future-resolvers");
const utils_1 = require("../utils");
function reconcileLibraries(future, exState, context) {
    const futureLibraries = (0, future_resolvers_1.resolveLibraries)(future.libraries, context.deploymentState);
    for (const [libName, exStateLib] of Object.entries(exState.libraries)) {
        if (futureLibraries[libName] === undefined) {
            return (0, utils_1.fail)(future, `Library ${libName} has been removed`);
        }
        if (futureLibraries[libName] !== exStateLib) {
            return (0, utils_1.fail)(future, `Library ${libName}'s address has been changed`);
        }
    }
    for (const libName of Object.keys(futureLibraries)) {
        if (exState.libraries[libName] === undefined) {
            return (0, utils_1.fail)(future, `Library ${libName} has been added`);
        }
    }
}
exports.reconcileLibraries = reconcileLibraries;
//# sourceMappingURL=reconcile-libraries.js.map