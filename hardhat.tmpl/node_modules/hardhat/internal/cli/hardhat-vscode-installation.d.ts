export declare enum InstallationState {
    VSCODE_FAILED_OR_NOT_INSTALLED = 0,
    EXTENSION_INSTALLED = 1,
    EXTENSION_NOT_INSTALLED = 2
}
export declare function isHardhatVSCodeInstalled(): InstallationState;
export declare function installHardhatVSCode(): boolean;
//# sourceMappingURL=hardhat-vscode-installation.d.ts.map