"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DeploymentResultType = void 0;
/**
 * The different kinds of results that a deployment can produce.
 *
 * @beta
 */
var DeploymentResultType;
(function (DeploymentResultType) {
    /**
     * One or more futures failed validation.
     */
    DeploymentResultType["VALIDATION_ERROR"] = "VALIDATION_ERROR";
    /**
     * One or more futures failed the reconciliation process with
     * the previous state of the deployment.
     */
    DeploymentResultType["RECONCILIATION_ERROR"] = "RECONCILIATION_ERROR";
    /**
     * One or more future's execution failed or timed out.
     */
    DeploymentResultType["EXECUTION_ERROR"] = "EXECUTION_ERROR";
    /**
     * One or more futures from a previous run failed or timed out.
     */
    DeploymentResultType["PREVIOUS_RUN_ERROR"] = "PREVIOUS_RUN_ERROR";
    /**
     * The entire deployment was successful.
     */
    DeploymentResultType["SUCCESSFUL_DEPLOYMENT"] = "SUCCESSFUL_DEPLOYMENT";
})(DeploymentResultType || (exports.DeploymentResultType = DeploymentResultType = {}));
//# sourceMappingURL=deploy.js.map