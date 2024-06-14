"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Parser = void 0;
const solidity_files_cache_1 = require("../../builtin-tasks/utils/solidity-files-cache");
const napi_rs_1 = require("../../common/napi-rs");
class Parser {
    constructor(_solidityFilesCache) {
        this._cache = new Map();
        this._solidityFilesCache =
            _solidityFilesCache ?? solidity_files_cache_1.SolidityFilesCache.createEmpty();
    }
    parse(fileContent, absolutePath, contentHash) {
        const cacheResult = this._getFromCache(absolutePath, contentHash);
        if (cacheResult !== null) {
            return cacheResult;
        }
        const { analyze } = (0, napi_rs_1.requireNapiRsModule)("@nomicfoundation/solidity-analyzer");
        const result = analyze(fileContent);
        this._cache.set(contentHash, result);
        return result;
    }
    /**
     * Get parsed data from the internal cache, or from the solidity files cache.
     *
     * Returns null if cannot find it in either one.
     */
    _getFromCache(absolutePath, contentHash) {
        const internalCacheEntry = this._cache.get(contentHash);
        if (internalCacheEntry !== undefined) {
            return internalCacheEntry;
        }
        const solidityFilesCacheEntry = this._solidityFilesCache.getEntry(absolutePath);
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
exports.Parser = Parser;
//# sourceMappingURL=parse.js.map