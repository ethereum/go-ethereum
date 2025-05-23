import { spawnSync } from "child_process";

export enum InstallationState {
  VSCODE_FAILED_OR_NOT_INSTALLED,
  EXTENSION_INSTALLED,
  EXTENSION_NOT_INSTALLED,
}

const HARDHAT_VSCODE_ID = "NomicFoundation.hardhat-solidity";

export function isHardhatVSCodeInstalled(): InstallationState {
  try {
    const { stdout, status } = spawnSync("code", ["--list-extensions"], {
      encoding: "utf8",
    });

    if (status !== 0) {
      return InstallationState.VSCODE_FAILED_OR_NOT_INSTALLED;
    }

    return stdout.includes(HARDHAT_VSCODE_ID)
      ? InstallationState.EXTENSION_INSTALLED
      : InstallationState.EXTENSION_NOT_INSTALLED;
  } catch (e) {
    return InstallationState.VSCODE_FAILED_OR_NOT_INSTALLED;
  }
}

export function installHardhatVSCode(): boolean {
  try {
    const { status } = spawnSync(
      "code",
      ["--install-extension", HARDHAT_VSCODE_ID],
      {
        encoding: "utf8",
      }
    );

    return status === 0;
  } catch (e) {
    return false;
  }
}
