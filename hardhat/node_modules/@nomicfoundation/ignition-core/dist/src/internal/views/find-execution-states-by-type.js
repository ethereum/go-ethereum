"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.findExecutionStatesByType = void 0;
function findExecutionStatesByType(exStateType, deployment) {
    const exStates = Object.values(deployment.executionStates).filter((exs) => exs.type === exStateType);
    return exStates;
}
exports.findExecutionStatesByType = findExecutionStatesByType;
//# sourceMappingURL=find-execution-states-by-type.js.map