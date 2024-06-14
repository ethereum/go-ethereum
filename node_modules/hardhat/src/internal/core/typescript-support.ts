import { HardhatConfig } from "../../types";

import { resolveConfigPath } from "./config/config-loading";
import { HardhatError } from "./errors";
import { ERRORS } from "./errors-list";
import { isRunningHardhatCoreTests } from "./execution-mode";

let cachedIsTypescriptSupported: boolean | undefined;

/**
 * Returns true if Hardhat will run in using typescript mode.
 * @param configPath The config path if provider by the user.
 */
export function willRunWithTypescript(configPath?: string): boolean {
  const config = resolveConfigPath(configPath);
  return isNonEsmTypescriptFile(config);
}

/**
 * Returns true if an Hardhat is already running with typescript.
 */
export function isRunningWithTypescript(config: HardhatConfig): boolean {
  return isNonEsmTypescriptFile(config.paths.configFile);
}

export function isTypescriptSupported() {
  if (cachedIsTypescriptSupported === undefined) {
    try {
      // We resolve these from Hardhat's installation.
      require.resolve("typescript");
      require.resolve("ts-node");
      cachedIsTypescriptSupported = true;
    } catch {
      cachedIsTypescriptSupported = false;
    }
  }

  return cachedIsTypescriptSupported;
}

export function loadTsNode(
  tsConfigPath?: string,
  shouldTypecheck: boolean = false
) {
  try {
    require.resolve("typescript");
  } catch {
    throw new HardhatError(ERRORS.GENERAL.TYPESCRIPT_NOT_INSTALLED);
  }

  try {
    require.resolve("ts-node");
  } catch {
    throw new HardhatError(ERRORS.GENERAL.TS_NODE_NOT_INSTALLED);
  }

  // If we are running tests we just want to transpile
  if (isRunningHardhatCoreTests()) {
    // eslint-disable-next-line import/no-extraneous-dependencies
    require("ts-node/register/transpile-only");
    return;
  }

  if (tsConfigPath !== undefined) {
    process.env.TS_NODE_PROJECT = tsConfigPath;
  }

  // See: https://github.com/nomiclabs/hardhat/issues/265
  if (process.env.TS_NODE_FILES === undefined) {
    process.env.TS_NODE_FILES = "true";
  }

  let tsNodeRequirement = "ts-node/register";

  if (!shouldTypecheck) {
    tsNodeRequirement += "/transpile-only";
  }

  // eslint-disable-next-line import/no-extraneous-dependencies
  require(tsNodeRequirement);
}

function isNonEsmTypescriptFile(path: string): boolean {
  return /\.(ts|cts)$/i.test(path);
}

export function isTypescriptFile(path: string): boolean {
  return /\.(ts|cts|mts)$/i.test(path);
}

export function isJavascriptFile(path: string): boolean {
  return /\.(js|cjs|mjs)$/i.test(path);
}
