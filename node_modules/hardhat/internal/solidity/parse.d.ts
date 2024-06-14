import { SolidityFilesCache } from "../../builtin-tasks/utils/solidity-files-cache";
interface ParsedData {
    imports: string[];
    versionPragmas: string[];
}
export declare class Parser {
    private _cache;
    private _solidityFilesCache;
    constructor(_solidityFilesCache?: SolidityFilesCache);
    parse(fileContent: string, absolutePath: string, contentHash: string): ParsedData;
    /**
     * Get parsed data from the internal cache, or from the solidity files cache.
     *
     * Returns null if cannot find it in either one.
     */
    private _getFromCache;
}
export {};
//# sourceMappingURL=parse.d.ts.map