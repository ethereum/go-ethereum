"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.calculateDeploymentCompleteDisplay = void 0;
const ignition_core_1 = require("@nomicfoundation/ignition-core");
const chalk_1 = __importDefault(require("chalk"));
const cwd_relative_path_1 = require("./cwd-relative-path");
const was_anything_executed_1 = require("./was-anything-executed");
function calculateDeploymentCompleteDisplay(event, uiState) {
    const moduleName = uiState.moduleName ?? "unknown";
    const isResumed = uiState.isResumed ?? false;
    switch (event.result.type) {
        case ignition_core_1.DeploymentResultType.SUCCESSFUL_DEPLOYMENT: {
            const isRerunWithNoChanges = isResumed && !(0, was_anything_executed_1.wasAnythingExecuted)(uiState);
            return _displaySuccessfulDeployment(event.result, {
                moduleName,
                isRerunWithNoChanges,
                deploymentDir: uiState.deploymentDir,
            });
        }
        case ignition_core_1.DeploymentResultType.VALIDATION_ERROR: {
            return _displayValidationErrors(event.result, { moduleName });
        }
        case ignition_core_1.DeploymentResultType.RECONCILIATION_ERROR: {
            return _displayReconciliationErrors(event.result, { moduleName });
        }
        case ignition_core_1.DeploymentResultType.PREVIOUS_RUN_ERROR: {
            return _displayPreviousRunErrors(event.result, { moduleName });
        }
        case ignition_core_1.DeploymentResultType.EXECUTION_ERROR: {
            return _displayExecutionErrors(event.result, { moduleName });
        }
    }
}
exports.calculateDeploymentCompleteDisplay = calculateDeploymentCompleteDisplay;
function _displaySuccessfulDeployment(result, { moduleName, isRerunWithNoChanges, deploymentDir, }) {
    const fillerText = isRerunWithNoChanges
        ? `Nothing new to deploy based on previous execution stored in ${(0, cwd_relative_path_1.pathFromCwd)(deploymentDir ?? "Not provided")}`
        : `successfully deployed 🚀`;
    let text = `[ ${moduleName} ] ${fillerText}

${chalk_1.default.bold("Deployed Addresses")}

`;
    const deployedContracts = Object.values(result.contracts);
    if (deployedContracts.length > 0) {
        text += deployedContracts
            .map((contract) => `${contract.id} - ${contract.address}`)
            .join("\n");
    }
    else {
        text += `${chalk_1.default.italic("No contracts were deployed")}`;
    }
    return text;
}
function _displayValidationErrors(result, { moduleName }) {
    let text = `[ ${moduleName} ] validation failed ⛔

The module contains futures that would fail to execute:

`;
    text += Object.entries(result.errors)
        .map(([futureId, errors]) => {
        let futureSection = `${futureId}:\n`;
        futureSection += errors.map((error) => ` - ${error}`).join("\n");
        return futureSection;
    })
        .join("\n\n");
    text += `\n\nUpdate the invalid futures and rerun the deployment.`;
    return text;
}
function _displayReconciliationErrors(result, { moduleName }) {
    let text = `[ ${moduleName} ] reconciliation failed ⛔

The module contains changes to executed futures:

`;
    text += Object.entries(result.errors)
        .map(([futureId, errors]) => {
        let errorSection = `${futureId}:\n`;
        errorSection += errors.map((error) => ` - ${error}`).join("\n");
        return errorSection;
    })
        .join("\n\n");
    text += `\n\nConsider modifying your module to remove the inconsistencies with deployed futures.`;
    return text;
}
function _displayPreviousRunErrors(result, { moduleName }) {
    let text = `[ ${moduleName} ] deployment cancelled ⛔\n\n`;
    text += `These futures failed or timed out on a previous run:\n`;
    text += Object.keys(result.errors)
        .map((futureId) => ` - ${futureId}`)
        .join("\n");
    text += `\n\nUse the ${chalk_1.default.italic("wipe")} task to reset them.`;
    return text;
}
function _displayExecutionErrors(result, { moduleName }) {
    const sections = [];
    let text = `[ ${moduleName} ] failed ⛔\n\n`;
    if (result.timedOut.length > 0) {
        let timedOutSection = `Futures timed out with transactions unconfirmed after maximum fee bumps:\n`;
        timedOutSection += Object.values(result.timedOut)
            .map(({ futureId }) => ` - ${futureId}`)
            .join("\n");
        sections.push(timedOutSection);
    }
    if (result.failed.length > 0) {
        let failedSection = `Futures failed during execution:\n`;
        failedSection += Object.values(result.failed)
            .map(({ futureId, error }) => ` - ${futureId}: ${error}`)
            .join("\n");
        failedSection +=
            "\n\nTo learn how to handle these errors: https://hardhat.org/ignition-errors";
        sections.push(failedSection);
    }
    if (result.held.length > 0) {
        let heldSection = `Held:\n`;
        heldSection += Object.values(result.held)
            .map(({ futureId, reason }) => ` - ${futureId}: ${reason}`)
            .join("\n");
        sections.push(heldSection);
    }
    text += sections.join("\n\n");
    return text;
}
//# sourceMappingURL=calculate-deployment-complete-display.js.map