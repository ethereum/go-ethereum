import { download } from "../../util/download";
export declare enum CompilerPlatform {
    LINUX = "linux-amd64",
    WINDOWS = "windows-amd64",
    MACOS = "macosx-amd64",
    WASM = "wasm"
}
export interface Compiler {
    version: string;
    longVersion: string;
    compilerPath: string;
    isSolcJs: boolean;
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
    downloadCompiler(version: string, downloadStartedCb: (isCompilerDownloaded: boolean) => Promise<any>, downloadEndedCb: (isCompilerDownloaded: boolean) => Promise<any>): Promise<void>;
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
export declare class CompilerDownloader implements ICompilerDownloader {
    private readonly _platform;
    private readonly _compilersDir;
    private readonly _compilerListCachePeriodMs;
    private readonly _downloadFunction;
    static getCompilerPlatform(): CompilerPlatform;
    private static _downloaderPerPlatform;
    static getConcurrencySafeDownloader(platform: CompilerPlatform, compilersDir: string): CompilerDownloader;
    static defaultCompilerListCachePeriod: number;
    private readonly _mutex;
    /**
     * Use CompilerDownloader.getConcurrencySafeDownloader instead
     */
    constructor(_platform: CompilerPlatform, _compilersDir: string, _compilerListCachePeriodMs?: number, _downloadFunction?: typeof download);
    isCompilerDownloaded(version: string): Promise<boolean>;
    downloadCompiler(version: string, downloadStartedCb: (isCompilerDownloaded: boolean) => Promise<any>, downloadEndedCb: (isCompilerDownloaded: boolean) => Promise<any>): Promise<void>;
    getCompiler(version: string): Promise<Compiler | undefined>;
    private _getCompilerBuild;
    private _getCompilerListPath;
    private _readCompilerList;
    private _getCompilerDownloadPathFromBuild;
    private _getCompilerBinaryPathFromBuild;
    private _getCompilerDoesntWorkFile;
    private _shouldDownloadCompilerList;
    private _downloadCompilerList;
    private _downloadCompiler;
    private _verifyCompilerDownload;
    private _postProcessCompilerDownload;
    private _checkNativeSolc;
}
//# sourceMappingURL=downloader.d.ts.map