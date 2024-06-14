import fsExtra from "fs-extra";
import path from "path";
import resolve from "resolve";

import {
  FileContent,
  LibraryInfo,
  ResolvedFile as IResolvedFile,
} from "../../types/builtin-tasks";
import {
  includesOwnPackageName,
  isAbsolutePathSourceName,
  isLocalSourceName,
  normalizeSourceName,
  replaceBackslashes,
  validateSourceNameExistenceAndCasing,
  validateSourceNameFormat,
} from "../../utils/source-names";
import { assertHardhatInvariant, HardhatError } from "../core/errors";
import { ERRORS } from "../core/errors-list";
import { createNonCryptographicHashBasedIdentifier } from "../util/hash";

import { getRealPath } from "../util/fs-utils";
import { applyRemappings } from "../../utils/remappings";
import { Parser } from "./parse";

export interface ResolvedFilesMap {
  [sourceName: string]: ResolvedFile;
}

const NODE_MODULES = "node_modules";

export class ResolvedFile implements IResolvedFile {
  public readonly library?: LibraryInfo;

  constructor(
    public readonly sourceName: string,
    public readonly absolutePath: string,
    public readonly content: FileContent,
    public readonly contentHash: string,
    public readonly lastModificationDate: Date,
    libraryName?: string,
    libraryVersion?: string
  ) {
    assertHardhatInvariant(
      (libraryName === undefined && libraryVersion === undefined) ||
        (libraryName !== undefined && libraryVersion !== undefined),
      "Libraries should have both name and version, or neither one"
    );

    if (libraryName !== undefined && libraryVersion !== undefined) {
      this.library = {
        name: libraryName,
        version: libraryVersion,
      };
    }
  }

  public getVersionedName() {
    return (
      this.sourceName +
      (this.library !== undefined ? `@v${this.library.version}` : "")
    );
  }
}

export class Resolver {
  private readonly _cache: Map<string, ResolvedFile> = new Map();

  constructor(
    private readonly _projectRoot: string,
    private readonly _parser: Parser,
    private readonly _remappings: Record<string, string>,
    private readonly _readFile: (absolutePath: string) => Promise<string>,
    private readonly _transformImportName: (
      importName: string
    ) => Promise<string>
  ) {}

  /**
   * Resolves a source name into a ResolvedFile.
   *
   * @param sourceName The source name as it would be provided to solc.
   */
  public async resolveSourceName(sourceName: string): Promise<ResolvedFile> {
    const cached = this._cache.get(sourceName);
    if (cached !== undefined) {
      return cached;
    }

    const remappedSourceName = applyRemappings(this._remappings, sourceName);

    validateSourceNameFormat(remappedSourceName);

    let resolvedFile: ResolvedFile;

    if (await isLocalSourceName(this._projectRoot, remappedSourceName)) {
      resolvedFile = await this._resolveLocalSourceName(
        sourceName,
        remappedSourceName
      );
    } else {
      resolvedFile = await this._resolveLibrarySourceName(
        sourceName,
        remappedSourceName
      );
    }

    this._cache.set(sourceName, resolvedFile);
    return resolvedFile;
  }

  /**
   * Resolves an import from an already resolved file.
   * @param from The file were the import statement is present.
   * @param importName The path in the import statement.
   */
  public async resolveImport(
    from: ResolvedFile,
    importName: string
  ): Promise<ResolvedFile> {
    // sanity check for deprecated task
    if (importName !== (await this._transformImportName(importName))) {
      throw new HardhatError(
        ERRORS.TASK_DEFINITIONS.DEPRECATED_TRANSFORM_IMPORT_TASK
      );
    }

    const imported = applyRemappings(this._remappings, importName);

    const scheme = this._getUriScheme(imported);
    if (scheme !== undefined) {
      throw new HardhatError(ERRORS.RESOLVER.INVALID_IMPORT_PROTOCOL, {
        from: from.sourceName,
        imported,
        protocol: scheme,
      });
    }

    if (replaceBackslashes(imported) !== imported) {
      throw new HardhatError(ERRORS.RESOLVER.INVALID_IMPORT_BACKSLASH, {
        from: from.sourceName,
        imported,
      });
    }

    if (isAbsolutePathSourceName(imported)) {
      throw new HardhatError(ERRORS.RESOLVER.INVALID_IMPORT_ABSOLUTE_PATH, {
        from: from.sourceName,
        imported,
      });
    }

    // Edge-case where an import can contain the current package's name in monorepos.
    // The path can be resolved because there's a symlink in the node modules.
    if (await includesOwnPackageName(imported)) {
      throw new HardhatError(ERRORS.RESOLVER.INCLUDES_OWN_PACKAGE_NAME, {
        from: from.sourceName,
        imported,
      });
    }

    try {
      let sourceName: string;

      const isRelativeImport = this._isRelativeImport(imported);

      if (isRelativeImport) {
        sourceName = await this._relativeImportToSourceName(from, imported);
      } else {
        sourceName = normalizeSourceName(importName); // The sourceName of the imported file is not transformed
      }

      const cached = this._cache.get(sourceName);
      if (cached !== undefined) {
        return cached;
      }

      let resolvedFile: ResolvedFile;

      // We have this special case here, because otherwise local relative
      // imports can be treated as library imports. For example if
      // `contracts/c.sol` imports `../non-existent/a.sol`
      if (
        from.library === undefined &&
        isRelativeImport &&
        !this._isRelativeImportToLibrary(from, imported)
      ) {
        resolvedFile = await this._resolveLocalSourceName(
          sourceName,
          applyRemappings(this._remappings, sourceName)
        );
      } else {
        resolvedFile = await this.resolveSourceName(sourceName);
      }

      this._cache.set(sourceName, resolvedFile);
      return resolvedFile;
    } catch (error) {
      if (
        HardhatError.isHardhatErrorType(
          error,
          ERRORS.RESOLVER.FILE_NOT_FOUND
        ) ||
        HardhatError.isHardhatErrorType(
          error,
          ERRORS.RESOLVER.LIBRARY_FILE_NOT_FOUND
        )
      ) {
        if (imported !== importName) {
          throw new HardhatError(
            ERRORS.RESOLVER.IMPORTED_MAPPED_FILE_NOT_FOUND,
            {
              imported,
              importName,
              from: from.sourceName,
            },
            error
          );
        } else {
          throw new HardhatError(
            ERRORS.RESOLVER.IMPORTED_FILE_NOT_FOUND,
            {
              imported,
              from: from.sourceName,
            },
            error
          );
        }
      }

      if (
        HardhatError.isHardhatErrorType(
          error,
          ERRORS.RESOLVER.WRONG_SOURCE_NAME_CASING
        )
      ) {
        throw new HardhatError(
          ERRORS.RESOLVER.INVALID_IMPORT_WRONG_CASING,
          {
            imported,
            from: from.sourceName,
          },
          error
        );
      }

      if (
        HardhatError.isHardhatErrorType(
          error,
          ERRORS.RESOLVER.LIBRARY_NOT_INSTALLED
        )
      ) {
        throw new HardhatError(
          ERRORS.RESOLVER.IMPORTED_LIBRARY_NOT_INSTALLED,
          {
            library: error.messageArguments.library,
            from: from.sourceName,
          },
          error
        );
      }

      if (
        HardhatError.isHardhatErrorType(
          error,
          ERRORS.GENERAL.INVALID_READ_OF_DIRECTORY
        )
      ) {
        throw new HardhatError(
          ERRORS.RESOLVER.INVALID_IMPORT_OF_DIRECTORY,
          {
            imported,
            from: from.sourceName,
          },
          error
        );
      }

      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw error;
    }
  }

  private async _resolveLocalSourceName(
    sourceName: string,
    remappedSourceName: string
  ): Promise<ResolvedFile> {
    await this._validateSourceNameExistenceAndCasing(
      this._projectRoot,
      remappedSourceName,
      false
    );

    const absolutePath = path.join(this._projectRoot, remappedSourceName);
    return this._resolveFile(sourceName, absolutePath);
  }

  private async _resolveLibrarySourceName(
    sourceName: string,
    remappedSourceName: string
  ): Promise<ResolvedFile> {
    const normalizedSourceName = remappedSourceName.replace(
      /^node_modules\//,
      ""
    );
    const libraryName = this._getLibraryName(normalizedSourceName);

    let packageJsonPath;
    try {
      packageJsonPath = this._resolveNodeModulesFileFromProjectRoot(
        path.join(libraryName, "package.json")
      );
    } catch (error) {
      // if the project is using a dependency from hardhat itself but it can't
      // be found, this means that a global installation is being used, so we
      // resolve the dependency relative to this file
      if (libraryName === "hardhat") {
        const hardhatCoreDir = path.join(__dirname, "..", "..");
        packageJsonPath = path.join(hardhatCoreDir, "package.json");
      } else {
        throw new HardhatError(
          ERRORS.RESOLVER.LIBRARY_NOT_INSTALLED,
          {
            library: libraryName,
          },
          error as Error
        );
      }
    }

    let nodeModulesPath = path.dirname(path.dirname(packageJsonPath));
    if (this._isScopedPackage(normalizedSourceName)) {
      nodeModulesPath = path.dirname(nodeModulesPath);
    }

    let absolutePath: string;
    if (path.basename(nodeModulesPath) !== NODE_MODULES) {
      // this can happen in monorepos that use PnP, in those
      // cases we handle resolution differently
      const packageRoot = path.dirname(packageJsonPath);
      const pattern = new RegExp(`^${libraryName}/?`);
      const fileName = normalizedSourceName.replace(pattern, "");

      await this._validateSourceNameExistenceAndCasing(
        packageRoot,
        // TODO: this is _not_ a source name; we should handle this scenario in
        // a better way
        fileName,
        true
      );
      absolutePath = path.join(packageRoot, fileName);
    } else {
      await this._validateSourceNameExistenceAndCasing(
        nodeModulesPath,
        normalizedSourceName,
        true
      );
      absolutePath = path.join(nodeModulesPath, normalizedSourceName);
    }

    const packageInfo: {
      name: string;
      version: string;
    } = await fsExtra.readJson(packageJsonPath);
    const libraryVersion = packageInfo.version;

    return this._resolveFile(
      sourceName,
      // We resolve to the real path here, as we may be resolving a linked library
      await getRealPath(absolutePath),
      libraryName,
      libraryVersion
    );
  }

  private async _relativeImportToSourceName(
    from: ResolvedFile,
    imported: string
  ): Promise<string> {
    // This is a special case, were we turn relative imports from local files
    // into library imports if necessary. The reason for this is that many
    // users just do `import "../node_modules/lib/a.sol";`.
    if (this._isRelativeImportToLibrary(from, imported)) {
      return this._relativeImportToLibraryToSourceName(from, imported);
    }

    const sourceName = normalizeSourceName(
      path.join(path.dirname(from.sourceName), imported)
    );

    // If the file with the import is local, and the normalized version
    // starts with ../ means that it's trying to get outside of the project.
    if (from.library === undefined && sourceName.startsWith("../")) {
      throw new HardhatError(
        ERRORS.RESOLVER.INVALID_IMPORT_OUTSIDE_OF_PROJECT,
        { from: from.sourceName, imported }
      );
    }

    if (
      from.library !== undefined &&
      !this._isInsideSameDir(from.sourceName, sourceName)
    ) {
      // If the file is being imported from a library, this means that it's
      // trying to reach another one.
      throw new HardhatError(ERRORS.RESOLVER.ILLEGAL_IMPORT, {
        from: from.sourceName,
        imported,
      });
    }

    return sourceName;
  }

  private async _resolveFile(
    sourceName: string,
    absolutePath: string,
    libraryName?: string,
    libraryVersion?: string
  ): Promise<ResolvedFile> {
    const rawContent = await this._readFile(absolutePath);
    const stats = await fsExtra.stat(absolutePath);
    const lastModificationDate = new Date(stats.ctime);

    const contentHash = createNonCryptographicHashBasedIdentifier(
      Buffer.from(rawContent)
    ).toString("hex");

    const parsedContent = this._parser.parse(
      rawContent,
      absolutePath,
      contentHash
    );

    const content = {
      rawContent,
      ...parsedContent,
    };

    return new ResolvedFile(
      sourceName,
      absolutePath,
      content,
      contentHash,
      lastModificationDate,
      libraryName,
      libraryVersion
    );
  }

  private _isRelativeImport(imported: string): boolean {
    return imported.startsWith("./") || imported.startsWith("../");
  }

  private _resolveNodeModulesFileFromProjectRoot(fileName: string) {
    return resolve.sync(fileName, {
      basedir: this._projectRoot,
      preserveSymlinks: true,
    });
  }

  private _getLibraryName(sourceName: string): string {
    let endIndex: number;
    if (this._isScopedPackage(sourceName)) {
      endIndex = sourceName.indexOf("/", sourceName.indexOf("/") + 1);
    } else if (sourceName.indexOf("/") === -1) {
      endIndex = sourceName.length;
    } else {
      endIndex = sourceName.indexOf("/");
    }

    return sourceName.slice(0, endIndex);
  }

  private _getUriScheme(s: string): string | undefined {
    const re = /([a-zA-Z]+):\/\//;
    const match = re.exec(s);
    if (match === null) {
      return undefined;
    }

    return match[1];
  }

  private _isInsideSameDir(sourceNameInDir: string, sourceNameToTest: string) {
    const firstSlash = sourceNameInDir.indexOf("/");
    const dir =
      firstSlash !== -1
        ? sourceNameInDir.substring(0, firstSlash)
        : sourceNameInDir;

    return sourceNameToTest.startsWith(dir);
  }

  private _isScopedPackage(packageOrPackageFile: string): boolean {
    return packageOrPackageFile.startsWith("@");
  }

  private _isRelativeImportToLibrary(
    from: ResolvedFile,
    imported: string
  ): boolean {
    return (
      this._isRelativeImport(imported) &&
      from.library === undefined &&
      imported.includes(`${NODE_MODULES}/`)
    );
  }

  private _relativeImportToLibraryToSourceName(
    from: ResolvedFile,
    imported: string
  ): string {
    const sourceName = normalizeSourceName(
      path.join(path.dirname(from.sourceName), imported)
    );

    const nmIndex = sourceName.indexOf(`${NODE_MODULES}/`);
    return sourceName.substr(nmIndex + NODE_MODULES.length + 1);
  }

  private async _validateSourceNameExistenceAndCasing(
    fromDir: string,
    sourceName: string,
    isLibrary: boolean
  ) {
    try {
      await validateSourceNameExistenceAndCasing(fromDir, sourceName);
    } catch (error) {
      if (
        HardhatError.isHardhatErrorType(
          error,
          ERRORS.SOURCE_NAMES.FILE_NOT_FOUND
        )
      ) {
        throw new HardhatError(
          isLibrary
            ? ERRORS.RESOLVER.LIBRARY_FILE_NOT_FOUND
            : ERRORS.RESOLVER.FILE_NOT_FOUND,
          { file: sourceName },
          error
        );
      }

      if (
        HardhatError.isHardhatErrorType(error, ERRORS.SOURCE_NAMES.WRONG_CASING)
      ) {
        throw new HardhatError(
          ERRORS.RESOLVER.WRONG_SOURCE_NAME_CASING,
          {
            incorrect: sourceName,
            correct: error.messageArguments.correct,
          },
          error
        );
      }

      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw error;
    }
  }
}
