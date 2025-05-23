"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.decodeSimulationResult = void 0;
const assertions_1 = require("../../../utils/assertions");
const execution_result_1 = require("../../types/execution-result");
const execution_strategy_1 = require("../../types/execution-strategy");
function decodeSimulationResult(strategyGenerator, exState) {
    return async (simulationResult) => {
        const response = await strategyGenerator.next({
            type: execution_strategy_1.OnchainInteractionResponseType.SIMULATION_RESULT,
            result: simulationResult,
        });
        (0, assertions_1.assertIgnitionInvariant)(response.value.type === execution_strategy_1.SIMULATION_SUCCESS_SIGNAL_TYPE ||
            response.value.type === execution_result_1.ExecutionResultType.STRATEGY_SIMULATION_ERROR ||
            response.value.type === execution_result_1.ExecutionResultType.SIMULATION_ERROR, `Invalid response received from strategy after a simulation was run before sending a transaction for ExecutionState ${exState.id}`);
        if (response.value.type === execution_strategy_1.SIMULATION_SUCCESS_SIGNAL_TYPE) {
            return undefined;
        }
        return response.value;
    };
}
exports.decodeSimulationResult = decodeSimulationResult;
//# sourceMappingURL=decode-simulation-result.js.map