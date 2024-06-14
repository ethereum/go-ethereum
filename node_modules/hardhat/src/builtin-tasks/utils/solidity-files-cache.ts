import type { LoDashStatic } from "lodash";
import type { ProjectPathsConfig, SolcConfig } from "../../types";

import debug from "debug";
import fsExtra from "fs-extra";
import * as t from "io-ts";
import * as path from "path";

import { SOLIDITY_FILES_CACHE_FILENAME } from "../../internal/constants";

const log = debug("hardhat:core:tasks:compile:cache");

const FORMAT_VERSION = "hh-sol-cache-2";

const CacheEntryCodec = t.type({
  lastModificationDate: t.number,
  contentHash: t.string,
  sourceName: t.string,
  solcConfig: t.any,
  imports: t.array(t.string),
  versionPragmas: t.array(t.string),
  artifacts: t.array(t.string),
});

const CacheCodec = t.type({
  _format: t.string,
  files: t.record(t.string, CacheEntryCodec),
});

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

export class SolidityFilesCache {
  public static createEmpty(): SolidityFilesCache {
    return new SolidityFilesCache({
      _format: FORMAT_VERSION,
      files: {},
    });
  }

  public static async readFromFile(
    solidityFilesCachePath: string
  ): Promise<SolidityFilesCache> {
    let cacheRaw: Cache = {
      _format: FORMAT_VERSION,
      files: {},
    };
    if (await fsExtra.pathExists(solidityFilesCachePath)) {
      cacheRaw = await fsExtra.readJson(solidityFilesCachePath);
    }

    const result = CacheCodec.decode(cacheRaw);

    if (result.isRight()) {
      const solidityFilesCache = new SolidityFilesCache(result.value);
      await solidityFilesCache.removeNonExistingFiles();
      return solidityFilesCache;
    }

    log("There was a problem reading the cache");

    return new SolidityFilesCache({
      _format: FORMAT_VERSION,
      files: {},
    });
  }

  constructor(private _cache: Cache) {}

  public async removeNonExistingFiles() {
    await Promise.all(
      Object.keys(this._cache.files).map(async (absolutePath) => {
        if (!(await fsExtra.pathExists(absolutePath))) {
          this.removeEntry(absolutePath);
        }
      })
    );
  }

  public async writeToFile(solidityFilesCachePath: string) {
    await fsExtra.outputJson(solidityFilesCachePath, this._cache, {
      spaces: 2,
    });
  }

  public addFile(absolutePath: string, entry: CacheEntry) {
    this._cache.files[absolutePath] = entry;
  }

  public getEntries(): CacheEntry[] {
    return Object.values(this._cache.files);
  }

  public getEntry(file: string): CacheEntry | undefined {
    return this._cache.files[file];
  }

  public removeEntry(file: string) {
    delete this._cache.files[file];
  }

  public hasFileChanged(
    absolutePath: string,
    contentHash: string,
    solcConfig?: SolcConfig
  ): boolean {
    const isEqual = require("lodash/isEqual") as LoDashStatic["isEqual"];

    const cacheEntry = this.getEntry(absolutePath);

    if (cacheEntry === undefined) {
      // new file or no cache available, assume it's new
      return true;
    }

    if (cacheEntry.contentHash !== contentHash) {
      return true;
    }

    if (
      solcConfig !== undefined &&
      !isEqual(solcConfig, cacheEntry.solcConfig)
    ) {
      return true;
    }

    return false;
  }
}

export function getSolidityFilesCachePath(paths: ProjectPathsConfig): string {
  return path.join(paths.cache, SOLIDITY_FILES_CACHE_FILENAME);
}
