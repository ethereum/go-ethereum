import type envPathsT from "env-paths";

import debug from "debug";
import fs from "fs-extra";
import os from "os";
import path from "path";

const log = debug("hardhat:core:global-dir");

async function generatePaths(packageName = "hardhat") {
  const { default: envPaths } = await import("env-paths");
  return envPaths(packageName);
}

function generatePathsSync(packageName = "hardhat") {
  const envPaths: typeof envPathsT = require("env-paths");
  return envPaths(packageName);
}

function getConfigDirSync(): string {
  const { config } = generatePathsSync();
  fs.ensureDirSync(config);
  return config;
}

async function getDataDir(packageName?: string): Promise<string> {
  const { data } = await generatePaths(packageName);
  await fs.ensureDir(data);
  return data;
}

export async function getCacheDir(): Promise<string> {
  const { cache } = await generatePaths();
  await fs.ensureDir(cache);
  return cache;
}

export async function readAnalyticsId() {
  const globalDataDir = await getDataDir();
  const idFile = path.join(globalDataDir, "analytics.json");
  return readId(idFile);
}

/**
 * This is the first way that the analytics id was saved.
 */
export function readFirstLegacyAnalyticsId() {
  const oldIdFile = path.join(os.homedir(), ".buidler", "config.json");
  return readId(oldIdFile);
}

/**
 * This is the same way the analytics id is saved now, but using buidler as the
 * name of the project for env-paths
 */
export async function readSecondLegacyAnalyticsId() {
  const globalDataDir = await getDataDir("buidler");
  const idFile = path.join(globalDataDir, "analytics.json");
  return readId(idFile);
}

async function readId(idFile: string): Promise<string | undefined> {
  log(`Looking up Client Id at ${idFile}`);
  let clientId: string;
  try {
    const data = await fs.readJSON(idFile, { encoding: "utf8" });
    clientId = data.analytics.clientId;
  } catch (error) {
    return undefined;
  }

  log(`Client Id found: ${clientId}`);
  return clientId;
}

export async function writeAnalyticsId(clientId: string) {
  const globalDataDir = await getDataDir();
  const idFile = path.join(globalDataDir, "analytics.json");
  await fs.writeJSON(
    idFile,
    {
      analytics: {
        clientId,
      },
    },
    { encoding: "utf-8", spaces: 2 }
  );
  log(`Stored clientId ${clientId}`);
}

export async function getCompilersDir() {
  const cache = await getCacheDir();
  // Note: we introduce `-v2` to invalidate all the previous compilers at once
  const compilersCache = path.join(cache, "compilers-v2");
  await fs.ensureDir(compilersCache);
  return compilersCache;
}

/**
 * Checks if the user has given (or refused) consent for telemetry.
 *
 * Returns undefined if it can't be determined.
 */
export function hasConsentedTelemetry(): boolean | undefined {
  const configDir = getConfigDirSync();
  const telemetryConsentPath = path.join(configDir, "telemetry-consent.json");

  const fileExists = fs.pathExistsSync(telemetryConsentPath);

  if (!fileExists) {
    return undefined;
  }

  const { consent } = fs.readJSONSync(telemetryConsentPath);
  return consent;
}

export function writeTelemetryConsent(consent: boolean) {
  const configDir = getConfigDirSync();
  const telemetryConsentPath = path.join(configDir, "telemetry-consent.json");

  fs.writeJSONSync(telemetryConsentPath, { consent }, { spaces: 2 });
}

/**
 * Checks if we have already prompted the user to install the Hardhat for VSCode extension.
 */
export function hasPromptedForHHVSCode(): boolean {
  const configDir = getConfigDirSync();
  const extensionPromptedPath = path.join(configDir, "extension-prompt.json");

  const fileExists = fs.pathExistsSync(extensionPromptedPath);

  return fileExists;
}

export function writePromptedForHHVSCode() {
  const configDir = getConfigDirSync();
  const extensionPromptedPath = path.join(configDir, "extension-prompt.json");

  fs.writeFileSync(extensionPromptedPath, "{}");
}

export function getVarsFilePath(): string {
  return path.join(getConfigDirSync(), "vars.json");
}
