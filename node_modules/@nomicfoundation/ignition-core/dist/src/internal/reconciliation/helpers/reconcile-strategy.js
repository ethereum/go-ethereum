"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileStrategy = void 0;
const execution_state_1 = require("../../execution/types/execution-state");
const utils_1 = require("../utils");
function reconcileStrategy(future, exState, context) {
    /**
     * If the execution was successful, we don't need to reconcile the strategy.
     *
     * The strategy is set per run, so reconciling already completed futures
     * would lead to a false positive. We only want to reconcile futures that
     * will be run again.
     */
    if (exState.status === execution_state_1.ExecutionStatus.SUCCESS) {
        return undefined;
    }
    const storedStrategyName = exState.strategy;
    const newStrategyName = context.strategy;
    if (storedStrategyName !== newStrategyName) {
        return (0, utils_1.fail)(future, `Strategy changed from "${storedStrategyName}" to "${newStrategyName}"`);
    }
    // We may have an `undefined` strategy config when reading a journal, as
    // some previous versions of Ignition didn't set this property
    const storedStrategyConfig = exState.strategyConfig ?? {};
    const newStrategyConfig = context.strategyConfig;
    const isEqual = require("lodash/isEqual");
    if (!isEqual(storedStrategyConfig, newStrategyConfig)) {
        return (0, utils_1.fail)(future, `Strategy config changed from ${JSON.stringify(storedStrategyConfig)} to ${JSON.stringify(newStrategyConfig)}`);
    }
}
exports.reconcileStrategy = reconcileStrategy;
//# sourceMappingURL=reconcile-strategy.js.map