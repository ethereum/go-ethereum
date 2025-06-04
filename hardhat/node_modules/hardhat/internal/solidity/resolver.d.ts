import { FileContent, LibraryInfo, ResolvedFile as IResolvedFile } from "../../types/builtin-tasks";
import { Parser } from "./parse";
export interface ResolvedFilesMap {
    [sourceName: string]: ResolvedFile;
}
export declare class ResolvedFile implements IResolvedFile {
    readonly sourceName: string;
    readonly absolutePath: string;
    readonly content: FileContent;
    readonly contentHash: string;
    readonly lastModificationDate: Date;
    readonly library?: LibraryInfo;
    constructor(sourceName: string, absolutePath: string, content: FileContent, contentHash: string, lastModificationDate: Date, libraryName?: string, libraryVersion?: string);
    getVersionedName(): string;
}
export declare class Resolver {
    private readonly _projectRoot;
    private readonly _parser;
    private readonly _remappings;
    private readonly _readFile;
    private readonly _transformImportName;
    private readonly _cache;
    constructor(_projectRoot: string, _parser: Parser, _remappings: Record<string, string>, _readFile: (absolutePath: string) => Promise<string>, _transformImportName: (importName: string) => Promise<string>);
    /**
     * Resolves a source name into a ResolvedFile.
     *
     * @param sourceName The source name as it would be provided to solc.
     */
    resolveSourceName(sourceName: string): Promise<ResolvedFile>;
    /**
     * Resolves an import from an already resolved file.
     * @param from The file were the import statement is present.
     * @param importName The path in the import statement.
     */
    resolveImport(from: ResolvedFile, importName: string): Promise<ResolvedFile>;
    private _resolveLocalSourceName;
    private _resolveLibrarySourceName;
    private _relativeImportToSourceName;
    private _resolveFile;
    private _isRelativeImport;
    private _resolveNodeModulesFileFromProjectRoot;
    private _getLibraryName;
    private _getUriScheme;
    private _isInsideSameDir;
    private _isScopedPackage;
    private _isRelativeImportToLibrary;
    private _relativeImportToLibraryToSourceName;
    private _validateSourceNameExistenceAndCasing;
}
//# sourceMappingURL=resolver.d.ts.map