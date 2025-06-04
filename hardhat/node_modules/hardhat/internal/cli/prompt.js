"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.confirmHHVSCodeInstallation = exports.confirmTelemetryConsent = exports.confirmProjectCreation = exports.confirmRecommendedDepsInstallation = void 0;
function createConfirmationPrompt(name, message) {
    return {
        type: "confirm",
        name,
        message,
        initial: "y",
        default: "(Y/n)",
        isTrue(input) {
            if (typeof input === "string") {
                return input.toLowerCase() === "y";
            }
            return input;
        },
        isFalse(input) {
            if (typeof input === "string") {
                return input.toLowerCase() === "n";
            }
            return input;
        },
        format() {
            const that = this;
            const value = that.value === true ? "y" : "n";
            if (that.state.submitted === true) {
                return that.styles.submitted(value);
            }
            return value;
        },
    };
}
async function confirmRecommendedDepsInstallation(depsToInstall, packageManager) {
    const { default: enquirer } = await Promise.resolve().then(() => __importStar(require("enquirer")));
    let responses;
    try {
        responses = await enquirer.prompt([
            createConfirmationPrompt("shouldInstallPlugin", `Do you want to install this sample project's dependencies with ${packageManager} (${Object.keys(depsToInstall).join(" ")})?`),
        ]);
    }
    catch (e) {
        if (e === "") {
            return false;
        }
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw e;
    }
    return responses.shouldInstallPlugin;
}
exports.confirmRecommendedDepsInstallation = confirmRecommendedDepsInstallation;
async function confirmProjectCreation() {
    const enquirer = require("enquirer");
    return enquirer.prompt([
        {
            name: "projectRoot",
            type: "input",
            initial: process.cwd(),
            message: "Hardhat project root:",
        },
        createConfirmationPrompt("shouldAddGitIgnore", "Do you want to add a .gitignore?"),
    ]);
}
exports.confirmProjectCreation = confirmProjectCreation;
async function confirmTelemetryConsent() {
    return confirmationPromptWithTimeout("telemetryConsent", "Help us improve Hardhat with anonymous crash reports & basic usage data?");
}
exports.confirmTelemetryConsent = confirmTelemetryConsent;
/**
 * true = install ext
 * false = don't install and don't ask again
 * undefined = we couldn't confirm if the extension is installed or not
 */
async function confirmHHVSCodeInstallation() {
    return confirmationPromptWithTimeout("shouldInstallExtension", "Would you like to install the Hardhat for Visual Studio Code extension? It adds advanced editing assistance for Solidity to VSCode");
}
exports.confirmHHVSCodeInstallation = confirmHHVSCodeInstallation;
async function confirmationPromptWithTimeout(name, message, timeoutMilliseconds = 10000) {
    try {
        const enquirer = require("enquirer");
        const prompt = new enquirer.prompts.Confirm(createConfirmationPrompt(name, message));
        let timeout;
        const timeoutPromise = new Promise((resolve) => {
            timeout = setTimeout(resolve, timeoutMilliseconds);
        });
        const result = await Promise.race([prompt.run(), timeoutPromise]);
        clearTimeout(timeout);
        if (result === undefined) {
            await prompt.cancel();
        }
        return result;
    }
    catch (e) {
        if (e === "") {
            return undefined;
        }
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw e;
    }
}
//# sourceMappingURL=prompt.js.map