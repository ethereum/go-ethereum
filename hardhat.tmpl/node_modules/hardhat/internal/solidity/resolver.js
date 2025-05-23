"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Resolver = exports.ResolvedFile = void 0;
const fs_extra_1 = __importDefault(require("fs-extra"));
const path_1 = __importDefault(require("path"));
const resolve_1 = __importDefault(require("resolve"));
const source_names_1 = require("../../utils/source-names");
const errors_1 = require("../core/errors");
const errors_list_1 = require("../core/errors-list");
const hash_1 = require("../util/hash");
const fs_utils_1 = require("../util/fs-utils");
const remappings_1 = require("../../utils/remappings");
const NODE_MODULES = "node_modules";
class ResolvedFile {
    constructor(sourceName, absolutePath, content, contentHash, lastModificationDate, libraryName, libraryVersion) {
        this.sourceName = sourceName;
        this.absolutePath = absolutePath;
        this.content = content;
        this.contentHash = contentHash;
        this.lastModificationDate = lastModificationDate;
        (0, errors_1.assertHardhatInvariant)((libraryName === undefined && libraryVersion === undefined) ||
            (libraryName !== undefined && libraryVersion !== undefined), "Libraries should have both name and version, or neither one");
        if (libraryName !== undefined && libraryVersion !== undefined) {
            this.library = {
                name: libraryName,
                version: libraryVersion,
            };
        }
    }
    getVersionedName() {
        return (this.sourceName +
            (this.library !== undefined ? `@v${this.library.version}` : ""));
    }
}
exports.ResolvedFile = ResolvedFile;
class Resolver {
    constructor(_projectRoot, _parser, _remappings, _readFile, _transformImportName) {
        this._projectRoot = _projectRoot;
        this._parser = _parser;
        this._remappings = _remappings;
        this._readFile = _readFile;
        this._transformImportName = _transformImportName;
        this._cache = new Map();
    }
    /**
     * Resolves a source name into a ResolvedFile.
     *
     * @param sourceName The source name as it would be provided to solc.
     */
    async resolveSourceName(sourceName) {
        const cached = this._cache.get(sourceName);
        if (cached !== undefined) {
            return cached;
        }
        const remappedSourceName = (0, remappings_1.applyRemappings)(this._remappings, sourceName);
        (0, source_names_1.validateSourceNameFormat)(remappedSourceName);
        let resolvedFile;
        if (await (0, source_names_1.isLocalSourceName)(this._projectRoot, remappedSourceName)) {
            resolvedFile = await this._resolveLocalSourceName(sourceName, remappedSourceName);
        }
        else {
            resolvedFile = await this._resolveLibrarySourceName(sourceName, remappedSourceName);
        }
        this._cache.set(sourceName, resolvedFile);
        return resolvedFile;
    }
    /**
     * Resolves an import from an already resolved file.
     * @param from The file were the import statement is present.
     * @param importName The path in the import statement.
     */
    async resolveImport(from, importName) {
        // sanity check for deprecated task
        if (importName !== (await this._transformImportName(importName))) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.DEPRECATED_TRANSFORM_IMPORT_TASK);
        }
        const imported = (0, remappings_1.applyRemappings)(this._remappings, importName);
        const scheme = this._getUriScheme(imported);
        if (scheme !== undefined) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.INVALID_IMPORT_PROTOCOL, {
                from: from.sourceName,
                imported,
                protocol: scheme,
            });
        }
        if ((0, source_names_1.replaceBackslashes)(imported) !== imported) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.INVALID_IMPORT_BACKSLASH, {
                from: from.sourceName,
                imported,
            });
        }
        if ((0, source_names_1.isAbsolutePathSourceName)(imported)) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.INVALID_IMPORT_ABSOLUTE_PATH, {
                from: from.sourceName,
                imported,
            });
        }
        // Edge-case where an import can contain the current package's name in monorepos.
        // The path can be resolved because there's a symlink in the node modules.
        if (await (0, source_names_1.includesOwnPackageName)(imported)) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.INCLUDES_OWN_PACKAGE_NAME, {
                from: from.sourceName,
                imported,
            });
        }
        try {
            let sourceName;
            const isRelativeImport = this._isRelativeImport(imported);
            if (isRelativeImport) {
                sourceName = await this._relativeImportToSourceName(from, imported);
            }
            else {
                sourceName = (0, source_names_1.normalizeSourceName)(importName); // The sourceName of the imported file is not transformed
            }
            const cached = this._cache.get(sourceName);
            if (cached !== undefined) {
                return cached;
            }
            let resolvedFile;
            // We have this special case here, because otherwise local relative
            // imports can be treated as library imports. For example if
            // `contracts/c.sol` imports `../non-existent/a.sol`
            if (from.library === undefined &&
                isRelativeImport &&
                !this._isRelativeImportToLibrary(from, imported)) {
                resolvedFile = await this._resolveLocalSourceName(sourceName, (0, remappings_1.applyRemappings)(this._remappings, sourceName));
            }
            else {
                resolvedFile = await this.resolveSourceName(sourceName);
            }
            this._cache.set(sourceName, resolvedFile);
            return resolvedFile;
        }
        catch (error) {
            if (errors_1.HardhatError.isHardhatErrorType(error, errors_list_1.ERRORS.RESOLVER.FILE_NOT_FOUND) ||
                errors_1.HardhatError.isHardhatErrorType(error, errors_list_1.ERRORS.RESOLVER.LIBRARY_FILE_NOT_FOUND)) {
                if (imported !== importName) {
                    throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.IMPORTED_MAPPED_FILE_NOT_FOUND, {
                        imported,
                        importName,
                        from: from.sourceName,
                    }, error);
                }
                else {
                    throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.IMPORTED_FILE_NOT_FOUND, {
                        imported,
                        from: from.sourceName,
                    }, error);
                }
            }
            if (errors_1.HardhatError.isHardhatErrorType(error, errors_list_1.ERRORS.RESOLVER.WRONG_SOURCE_NAME_CASING)) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.INVALID_IMPORT_WRONG_CASING, {
                    imported,
                    from: from.sourceName,
                }, error);
            }
            if (errors_1.HardhatError.isHardhatErrorType(error, errors_list_1.ERRORS.RESOLVER.LIBRARY_NOT_INSTALLED)) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.IMPORTED_LIBRARY_NOT_INSTALLED, {
                    library: error.messageArguments.library,
                    from: from.sourceName,
                }, error);
            }
            if (errors_1.HardhatError.isHardhatErrorType(error, errors_list_1.ERRORS.GENERAL.INVALID_READ_OF_DIRECTORY)) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.INVALID_IMPORT_OF_DIRECTORY, {
                    imported,
                    from: from.sourceName,
                }, error);
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw error;
        }
    }
    async _resolveLocalSourceName(sourceName, remappedSourceName) {
        await this._validateSourceNameExistenceAndCasing(this._projectRoot, remappedSourceName, false);
        const absolutePath = path_1.default.join(this._projectRoot, remappedSourceName);
        return this._resolveFile(sourceName, absolutePath);
    }
    async _resolveLibrarySourceName(sourceName, remappedSourceName) {
        const normalizedSourceName = remappedSourceName.replace(/^node_modules\//, "");
        const libraryName = this._getLibraryName(normalizedSourceName);
        let packageJsonPath;
        try {
            packageJsonPath = this._resolveNodeModulesFileFromProjectRoot(path_1.default.join(libraryName, "package.json"));
        }
        catch (error) {
            // if the project is using a dependency from hardhat itself but it can't
            // be found, this means that a global installation is being used, so we
            // resolve the dependency relative to this file
            if (libraryName === "hardhat") {
                const hardhatCoreDir = path_1.default.join(__dirname, "..", "..");
                packageJsonPath = path_1.default.join(hardhatCoreDir, "package.json");
            }
            else {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.LIBRARY_NOT_INSTALLED, {
                    library: libraryName,
                }, error);
            }
        }
        let nodeModulesPath = path_1.default.dirname(path_1.default.dirname(packageJsonPath));
        if (this._isScopedPackage(normalizedSourceName)) {
            nodeModulesPath = path_1.default.dirname(nodeModulesPath);
        }
        let absolutePath;
        if (path_1.default.basename(nodeModulesPath) !== NODE_MODULES) {
            // this can happen in monorepos that use PnP, in those
            // cases we handle resolution differently
            const packageRoot = path_1.default.dirname(packageJsonPath);
            const pattern = new RegExp(`^${libraryName}/?`);
            const fileName = normalizedSourceName.replace(pattern, "");
            await this._validateSourceNameExistenceAndCasing(packageRoot, 
            // TODO: this is _not_ a source name; we should handle this scenario in
            // a better way
            fileName, true);
            absolutePath = path_1.default.join(packageRoot, fileName);
        }
        else {
            await this._validateSourceNameExistenceAndCasing(nodeModulesPath, normalizedSourceName, true);
            absolutePath = path_1.default.join(nodeModulesPath, normalizedSourceName);
        }
        const packageInfo = await fs_extra_1.default.readJson(packageJsonPath);
        const libraryVersion = packageInfo.version;
        return this._resolveFile(sourceName, 
        // We resolve to the real path here, as we may be resolving a linked library
        await (0, fs_utils_1.getRealPath)(absolutePath), libraryName, libraryVersion);
    }
    async _relativeImportToSourceName(from, imported) {
        // This is a special case, were we turn relative imports from local files
        // into library imports if necessary. The reason for this is that many
        // users just do `import "../node_modules/lib/a.sol";`.
        if (this._isRelativeImportToLibrary(from, imported)) {
            return this._relativeImportToLibraryToSourceName(from, imported);
        }
        const sourceName = (0, source_names_1.normalizeSourceName)(path_1.default.join(path_1.default.dirname(from.sourceName), imported));
        // If the file with the import is local, and the normalized version
        // starts with ../ means that it's trying to get outside of the project.
        if (from.library === undefined && sourceName.startsWith("../")) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.INVALID_IMPORT_OUTSIDE_OF_PROJECT, { from: from.sourceName, imported });
        }
        if (from.library !== undefined &&
            !this._isInsideSameDir(from.sourceName, sourceName)) {
            // If the file is being imported from a library, this means that it's
            // trying to reach another one.
            throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.ILLEGAL_IMPORT, {
                from: from.sourceName,
                imported,
            });
        }
        return sourceName;
    }
    async _resolveFile(sourceName, absolutePath, libraryName, libraryVersion) {
        const rawContent = await this._readFile(absolutePath);
        const stats = await fs_extra_1.default.stat(absolutePath);
        const lastModificationDate = new Date(stats.ctime);
        const contentHash = (0, hash_1.createNonCryptographicHashBasedIdentifier)(Buffer.from(rawContent)).toString("hex");
        const parsedContent = this._parser.parse(rawContent, absolutePath, contentHash);
        const content = {
            rawContent,
            ...parsedContent,
        };
        return new ResolvedFile(sourceName, absolutePath, content, contentHash, lastModificationDate, libraryName, libraryVersion);
    }
    _isRelativeImport(imported) {
        return imported.startsWith("./") || imported.startsWith("../");
    }
    _resolveNodeModulesFileFromProjectRoot(fileName) {
        return resolve_1.default.sync(fileName, {
            basedir: this._projectRoot,
            preserveSymlinks: true,
        });
    }
    _getLibraryName(sourceName) {
        let endIndex;
        if (this._isScopedPackage(sourceName)) {
            endIndex = sourceName.indexOf("/", sourceName.indexOf("/") + 1);
        }
        else if (sourceName.indexOf("/") === -1) {
            endIndex = sourceName.length;
        }
        else {
            endIndex = sourceName.indexOf("/");
        }
        return sourceName.slice(0, endIndex);
    }
    _getUriScheme(s) {
        const re = /([a-zA-Z]+):\/\//;
        const match = re.exec(s);
        if (match === null) {
            return undefined;
        }
        return match[1];
    }
    _isInsideSameDir(sourceNameInDir, sourceNameToTest) {
        const firstSlash = sourceNameInDir.indexOf("/");
        const dir = firstSlash !== -1
            ? sourceNameInDir.substring(0, firstSlash)
            : sourceNameInDir;
        return sourceNameToTest.startsWith(dir);
    }
    _isScopedPackage(packageOrPackageFile) {
        return packageOrPackageFile.startsWith("@");
    }
    _isRelativeImportToLibrary(from, imported) {
        return (this._isRelativeImport(imported) &&
            from.library === undefined &&
            imported.includes(`${NODE_MODULES}/`));
    }
    _relativeImportToLibraryToSourceName(from, imported) {
        const sourceName = (0, source_names_1.normalizeSourceName)(path_1.default.join(path_1.default.dirname(from.sourceName), imported));
        const nmIndex = sourceName.indexOf(`${NODE_MODULES}/`);
        return sourceName.substr(nmIndex + NODE_MODULES.length + 1);
    }
    async _validateSourceNameExistenceAndCasing(fromDir, sourceName, isLibrary) {
        try {
            await (0, source_names_1.validateSourceNameExistenceAndCasing)(fromDir, sourceName);
        }
        catch (error) {
            if (errors_1.HardhatError.isHardhatErrorType(error, errors_list_1.ERRORS.SOURCE_NAMES.FILE_NOT_FOUND)) {
                throw new errors_1.HardhatError(isLibrary
                    ? errors_list_1.ERRORS.RESOLVER.LIBRARY_FILE_NOT_FOUND
                    : errors_list_1.ERRORS.RESOLVER.FILE_NOT_FOUND, { file: sourceName }, error);
            }
            if (errors_1.HardhatError.isHardhatErrorType(error, errors_list_1.ERRORS.SOURCE_NAMES.WRONG_CASING)) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.RESOLVER.WRONG_SOURCE_NAME_CASING, {
                    incorrect: sourceName,
                    correct: error.messageArguments.correct,
                }, error);
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw error;
        }
    }
}
exports.Resolver = Resolver;
//# sourceMappingURL=resolver.js.map