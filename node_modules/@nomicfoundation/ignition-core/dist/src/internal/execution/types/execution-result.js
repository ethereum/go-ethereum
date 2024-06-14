"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ExecutionResultType = void 0;
/**
 * The differnt types of result that executing a future can produce.
 */
var ExecutionResultType;
(function (ExecutionResultType) {
    ExecutionResultType["SUCCESS"] = "SUCCESS";
    ExecutionResultType["SIMULATION_ERROR"] = "SIMULATION_ERROR";
    ExecutionResultType["STRATEGY_SIMULATION_ERROR"] = "STRATEGY_SIMULATION_ERROR";
    ExecutionResultType["REVERTED_TRANSACTION"] = "REVERTED_TRANSACTION";
    ExecutionResultType["STATIC_CALL_ERROR"] = "STATIC_CALL_ERROR";
    ExecutionResultType["STRATEGY_ERROR"] = "STRATEGY_ERROR";
    ExecutionResultType["STRATEGY_HELD"] = "STRATEGY_HELD";
})(ExecutionResultType || (exports.ExecutionResultType = ExecutionResultType = {}));
//# sourceMappingURL=execution-result.js.map