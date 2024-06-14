"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.batches = void 0;
const batcher_1 = require("./internal/batcher");
const deployment_state_reducer_1 = require("./internal/execution/reducers/deployment-state-reducer");
/**
 * Provides a array of batches, where each batch is an array of futureIds,
 * based on Ignition's batching algorithm, assuming a the module is being
 * run from as a fresh deployment.
 *
 * @param ignitionModule - the Ignition module to be get batch information for
 * @returns the batches Ignition will use for the module
 *
 * @beta
 */
function batches(ignitionModule) {
    const deploymentState = (0, deployment_state_reducer_1.deploymentStateReducer)(undefined);
    return batcher_1.Batcher.batch(ignitionModule, deploymentState);
}
exports.batches = batches;
//# sourceMappingURL=batches.js.map