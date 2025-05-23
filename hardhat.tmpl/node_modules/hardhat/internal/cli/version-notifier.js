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
exports.showNewVersionNotification = void 0;
const boxen_1 = __importDefault(require("boxen"));
const picocolors_1 = __importDefault(require("picocolors"));
const fs_extra_1 = __importDefault(require("fs-extra"));
const node_path_1 = require("node:path");
const promises_1 = require("node:fs/promises");
const semver_1 = __importDefault(require("semver"));
const global_dir_1 = require("../util/global-dir");
const packageInfo_1 = require("../util/packageInfo");
const GITHUB_API_URL = "https://api.github.com";
const GITHUB_OWNER = "NomicFoundation";
const GITHUB_REPO = "hardhat";
const V3_RELEASE_TAG = "hardhat@3.0.0";
const V3_RELEASE_VERSION_NOTIFIER_ASSET_NAME = "version-notifier-message.txt";
const V3_RELEASE_MAX_TIMES_SHOWN = 5;
const CURRENT_HARDHAT_MAJOR_VERSION = 2;
const boxenOptions = {
    padding: 1,
    borderStyle: "round",
    borderColor: "yellow",
};
/* eslint-enable @typescript-eslint/naming-convention */
async function showNewVersionNotification() {
    const cache = await readCache();
    const lastCheckDate = new Date(cache.lastCheck);
    const now = new Date();
    const oneDay = 1000 * 60 * 60 * 24;
    if (now.getTime() - lastCheckDate.getTime() < oneDay) {
        return;
    }
    const hardhatVersion = (0, packageInfo_1.getHardhatVersion)();
    const releases = await getReleases();
    const sortedV2Versions = releases
        // filter and map releases to versions
        .flatMap((release) => {
        const [packageName, rawPackageVersion] = release.tag_name.split("@");
        const packageVersion = semver_1.default.valid(rawPackageVersion);
        // filter out a release if:
        // - it's not a hardhat-core release
        // - it's a draft or a prerelease
        // - the version is invalid
        // - the major version is not the current major
        if (packageName !== GITHUB_REPO ||
            release.draft ||
            release.prerelease ||
            packageVersion === null ||
            semver_1.default.major(packageVersion) !== CURRENT_HARDHAT_MAJOR_VERSION) {
            return [];
        }
        return [packageVersion];
    })
        // sort in descending order by version
        .sort((releaseAVersion, releaseBVersion) => {
        return semver_1.default.rcompare(releaseAVersion, releaseBVersion);
    });
    const latestV2Version = sortedV2Versions[0];
    const v3Release = cache.v3Release ?? (await getV3Release());
    if (latestV2Version === undefined && v3Release === undefined) {
        // this should never happen unless the github api is down
        return;
    }
    if (latestV2Version !== undefined &&
        semver_1.default.gt(latestV2Version, hardhatVersion)) {
        let installationCommand = "npm install";
        if (await fs_extra_1.default.pathExists("yarn.lock")) {
            installationCommand = "yarn add";
        }
        else if (await fs_extra_1.default.pathExists("pnpm-lock.yaml")) {
            installationCommand = "pnpm install";
        }
        console.log((0, boxen_1.default)(`New Hardhat release available! ${picocolors_1.default.red(hardhatVersion)} -> ${picocolors_1.default.green(latestV2Version)}.

Changelog: https://hardhat.org/release/${latestV2Version}

Run "${installationCommand} hardhat@latest" to update.`, boxenOptions));
    }
    if (v3Release !== undefined &&
        cache.v3TimesShown < V3_RELEASE_MAX_TIMES_SHOWN) {
        const releaseVersion = semver_1.default.valid(v3Release.tag_name.split("@")[1]);
        if (releaseVersion !== null) {
            cache.v3ReleaseMessage ??= await getV3ReleaseMessage(v3Release);
            if (cache.v3ReleaseMessage !== undefined) {
                console.log((0, boxen_1.default)(cache.v3ReleaseMessage, boxenOptions));
                cache.v3TimesShown++;
            }
        }
    }
    await writeCache({
        ...cache,
        lastCheck: now.toISOString(),
        v3Release,
    });
}
exports.showNewVersionNotification = showNewVersionNotification;
async function readCache() {
    const cacheDir = await (0, global_dir_1.getCacheDir)();
    const versionNotifierCachePath = (0, node_path_1.join)(cacheDir, "version-notifier.json");
    let cache = {
        lastCheck: 0,
        v3TimesShown: 0,
    };
    try {
        const fileContents = await (0, promises_1.readFile)(versionNotifierCachePath, "utf-8");
        const { lastCheck, v3TimesShown } = JSON.parse(fileContents);
        cache = {
            lastCheck: typeof lastCheck === "string" ? lastCheck : 0,
            v3TimesShown: typeof v3TimesShown === "number" ? v3TimesShown : 0,
        };
    }
    catch (error) {
        // We don't care if it fails
    }
    return cache;
}
async function writeCache(cache) {
    const cacheDir = await (0, global_dir_1.getCacheDir)();
    const versionNotifierCachePath = (0, node_path_1.join)(cacheDir, "version-notifier.json");
    try {
        await (0, promises_1.mkdir)(cacheDir, { recursive: true });
        await (0, promises_1.writeFile)(versionNotifierCachePath, JSON.stringify(cache, null, 2));
    }
    catch (error) {
        // We don't care if it fails
    }
}
async function getReleases() {
    const { request } = await Promise.resolve().then(() => __importStar(require("undici")));
    let releases = [];
    try {
        const githubResponse = await request(`${GITHUB_API_URL}/repos/${GITHUB_OWNER}/${GITHUB_REPO}/releases`, {
            method: "GET",
            headers: {
                "User-Agent": "Hardhat",
                "X-GitHub-Api-Version": "2022-11-28",
            },
            query: {
                per_page: 100,
            },
        });
        releases = (await githubResponse.body.json());
    }
    catch (error) {
        // We don't care if it fails
    }
    return releases;
}
async function getV3Release() {
    const { request } = await Promise.resolve().then(() => __importStar(require("undici")));
    let v3Release;
    try {
        const githubResponse = await request(`${GITHUB_API_URL}/repos/${GITHUB_OWNER}/${GITHUB_REPO}/releases/tags/${V3_RELEASE_TAG}`, {
            method: "GET",
            headers: {
                "User-Agent": "Hardhat",
                "X-GitHub-Api-Version": "2022-11-28",
            },
        });
        const jsonResponse = (await githubResponse.body.json());
        if (jsonResponse.message === "Not Found") {
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw new Error("Not Found");
        }
        v3Release = jsonResponse;
    }
    catch (error) {
        // We don't care if it fails
    }
    return v3Release;
}
async function getV3ReleaseMessage(v3Release) {
    const { request } = await Promise.resolve().then(() => __importStar(require("undici")));
    const versionNotifierAsset = v3Release.assets.find(({ name }) => name === V3_RELEASE_VERSION_NOTIFIER_ASSET_NAME);
    if (versionNotifierAsset === undefined) {
        return;
    }
    let v3ReleaseMessage;
    try {
        const githubResponse = await request(versionNotifierAsset.browser_download_url, {
            method: "GET",
            maxRedirections: 10,
        });
        v3ReleaseMessage = await githubResponse.body.text();
    }
    catch (error) {
        // We don't care if it fails
    }
    return v3ReleaseMessage;
}
//# sourceMappingURL=version-notifier.js.map