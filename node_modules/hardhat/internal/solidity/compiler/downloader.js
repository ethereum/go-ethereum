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
exports.CompilerDownloader = exports.CompilerPlatform = void 0;
const path_1 = __importDefault(require("path"));
const fs_extra_1 = __importDefault(require("fs-extra"));
const debug_1 = __importDefault(require("debug"));
const os_1 = __importDefault(require("os"));
const child_process_1 = require("child_process");
const util_1 = require("util");
const download_1 = require("../../util/download");
const errors_1 = require("../../core/errors");
const errors_list_1 = require("../../core/errors-list");
const multi_process_mutex_1 = require("../../util/multi-process-mutex");
const log = (0, debug_1.default)("hardhat:core:solidity:downloader");
const COMPILER_REPOSITORY_URL = "https://binaries.soliditylang.org";
var CompilerPlatform;
(function (CompilerPlatform) {
    CompilerPlatform["LINUX"] = "linux-amd64";
    CompilerPlatform["WINDOWS"] = "windows-amd64";
    CompilerPlatform["MACOS"] = "macosx-amd64";
    CompilerPlatform["WASM"] = "wasm";
})(CompilerPlatform = exports.CompilerPlatform || (exports.CompilerPlatform = {}));
/**
 * Default implementation of ICompilerDownloader.
 *
 * Important things to note:
 *   1. If a compiler version is not found, this downloader may fail.
 *    1.1. It only re-downloads the list of compilers once every X time.
 *      1.1.1 If a user tries to download a new compiler before X amount of time
 *      has passed since its release, they may need to clean the cache, as
 *      indicated in the error messages.
 */
class CompilerDownloader {
    static getCompilerPlatform() {
        // TODO: This check is seriously wrong. It doesn't take into account
        //  the architecture nor the toolchain. This should check the triplet of
        //  system instead (see: https://wiki.osdev.org/Target_Triplet).
        //
        //  The only reason this downloader works is that it validates if the
        //  binaries actually run.
        switch (os_1.default.platform()) {
            case "win32":
                return CompilerPlatform.WINDOWS;
            case "linux":
                return CompilerPlatform.LINUX;
            case "darwin":
                return CompilerPlatform.MACOS;
            default:
                return CompilerPlatform.WASM;
        }
    }
    static getConcurrencySafeDownloader(platform, compilersDir) {
        const key = platform + compilersDir;
        if (!this._downloaderPerPlatform.has(key)) {
            this._downloaderPerPlatform.set(key, new CompilerDownloader(platform, compilersDir));
        }
        return this._downloaderPerPlatform.get(key);
    }
    /**
     * Use CompilerDownloader.getConcurrencySafeDownloader instead
     */
    constructor(_platform, _compilersDir, _compilerListCachePeriodMs = CompilerDownloader.defaultCompilerListCachePeriod, _downloadFunction = download_1.download) {
        this._platform = _platform;
        this._compilersDir = _compilersDir;
        this._compilerListCachePeriodMs = _compilerListCachePeriodMs;
        this._downloadFunction = _downloadFunction;
        this._mutex = new multi_process_mutex_1.MultiProcessMutex("compiler-download");
    }
    async isCompilerDownloaded(version) {
        const build = await this._getCompilerBuild(version);
        if (build === undefined) {
            return false;
        }
        const downloadPath = this._getCompilerBinaryPathFromBuild(build);
        return fs_extra_1.default.pathExists(downloadPath);
    }
    async downloadCompiler(version, downloadStartedCb, downloadEndedCb) {
        // Since only one process at a time can acquire the mutex, we avoid the risk of downloading the same compiler multiple times.
        // This is because the mutex blocks access until a compiler has been fully downloaded, preventing any new process
        // from checking whether that version of the compiler exists. Without mutex it might incorrectly
        // return false, indicating that the compiler isn't present, even though it is currently being downloaded.
        await this._mutex.use(async () => {
            const isCompilerDownloaded = await this.isCompilerDownloaded(version);
            if (isCompilerDownloaded === true) {
                return;
            }
            await downloadStartedCb(isCompilerDownloaded);
            let build = await this._getCompilerBuild(version);
            if (build === undefined && (await this._shouldDownloadCompilerList())) {
                try {
                    await this._downloadCompilerList();
                }
                catch (e) {
                    throw new errors_1.HardhatError(errors_list_1.ERRORS.SOLC.VERSION_LIST_DOWNLOAD_FAILED, {}, e);
                }
                build = await this._getCompilerBuild(version);
            }
            if (build === undefined) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.SOLC.INVALID_VERSION, { version });
            }
            let downloadPath;
            try {
                downloadPath = await this._downloadCompiler(build);
            }
            catch (e) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.SOLC.DOWNLOAD_FAILED, {
                    remoteVersion: build.longVersion,
                }, e);
            }
            const verified = await this._verifyCompilerDownload(build, downloadPath);
            if (!verified) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.SOLC.INVALID_DOWNLOAD, {
                    remoteVersion: build.longVersion,
                });
            }
            await this._postProcessCompilerDownload(build, downloadPath);
            await downloadEndedCb(isCompilerDownloaded);
        });
    }
    async getCompiler(version) {
        const build = await this._getCompilerBuild(version);
        (0, errors_1.assertHardhatInvariant)(build !== undefined, "Trying to get a compiler before it was downloaded");
        const compilerPath = this._getCompilerBinaryPathFromBuild(build);
        (0, errors_1.assertHardhatInvariant)(await fs_extra_1.default.pathExists(compilerPath), "Trying to get a compiler before it was downloaded");
        if (await fs_extra_1.default.pathExists(this._getCompilerDoesntWorkFile(build))) {
            return undefined;
        }
        return {
            version,
            longVersion: build.longVersion,
            compilerPath,
            isSolcJs: this._platform === CompilerPlatform.WASM,
        };
    }
    async _getCompilerBuild(version) {
        const listPath = this._getCompilerListPath();
        if (!(await fs_extra_1.default.pathExists(listPath))) {
            return undefined;
        }
        const list = await this._readCompilerList(listPath);
        return list.builds.find((b) => b.version === version);
    }
    _getCompilerListPath() {
        return path_1.default.join(this._compilersDir, this._platform, "list.json");
    }
    async _readCompilerList(listPath) {
        return fs_extra_1.default.readJSON(listPath);
    }
    _getCompilerDownloadPathFromBuild(build) {
        return path_1.default.join(this._compilersDir, this._platform, build.path);
    }
    _getCompilerBinaryPathFromBuild(build) {
        const downloadPath = this._getCompilerDownloadPathFromBuild(build);
        if (this._platform !== CompilerPlatform.WINDOWS ||
            !downloadPath.endsWith(".zip")) {
            return downloadPath;
        }
        return path_1.default.join(this._compilersDir, build.version, "solc.exe");
    }
    _getCompilerDoesntWorkFile(build) {
        return `${this._getCompilerBinaryPathFromBuild(build)}.does.not.work`;
    }
    async _shouldDownloadCompilerList() {
        const listPath = this._getCompilerListPath();
        if (!(await fs_extra_1.default.pathExists(listPath))) {
            return true;
        }
        const stats = await fs_extra_1.default.stat(listPath);
        const age = new Date().valueOf() - stats.ctimeMs;
        return age > this._compilerListCachePeriodMs;
    }
    async _downloadCompilerList() {
        log(`Downloading compiler list for platform ${this._platform}`);
        const url = `${COMPILER_REPOSITORY_URL}/${this._platform}/list.json`;
        const downloadPath = this._getCompilerListPath();
        await this._downloadFunction(url, downloadPath);
    }
    async _downloadCompiler(build) {
        log(`Downloading compiler ${build.longVersion}`);
        const url = `${COMPILER_REPOSITORY_URL}/${this._platform}/${build.path}`;
        const downloadPath = this._getCompilerDownloadPathFromBuild(build);
        await this._downloadFunction(url, downloadPath);
        return downloadPath;
    }
    async _verifyCompilerDownload(build, downloadPath) {
        const { bytesToHex } = require("@nomicfoundation/ethereumjs-util");
        const { keccak256 } = await Promise.resolve().then(() => __importStar(require("../../util/keccak")));
        const expectedKeccak256 = build.keccak256;
        const compiler = await fs_extra_1.default.readFile(downloadPath);
        const compilerKeccak256 = bytesToHex(keccak256(compiler));
        if (expectedKeccak256 !== compilerKeccak256) {
            await fs_extra_1.default.unlink(downloadPath);
            return false;
        }
        return true;
    }
    async _postProcessCompilerDownload(build, downloadPath) {
        if (this._platform === CompilerPlatform.WASM) {
            return;
        }
        if (this._platform === CompilerPlatform.LINUX ||
            this._platform === CompilerPlatform.MACOS) {
            fs_extra_1.default.chmodSync(downloadPath, 0o755);
        }
        else if (this._platform === CompilerPlatform.WINDOWS &&
            downloadPath.endsWith(".zip")) {
            // some window builds are zipped, some are not
            const AdmZip = require("adm-zip");
            const solcFolder = path_1.default.join(this._compilersDir, build.version);
            await fs_extra_1.default.ensureDir(solcFolder);
            const zip = new AdmZip(downloadPath);
            zip.extractAllTo(solcFolder);
        }
        log("Checking native solc binary");
        const nativeSolcWorks = await this._checkNativeSolc(build);
        if (nativeSolcWorks) {
            return;
        }
        await fs_extra_1.default.createFile(this._getCompilerDoesntWorkFile(build));
    }
    async _checkNativeSolc(build) {
        const solcPath = this._getCompilerBinaryPathFromBuild(build);
        const execFileP = (0, util_1.promisify)(child_process_1.execFile);
        try {
            await execFileP(solcPath, ["--version"]);
            return true;
        }
        catch {
            return false;
        }
    }
}
CompilerDownloader._downloaderPerPlatform = new Map();
CompilerDownloader.defaultCompilerListCachePeriod = 360000;
exports.CompilerDownloader = CompilerDownloader;
//# sourceMappingURL=downloader.js.map