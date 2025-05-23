"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.BannerManager = void 0;
const node_path_1 = __importDefault(require("node:path"));
const promises_1 = __importDefault(require("node:fs/promises"));
const debug_1 = __importDefault(require("debug"));
const global_dir_1 = require("../util/global-dir");
const request_1 = require("../util/request");
const log = (0, debug_1.default)("hardhat:util:banner-manager");
const BANNER_CONFIG_URL = "https://raw.githubusercontent.com/NomicFoundation/hardhat/refs/heads/main/banner-config.json";
const BANNER_CACHE_FILE_NAME = "banner-config.json";
class BannerManager {
    constructor(_bannerConfig, _lastDisplayTime, _lastRequestTime) {
        this._bannerConfig = _bannerConfig;
        this._lastDisplayTime = _lastDisplayTime;
        this._lastRequestTime = _lastRequestTime;
    }
    static async getInstance() {
        if (this._instance === undefined) {
            log("Initializing BannerManager");
            const { bannerConfig, lastDisplayTime, lastRequestTime } = await readCache();
            this._instance = new BannerManager(bannerConfig, lastDisplayTime, lastRequestTime);
        }
        return this._instance;
    }
    async _requestBannerConfig(timeout) {
        if (this._bannerConfig !== undefined) {
            const timeSinceLastRequest = Date.now() - this._lastRequestTime;
            if (timeSinceLastRequest <
                this._bannerConfig.minSecondsBetweenRequests * 1000) {
                log(`Skipping banner config request. Time since last request: ${timeSinceLastRequest}ms`);
                return;
            }
        }
        try {
            const bannerConfig = await (0, request_1.requestJson)(BANNER_CONFIG_URL, timeout);
            if (!this._isBannerConfig(bannerConfig)) {
                log(`Invalid banner config received:`, bannerConfig);
                return;
            }
            this._bannerConfig = bannerConfig;
            this._lastRequestTime = Date.now();
            await writeCache({
                bannerConfig: this._bannerConfig,
                lastDisplayTime: this._lastDisplayTime,
                lastRequestTime: this._lastRequestTime,
            });
        }
        catch (error) {
            log(`Error requesting banner config: ${error instanceof Error ? error.message : JSON.stringify(error)}`);
        }
    }
    _isBannerConfig(value) {
        if (typeof value !== "object" || value === null || Array.isArray(value)) {
            return false;
        }
        return (Object.getOwnPropertyNames(value).length === 4 &&
            "enabled" in value &&
            typeof value.enabled === "boolean" &&
            "formattedMessages" in value &&
            Array.isArray(value.formattedMessages) &&
            value.formattedMessages.every((message) => typeof message === "string") &&
            "minSecondsBetweenDisplays" in value &&
            typeof value.minSecondsBetweenDisplays === "number" &&
            "minSecondsBetweenRequests" in value &&
            typeof value.minSecondsBetweenRequests === "number");
    }
    async showBanner(timeout) {
        await this._requestBannerConfig(timeout);
        if (this._bannerConfig === undefined ||
            !this._bannerConfig.enabled ||
            this._bannerConfig.formattedMessages.length === 0) {
            log("Banner is disabled or no messages available.");
            return;
        }
        const { formattedMessages, minSecondsBetweenDisplays } = this._bannerConfig;
        const timeSinceLastDisplay = Date.now() - this._lastDisplayTime;
        if (timeSinceLastDisplay < minSecondsBetweenDisplays * 1000) {
            log(`Skipping banner display. Time since last display: ${timeSinceLastDisplay}ms`);
            return;
        }
        // select a random message from the formattedMessages array
        const randomIndex = Math.floor(Math.random() * formattedMessages.length);
        const message = formattedMessages[randomIndex];
        console.log(message);
        this._lastDisplayTime = Date.now();
        await writeCache({
            bannerConfig: this._bannerConfig,
            lastDisplayTime: this._lastDisplayTime,
            lastRequestTime: this._lastRequestTime,
        });
    }
}
exports.BannerManager = BannerManager;
async function readCache() {
    const cacheDir = await (0, global_dir_1.getCacheDir)();
    const bannerCacheFilePath = node_path_1.default.join(cacheDir, BANNER_CACHE_FILE_NAME);
    let cache = {
        bannerConfig: undefined,
        lastDisplayTime: 0,
        lastRequestTime: 0,
    };
    try {
        const fileContents = await promises_1.default.readFile(bannerCacheFilePath, "utf-8");
        cache = JSON.parse(fileContents);
    }
    catch (error) {
        log(`Error reading cache file: ${error instanceof Error ? error.message : JSON.stringify(error)}`);
    }
    return cache;
}
async function writeCache(cache) {
    const cacheDir = await (0, global_dir_1.getCacheDir)();
    const bannerCacheFilePath = node_path_1.default.join(cacheDir, BANNER_CACHE_FILE_NAME);
    try {
        await promises_1.default.mkdir(cacheDir, { recursive: true });
        await promises_1.default.writeFile(bannerCacheFilePath, JSON.stringify(cache, null, 2));
    }
    catch (error) {
        log(`Error writing cache file:  ${error instanceof Error ? error.message : JSON.stringify(error)}`);
    }
}
//# sourceMappingURL=banner-manager.js.map