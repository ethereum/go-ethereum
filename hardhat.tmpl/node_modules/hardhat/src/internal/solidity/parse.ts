import type SolidityAnalyzerT from "@nomicfoundation/solidity-analyzer";

import { SolidityFilesCache } from "../../builtin-tasks/utils/solidity-files-cache";
import { requireNapiRsModule } from "../../common/napi-rs";

interface ParsedData {
  imports: string[];
  versionPragmas: string[];
}

export class Parser {
  private _cache = new Map<string, ParsedData>();
  private _solidityFilesCache: SolidityFilesCache;

  constructor(_solidityFilesCache?: SolidityFilesCache) {
    this._solidityFilesCache =
      _solidityFilesCache ?? SolidityFilesCache.createEmpty();
  }

  public parse(
    fileContent: string,
    absolutePath: string,
    contentHash: string
  ): ParsedData {
    const cacheResult = this._getFromCache(absolutePath, contentHash);

    if (cacheResult !== null) {
      return cacheResult;
    }

    const { analyze } = requireNapiRsModule(
      "@nomicfoundation/solidity-analyzer"
    ) as typeof SolidityAnalyzerT;
    const result = analyze(fileContent);

    this._cache.set(contentHash, result);

    return result;
  }

  /**
   * Get parsed data from the internal cache, or from the solidity files cache.
   *
   * Returns null if cannot find it in either one.
   */
  private _getFromCache(
    absolutePath: string,
    contentHash: string
  ): ParsedData | null {
    const internalCacheEntry = this._cache.get(contentHash);

    if (internalCacheEntry !== undefined) {
      return internalCacheEntry;
    }

    const solidityFilesCacheEntry =
      this._solidityFilesCache.getEntry(absolutePath);

    if (solidityFilesCacheEntry === undefined) {
      return null;
    }

    const { imports, versionPragmas } = solidityFilesCacheEntry;

    if (solidityFilesCacheEntry.contentHash !== contentHash) {
      return null;
    }

    return { imports, versionPragmas };
  }
}
