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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getVarsFilePath = exports.writePromptedForHHVSCode = exports.hasPromptedForHHVSCode = exports.writeTelemetryConsent = exports.hasConsentedTelemetry = exports.getCompilersDir = exports.writeAnalyticsId = exports.readSecondLegacyAnalyticsId = exports.readFirstLegacyAnalyticsId = exports.readAnalyticsId = exports.getCacheDir = void 0;
const debug_1 = __importDefault(require("debug"));
const fs_extra_1 = __importDefault(require("fs-extra"));
const os_1 = __importDefault(require("os"));
const path_1 = __importDefault(require("path"));
const log = (0, debug_1.default)("hardhat:core:global-dir");
async function generatePaths(packageName = "hardhat") {
    const { default: envPaths } = await Promise.resolve().then(() => __importStar(require("env-paths")));
    return envPaths(packageName);
}
function generatePathsSync(packageName = "hardhat") {
    const envPaths = require("env-paths");
    return envPaths(packageName);
}
function getConfigDirSync() {
    const { config } = generatePathsSync();
    fs_extra_1.default.ensureDirSync(config);
    return config;
}
async function getDataDir(packageName) {
    const { data } = await generatePaths(packageName);
    await fs_extra_1.default.ensureDir(data);
    return data;
}
async function getCacheDir() {
    const { cache } = await generatePaths();
    await fs_extra_1.default.ensureDir(cache);
    return cache;
}
exports.getCacheDir = getCacheDir;
async function readAnalyticsId() {
    const globalDataDir = await getDataDir();
    const idFile = path_1.default.join(globalDataDir, "analytics.json");
    return readId(idFile);
}
exports.readAnalyticsId = readAnalyticsId;
/**
 * This is the first way that the analytics id was saved.
 */
function readFirstLegacyAnalyticsId() {
    const oldIdFile = path_1.default.join(os_1.default.homedir(), ".buidler", "config.json");
    return readId(oldIdFile);
}
exports.readFirstLegacyAnalyticsId = readFirstLegacyAnalyticsId;
/**
 * This is the same way the analytics id is saved now, but using buidler as the
 * name of the project for env-paths
 */
async function readSecondLegacyAnalyticsId() {
    const globalDataDir = await getDataDir("buidler");
    const idFile = path_1.default.join(globalDataDir, "analytics.json");
    return readId(idFile);
}
exports.readSecondLegacyAnalyticsId = readSecondLegacyAnalyticsId;
async function readId(idFile) {
    log(`Looking up Client Id at ${idFile}`);
    let clientId;
    try {
        const data = await fs_extra_1.default.readJSON(idFile, { encoding: "utf8" });
        clientId = data.analytics.clientId;
    }
    catch (error) {
        return undefined;
    }
    log(`Client Id found: ${clientId}`);
    return clientId;
}
async function writeAnalyticsId(clientId) {
    const globalDataDir = await getDataDir();
    const idFile = path_1.default.join(globalDataDir, "analytics.json");
    await fs_extra_1.default.writeJSON(idFile, {
        analytics: {
            clientId,
        },
    }, { encoding: "utf-8", spaces: 2 });
    log(`Stored clientId ${clientId}`);
}
exports.writeAnalyticsId = writeAnalyticsId;
async function getCompilersDir() {
    const cache = await getCacheDir();
    // Note: we introduce `-v2` to invalidate all the previous compilers at once
    const compilersCache = path_1.default.join(cache, "compilers-v2");
    await fs_extra_1.default.ensureDir(compilersCache);
    return compilersCache;
}
exports.getCompilersDir = getCompilersDir;
/**
 * Checks if the user has given (or refused) consent for telemetry.
 *
 * Returns undefined if it can't be determined.
 */
function hasConsentedTelemetry() {
    const configDir = getConfigDirSync();
    const telemetryConsentPath = path_1.default.join(configDir, "telemetry-consent.json");
    const fileExists = fs_extra_1.default.pathExistsSync(telemetryConsentPath);
    if (!fileExists) {
        return undefined;
    }
    const { consent } = fs_extra_1.default.readJSONSync(telemetryConsentPath);
    return consent;
}
exports.hasConsentedTelemetry = hasConsentedTelemetry;
function writeTelemetryConsent(consent) {
    const configDir = getConfigDirSync();
    const telemetryConsentPath = path_1.default.join(configDir, "telemetry-consent.json");
    fs_extra_1.default.writeJSONSync(telemetryConsentPath, { consent }, { spaces: 2 });
}
exports.writeTelemetryConsent = writeTelemetryConsent;
/**
 * Checks if we have already prompted the user to install the Hardhat for VSCode extension.
 */
function hasPromptedForHHVSCode() {
    const configDir = getConfigDirSync();
    const extensionPromptedPath = path_1.default.join(configDir, "extension-prompt.json");
    const fileExists = fs_extra_1.default.pathExistsSync(extensionPromptedPath);
    return fileExists;
}
exports.hasPromptedForHHVSCode = hasPromptedForHHVSCode;
function writePromptedForHHVSCode() {
    const configDir = getConfigDirSync();
    const extensionPromptedPath = path_1.default.join(configDir, "extension-prompt.json");
    fs_extra_1.default.writeFileSync(extensionPromptedPath, "{}");
}
exports.writePromptedForHHVSCode = writePromptedForHHVSCode;
function getVarsFilePath() {
    return path_1.default.join(getConfigDirSync(), "vars.json");
}
exports.getVarsFilePath = getVarsFilePath;
//# sourceMappingURL=global-dir.js.map