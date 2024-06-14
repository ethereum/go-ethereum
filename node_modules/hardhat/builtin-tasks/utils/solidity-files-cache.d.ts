import type { ProjectPathsConfig, SolcConfig } from "../../types";
export interface CacheEntry {
    lastModificationDate: number;
    contentHash: string;
    sourceName: string;
    solcConfig: SolcConfig;
    imports: string[];
    versionPragmas: string[];
    artifacts: string[];
}
export interface Cache {
    _format: string;
    files: Record<string, CacheEntry>;
}
export declare class SolidityFilesCache {
    private _cache;
    static createEmpty(): SolidityFilesCache;
    static readFromFile(solidityFilesCachePath: string): Promise<SolidityFilesCache>;
    constructor(_cache: Cache);
    removeNonExistingFiles(): Promise<void>;
    writeToFile(solidityFilesCachePath: string): Promise<void>;
    addFile(absolutePath: string, entry: CacheEntry): void;
    getEntries(): CacheEntry[];
    getEntry(file: string): CacheEntry | undefined;
    removeEntry(file: string): void;
    hasFileChanged(absolutePath: string, contentHash: string, solcConfig?: SolcConfig): boolean;
}
export declare function getSolidityFilesCachePath(paths: ProjectPathsConfig): string;
//# sourceMappingURL=solidity-files-cache.d.ts.map