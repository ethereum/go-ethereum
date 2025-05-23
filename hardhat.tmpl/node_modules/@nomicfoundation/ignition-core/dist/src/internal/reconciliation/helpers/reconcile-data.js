"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileData = void 0;
const assertions_1 = require("../../utils/assertions");
const find_result_for_future_by_id_1 = require("../../views/find-result-for-future-by-id");
const compare_1 = require("./compare");
function reconcileData(future, exState, context) {
    if (typeof future.data === "string" || future.data === undefined) {
        return (0, compare_1.compare)(future, "Data", exState.data, future.data ?? "0x");
    }
    const newData = (0, find_result_for_future_by_id_1.findResultForFutureById)(context.deploymentState, future.data.id);
    (0, assertions_1.assertIgnitionInvariant)(typeof newData === "string", "Expected data to be a string");
    return (0, compare_1.compare)(future, "Data", exState.data, newData);
}
exports.reconcileData = reconcileData;
//# sourceMappingURL=reconcile-data.js.map