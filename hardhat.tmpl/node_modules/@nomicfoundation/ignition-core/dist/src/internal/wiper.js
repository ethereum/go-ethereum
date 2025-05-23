"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Wiper = void 0;
const errors_1 = require("../errors");
const errors_list_1 = require("./errors-list");
const deployment_state_helpers_1 = require("./execution/deployment-state-helpers");
const messages_1 = require("./execution/types/messages");
class Wiper {
    _deploymentLoader;
    constructor(_deploymentLoader) {
        this._deploymentLoader = _deploymentLoader;
    }
    async wipe(futureId) {
        const deploymentState = await (0, deployment_state_helpers_1.loadDeploymentState)(this._deploymentLoader);
        if (deploymentState === undefined) {
            throw new errors_1.IgnitionError(errors_list_1.ERRORS.WIPE.UNINITIALIZED_DEPLOYMENT, {
                futureId,
            });
        }
        const executionState = deploymentState.executionStates[futureId];
        if (executionState === undefined) {
            throw new errors_1.IgnitionError(errors_list_1.ERRORS.WIPE.NO_STATE_FOR_FUTURE, { futureId });
        }
        const dependents = Object.values(deploymentState.executionStates).filter((psm) => psm.dependencies.has(futureId));
        if (dependents.length > 0) {
            throw new errors_1.IgnitionError(errors_list_1.ERRORS.WIPE.DEPENDENT_FUTURES, {
                futureId,
                dependents: dependents.map((d) => d.id).join(", "),
            });
        }
        const wipeMessage = {
            type: messages_1.JournalMessageType.WIPE_APPLY,
            futureId,
        };
        return (0, deployment_state_helpers_1.applyNewMessage)(wipeMessage, deploymentState, this._deploymentLoader);
    }
}
exports.Wiper = Wiper;
//# sourceMappingURL=wiper.js.map