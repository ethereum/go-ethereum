"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.findExecutionStateById = void 0;
const assertions_1 = require("../utils/assertions");
function findExecutionStateById(exStateType, deployment, futureId) {
    const exState = deployment.executionStates[futureId];
    (0, assertions_1.assertIgnitionInvariant)(exState !== undefined, `Expected execution state for ${futureId} to exist, but it did not`);
    (0, assertions_1.assertIgnitionInvariant)(exState.type === exStateType, `Expected execution state for ${futureId} to be a ${exStateType}, but instead it was ${exState.type}`);
    return exState;
}
exports.findExecutionStateById = findExecutionStateById;
//# sourceMappingURL=find-execution-state-by-id.js.map