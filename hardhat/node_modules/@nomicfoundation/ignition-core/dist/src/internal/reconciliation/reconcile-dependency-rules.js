"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileDependencyRules = void 0;
const difference_1 = __importDefault(require("lodash/difference"));
const execution_state_1 = require("../execution/types/execution-state");
const utils_1 = require("./utils");
function reconcileDependencyRules(future, executionState, context) {
    const previousDeps = [...executionState.dependencies];
    const currentDeps = [...future.dependencies].map((f) => f.id);
    const additionalDeps = (0, difference_1.default)(currentDeps, previousDeps);
    for (const additionalDep of additionalDeps) {
        const additionalExecutionState = context.deploymentState.executionStates[additionalDep];
        if (additionalExecutionState === undefined) {
            return (0, utils_1.fail)(future, `A dependency from ${future.id} to ${additionalDep} has been added. The former has started executing before the latter started executing, so this change is incompatible.`);
        }
        // TODO: Check that is was successfully executed before `executionState` was created.
        if (additionalExecutionState.status === execution_state_1.ExecutionStatus.SUCCESS) {
            continue;
        }
        return (0, utils_1.fail)(future, `A dependency from ${future.id} to ${additionalDep} has been added, and both futures had already started executing, so this change is incompatible`);
    }
    return { success: true };
}
exports.reconcileDependencyRules = reconcileDependencyRules;
//# sourceMappingURL=reconcile-dependency-rules.js.map