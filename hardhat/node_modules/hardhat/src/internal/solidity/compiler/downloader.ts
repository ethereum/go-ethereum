import path from "path";
import fsExtra from "fs-extra";
import debug from "debug";
import os from "os";
import { execFile } from "child_process";
import { promisify } from "util";

import { download } from "../../util/download";
import { assertHardhatInvariant, HardhatError } from "../../core/errors";
import { ERRORS } from "../../core/errors-list";
import { MultiProcessMutex } from "../../util/multi-process-mutex";

const log = debug("hardhat:core:solidity:downloader");

const COMPILER_REPOSITORY_URL = "https://binaries.soliditylang.org";

export enum CompilerPlatform {
  LINUX = "linux-amd64",
  WINDOWS = "windows-amd64",
  MACOS = "macosx-amd64",
  WASM = "wasm",
}

export interface Compiler {
  version: string;
  longVersion: string;
  compilerPath: string;
  isSolcJs: boolean;
}

interface CompilerBuild {
  path: string;
  version: string;
  build: string;
  longVersion: string;
  keccak256: string;
  urls: string[];
  platform: CompilerPlatform;
}

interface CompilerList {
  builds: CompilerBuild[];
  releases: { [version: string]: string };
  latestRelease: string;
}

/**
 * A compiler downloader which must be specialized per-platform. It can't and
 * shouldn't support multiple platforms at the same time.
 */
export interface ICompilerDownloader {
  /**
   * Returns true if the compiler has been downloaded.
   *
   * This function access the filesystem, but doesn't modify it.
   */
  isCompilerDownloaded(version: string): Promise<boolean>;

  /**
   * Downloads the compiler for a given version, which can later be obtained
   * with getCompiler.
   */
  downloadCompiler(
    version: string,
    downloadStartedCb: (isCompilerDownloaded: boolean) => Promise<any>,
    downloadEndedCb: (isCompilerDownloaded: boolean) => Promise<any>
  ): Promise<void>;

  /**
   * Returns the compiler, which MUST be downloaded before calling this function.
   *
   * Returns undefined if the compiler has been downloaded but can't be run.
   *
   * This function access the filesystem, but doesn't modify it.
   */
  getCompiler(version: string): Promise<Compiler | undefined>;
}

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
export class CompilerDownloader implements ICompilerDownloader {
  public static getCompilerPlatform(): CompilerPlatform {
    // TODO: This check is seriously wrong. It doesn't take into account
    //  the architecture nor the toolchain. This should check the triplet of
    //  system instead (see: https://wiki.osdev.org/Target_Triplet).
    //
    //  The only reason this downloader works is that it validates if the
    //  binaries actually run.
    switch (os.platform()) {
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

  private static _downloaderPerPlatform: Map<string, CompilerDownloader> =
    new Map();

  public static getConcurrencySafeDownloader(
    platform: CompilerPlatform,
    compilersDir: string
  ) {
    const key = platform + compilersDir;

    if (!this._downloaderPerPlatform.has(key)) {
      this._downloaderPerPlatform.set(
        key,
        new CompilerDownloader(platform, compilersDir)
      );
    }

    return this._downloaderPerPlatform.get(key)!;
  }

  public static defaultCompilerListCachePeriod = 3_600_00;
  private readonly _mutex = new MultiProcessMutex("compiler-download");

  /**
   * Use CompilerDownloader.getConcurrencySafeDownloader instead
   */
  constructor(
    private readonly _platform: CompilerPlatform,
    private readonly _compilersDir: string,
    private readonly _compilerListCachePeriodMs = CompilerDownloader.defaultCompilerListCachePeriod,
    private readonly _downloadFunction: typeof download = download
  ) {}

  public async isCompilerDownloaded(version: string): Promise<boolean> {
    const build = await this._getCompilerBuild(version);

    if (build === undefined) {
      return false;
    }

    const downloadPath = this._getCompilerBinaryPathFromBuild(build);

    return fsExtra.pathExists(downloadPath);
  }

  public async downloadCompiler(
    version: string,
    downloadStartedCb: (isCompilerDownloaded: boolean) => Promise<any>,
    downloadEndedCb: (isCompilerDownloaded: boolean) => Promise<any>
  ): Promise<void> {
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
        } catch (e: any) {
          throw new HardhatError(
            ERRORS.SOLC.VERSION_LIST_DOWNLOAD_FAILED,
            {},
            e
          );
        }

        build = await this._getCompilerBuild(version);
      }

      if (build === undefined) {
        throw new HardhatError(ERRORS.SOLC.INVALID_VERSION, { version });
      }

      let downloadPath: string;
      try {
        downloadPath = await this._downloadCompiler(build);
      } catch (e: any) {
        throw new HardhatError(
          ERRORS.SOLC.DOWNLOAD_FAILED,
          {
            remoteVersion: build.longVersion,
          },
          e
        );
      }

      const verified = await this._verifyCompilerDownload(build, downloadPath);
      if (!verified) {
        throw new HardhatError(ERRORS.SOLC.INVALID_DOWNLOAD, {
          remoteVersion: build.longVersion,
        });
      }

      await this._postProcessCompilerDownload(build, downloadPath);

      await downloadEndedCb(isCompilerDownloaded);
    });
  }

  public async getCompiler(version: string): Promise<Compiler | undefined> {
    const build = await this._getCompilerBuild(version);

    assertHardhatInvariant(
      build !== undefined,
      "Trying to get a compiler before it was downloaded"
    );

    const compilerPath = this._getCompilerBinaryPathFromBuild(build);

    assertHardhatInvariant(
      await fsExtra.pathExists(compilerPath),
      "Trying to get a compiler before it was downloaded"
    );

    if (await fsExtra.pathExists(this._getCompilerDoesntWorkFile(build))) {
      return undefined;
    }

    return {
      version,
      longVersion: build.longVersion,
      compilerPath,
      isSolcJs: this._platform === CompilerPlatform.WASM,
    };
  }

  private async _getCompilerBuild(
    version: string
  ): Promise<CompilerBuild | undefined> {
    const listPath = this._getCompilerListPath();
    if (!(await fsExtra.pathExists(listPath))) {
      return undefined;
    }

    const list = await this._readCompilerList(listPath);
    return list.builds.find((b) => b.version === version);
  }

  private _getCompilerListPath(): string {
    return path.join(this._compilersDir, this._platform, "list.json");
  }

  private async _readCompilerList(listPath: string): Promise<CompilerList> {
    return fsExtra.readJSON(listPath);
  }

  private _getCompilerDownloadPathFromBuild(build: CompilerBuild): string {
    return path.join(this._compilersDir, this._platform, build.path);
  }

  private _getCompilerBinaryPathFromBuild(build: CompilerBuild): string {
    const downloadPath = this._getCompilerDownloadPathFromBuild(build);

    if (
      this._platform !== CompilerPlatform.WINDOWS ||
      !downloadPath.endsWith(".zip")
    ) {
      return downloadPath;
    }

    return path.join(this._compilersDir, build.version, "solc.exe");
  }

  private _getCompilerDoesntWorkFile(build: CompilerBuild): string {
    return `${this._getCompilerBinaryPathFromBuild(build)}.does.not.work`;
  }

  private async _shouldDownloadCompilerList(): Promise<boolean> {
    const listPath = this._getCompilerListPath();
    if (!(await fsExtra.pathExists(listPath))) {
      return true;
    }

    const stats = await fsExtra.stat(listPath);
    const age = new Date().valueOf() - stats.ctimeMs;

    return age > this._compilerListCachePeriodMs;
  }

  private async _downloadCompilerList(): Promise<void> {
    log(`Downloading compiler list for platform ${this._platform}`);
    const url = `${COMPILER_REPOSITORY_URL}/${this._platform}/list.json`;
    const downloadPath = this._getCompilerListPath();

    await this._downloadFunction(url, downloadPath);
  }

  private async _downloadCompiler(build: CompilerBuild): Promise<string> {
    log(`Downloading compiler ${build.longVersion}`);
    const url = `${COMPILER_REPOSITORY_URL}/${this._platform}/${build.path}`;
    const downloadPath = this._getCompilerDownloadPathFromBuild(build);

    await this._downloadFunction(url, downloadPath);

    return downloadPath;
  }

  private async _verifyCompilerDownload(
    build: CompilerBuild,
    downloadPath: string
  ): Promise<boolean> {
    const { bytesToHex } =
      require("@ethereumjs/util") as typeof import("@ethereumjs/util");
    const { keccak256 } = await import("../../util/keccak");

    const expectedKeccak256 = build.keccak256;
    const compiler = await fsExtra.readFile(downloadPath);

    const compilerKeccak256 = bytesToHex(keccak256(compiler));

    if (expectedKeccak256 !== compilerKeccak256) {
      await fsExtra.unlink(downloadPath);
      return false;
    }

    return true;
  }

  private async _postProcessCompilerDownload(
    build: CompilerBuild,
    downloadPath: string
  ): Promise<void> {
    if (this._platform === CompilerPlatform.WASM) {
      return;
    }

    if (
      this._platform === CompilerPlatform.LINUX ||
      this._platform === CompilerPlatform.MACOS
    ) {
      fsExtra.chmodSync(downloadPath, 0o755);
    } else if (
      this._platform === CompilerPlatform.WINDOWS &&
      downloadPath.endsWith(".zip")
    ) {
      // some window builds are zipped, some are not
      const AdmZip = require("adm-zip");

      const solcFolder = path.join(this._compilersDir, build.version);
      await fsExtra.ensureDir(solcFolder);

      const zip = new AdmZip(downloadPath);
      zip.extractAllTo(solcFolder);
    }

    log("Checking native solc binary");
    const nativeSolcWorks = await this._checkNativeSolc(build);

    if (nativeSolcWorks) {
      return;
    }

    await fsExtra.createFile(this._getCompilerDoesntWorkFile(build));
  }

  private async _checkNativeSolc(build: CompilerBuild): Promise<boolean> {
    const solcPath = this._getCompilerBinaryPathFromBuild(build);
    const execFileP = promisify(execFile);

    try {
      await execFileP(solcPath, ["--version"]);
      return true;
    } catch {
      return false;
    }
  }
}
