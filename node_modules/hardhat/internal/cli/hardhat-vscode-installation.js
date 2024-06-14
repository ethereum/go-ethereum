"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.installHardhatVSCode = exports.isHardhatVSCodeInstalled = exports.InstallationState = void 0;
const child_process_1 = require("child_process");
var InstallationState;
(function (InstallationState) {
    InstallationState[InstallationState["VSCODE_FAILED_OR_NOT_INSTALLED"] = 0] = "VSCODE_FAILED_OR_NOT_INSTALLED";
    InstallationState[InstallationState["EXTENSION_INSTALLED"] = 1] = "EXTENSION_INSTALLED";
    InstallationState[InstallationState["EXTENSION_NOT_INSTALLED"] = 2] = "EXTENSION_NOT_INSTALLED";
})(InstallationState = exports.InstallationState || (exports.InstallationState = {}));
const HARDHAT_VSCODE_ID = "NomicFoundation.hardhat-solidity";
function isHardhatVSCodeInstalled() {
    try {
        const { stdout, status } = (0, child_process_1.spawnSync)("code", ["--list-extensions"], {
            encoding: "utf8",
        });
        if (status !== 0) {
            return InstallationState.VSCODE_FAILED_OR_NOT_INSTALLED;
        }
        return stdout.includes(HARDHAT_VSCODE_ID)
            ? InstallationState.EXTENSION_INSTALLED
            : InstallationState.EXTENSION_NOT_INSTALLED;
    }
    catch (e) {
        return InstallationState.VSCODE_FAILED_OR_NOT_INSTALLED;
    }
}
exports.isHardhatVSCodeInstalled = isHardhatVSCodeInstalled;
function installHardhatVSCode() {
    try {
        const { status } = (0, child_process_1.spawnSync)("code", ["--install-extension", HARDHAT_VSCODE_ID], {
            encoding: "utf8",
        });
        return status === 0;
    }
    catch (e) {
        return false;
    }
}
exports.installHardhatVSCode = installHardhatVSCode;
//# sourceMappingURL=hardhat-vscode-installation.js.map