"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.errorDeploymentResultToExceptionMessage = void 0;
const ignition_core_1 = require("@nomicfoundation/ignition-core");
const plugins_1 = require("hardhat/plugins");
/**
 * Converts the result of an errored deployment into a message that can
 * be shown to the user in an exception.
 *
 * @param result - the errored deployment's result
 * @returns the text of the message
 */
function errorDeploymentResultToExceptionMessage(result) {
    switch (result.type) {
        case ignition_core_1.DeploymentResultType.VALIDATION_ERROR:
            return _convertValidationError(result);
        case ignition_core_1.DeploymentResultType.RECONCILIATION_ERROR:
            return _convertReconciliationError(result);
        case ignition_core_1.DeploymentResultType.EXECUTION_ERROR:
            return _convertExecutionError(result);
        case ignition_core_1.DeploymentResultType.PREVIOUS_RUN_ERROR:
            return _convertPreviousRunError(result);
    }
}
exports.errorDeploymentResultToExceptionMessage = errorDeploymentResultToExceptionMessage;
function _convertValidationError(result) {
    const errorsList = Object.entries(result.errors).flatMap(([futureId, errors]) => errors.map((err) => `  * ${futureId}: ${err}`));
    return `The deployment wasn't run because of the following validation errors:

${errorsList.join("\n")}`;
}
function _convertReconciliationError(result) {
    const errorsList = Object.entries(result.errors).flatMap(([futureId, errors]) => errors.map((err) => `  * ${futureId}: ${err}`));
    return `The deployment wasn't run because of the following reconciliation errors:

${errorsList.join("\n")}`;
}
function _convertExecutionError(result) {
    const sections = [];
    const messageDetails = {
        timeouts: result.timedOut.length > 0,
        failures: result.failed.length > 0,
        held: result.held.length > 0,
    };
    if (messageDetails.timeouts) {
        const timeoutList = result.timedOut.map(({ futureId, networkInteractionId }) => `  * ${futureId}/${networkInteractionId}`);
        sections.push(`Timed out:\n\n${timeoutList.join("\n")}`);
    }
    if (messageDetails.failures) {
        const errorList = result.failed.map(({ futureId, networkInteractionId, error }) => `  * ${futureId}/${networkInteractionId}: ${error}`);
        sections.push(`Failures:\n\n${errorList.join("\n")}`);
    }
    if (messageDetails.held) {
        const reasonList = result.held.map(({ futureId, heldId, reason }) => `  * ${futureId}/${heldId}: ${reason}`);
        sections.push(`Held:\n\n${reasonList.join("\n")}`);
    }
    return `The deployment wasn't successful, there were ${_toText(messageDetails)}:

${sections.join("\n\n")}`;
}
function _convertPreviousRunError(result) {
    const errorsList = Object.entries(result.errors).flatMap(([futureId, errors]) => errors.map((err) => `  * ${futureId}: ${err}`));
    return `The deployment wasn't run because of the following errors in a previous run:

${errorsList.join("\n")}`;
}
function _toText({ timeouts, failures, held, }) {
    if (timeouts && failures && held) {
        return "timeouts, failures and holds";
    }
    else if (timeouts && failures) {
        return "timeouts and failures";
    }
    else if (failures && held) {
        return "failures and holds";
    }
    else if (timeouts && held) {
        return "timeouts and holds";
    }
    else if (timeouts) {
        return "timeouts";
    }
    else if (failures) {
        return "failures";
    }
    else if (held) {
        return "holds";
    }
    throw new plugins_1.HardhatPluginError("@nomicfoundation/hardhat-ignition", "Invariant violated: neither timeouts or failures");
}
//# sourceMappingURL=error-deployment-result-to-exception-message.js.map