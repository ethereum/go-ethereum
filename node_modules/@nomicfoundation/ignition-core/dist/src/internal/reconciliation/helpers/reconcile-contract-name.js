"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileContractName = void 0;
const compare_1 = require("./compare");
function reconcileContractName(future, exState, _context) {
    return (0, compare_1.compare)(future, "Contract name", exState.contractName, future.contractName);
}
exports.reconcileContractName = reconcileContractName;
//# sourceMappingURL=reconcile-contract-name.js.map