import path from "node:path";
import fs from "node:fs/promises";
import debug from "debug";
import { getCacheDir } from "../util/global-dir";
import { requestJson } from "../util/request";

const log = debug("hardhat:util:banner-manager");

interface BannerConfig {
  enabled: boolean;
  formattedMessages: string[];
  minSecondsBetweenDisplays: number;
  minSecondsBetweenRequests: number;
}

const BANNER_CONFIG_URL =
  "https://raw.githubusercontent.com/NomicFoundation/hardhat/refs/heads/main/banner-config.json";

const BANNER_CACHE_FILE_NAME = "banner-config.json";

export class BannerManager {
  private static _instance: BannerManager | undefined;

  private constructor(
    private _bannerConfig: BannerConfig | undefined,
    private _lastDisplayTime: number,
    private _lastRequestTime: number
  ) {}

  public static async getInstance(): Promise<BannerManager> {
    if (this._instance === undefined) {
      log("Initializing BannerManager");
      const { bannerConfig, lastDisplayTime, lastRequestTime } =
        await readCache();
      this._instance = new BannerManager(
        bannerConfig,
        lastDisplayTime,
        lastRequestTime
      );
    }

    return this._instance;
  }

  private async _requestBannerConfig(timeout?: number): Promise<void> {
    if (this._bannerConfig !== undefined) {
      const timeSinceLastRequest = Date.now() - this._lastRequestTime;
      if (
        timeSinceLastRequest <
        this._bannerConfig.minSecondsBetweenRequests * 1000
      ) {
        log(
          `Skipping banner config request. Time since last request: ${timeSinceLastRequest}ms`
        );
        return;
      }
    }

    try {
      const bannerConfig = await requestJson(BANNER_CONFIG_URL, timeout);

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
    } catch (error) {
      log(
        `Error requesting banner config: ${
          error instanceof Error ? error.message : JSON.stringify(error)
        }`
      );
    }
  }

  private _isBannerConfig(value: unknown): value is BannerConfig {
    if (typeof value !== "object" || value === null || Array.isArray(value)) {
      return false;
    }

    return (
      Object.getOwnPropertyNames(value).length === 4 &&
      "enabled" in value &&
      typeof value.enabled === "boolean" &&
      "formattedMessages" in value &&
      Array.isArray(value.formattedMessages) &&
      value.formattedMessages.every((message) => typeof message === "string") &&
      "minSecondsBetweenDisplays" in value &&
      typeof value.minSecondsBetweenDisplays === "number" &&
      "minSecondsBetweenRequests" in value &&
      typeof value.minSecondsBetweenRequests === "number"
    );
  }

  public async showBanner(timeout?: number): Promise<void> {
    await this._requestBannerConfig(timeout);

    if (
      this._bannerConfig === undefined ||
      !this._bannerConfig.enabled ||
      this._bannerConfig.formattedMessages.length === 0
    ) {
      log("Banner is disabled or no messages available.");
      return;
    }

    const { formattedMessages, minSecondsBetweenDisplays } = this._bannerConfig;

    const timeSinceLastDisplay = Date.now() - this._lastDisplayTime;
    if (timeSinceLastDisplay < minSecondsBetweenDisplays * 1000) {
      log(
        `Skipping banner display. Time since last display: ${timeSinceLastDisplay}ms`
      );
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

interface BannerCache {
  bannerConfig: BannerConfig | undefined;
  lastDisplayTime: number;
  lastRequestTime: number;
}

async function readCache(): Promise<BannerCache> {
  const cacheDir = await getCacheDir();
  const bannerCacheFilePath = path.join(cacheDir, BANNER_CACHE_FILE_NAME);

  let cache: BannerCache = {
    bannerConfig: undefined,
    lastDisplayTime: 0,
    lastRequestTime: 0,
  };
  try {
    const fileContents = await fs.readFile(bannerCacheFilePath, "utf-8");
    cache = JSON.parse(fileContents);
  } catch (error) {
    log(
      `Error reading cache file: ${
        error instanceof Error ? error.message : JSON.stringify(error)
      }`
    );
  }

  return cache;
}

async function writeCache(cache: BannerCache) {
  const cacheDir = await getCacheDir();
  const bannerCacheFilePath = path.join(cacheDir, BANNER_CACHE_FILE_NAME);

  try {
    await fs.mkdir(cacheDir, { recursive: true });
    await fs.writeFile(bannerCacheFilePath, JSON.stringify(cache, null, 2));
  } catch (error) {
    log(
      `Error writing cache file:  ${
        error instanceof Error ? error.message : JSON.stringify(error)
      }`
    );
  }
}
