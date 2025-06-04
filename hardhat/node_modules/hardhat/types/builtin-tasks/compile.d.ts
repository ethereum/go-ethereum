import { SolcConfig } from "../config";
/**
 * A Solidity file.
 */
export interface ResolvedFile {
    library?: LibraryInfo;
    sourceName: string;
    absolutePath: string;
    content: FileContent;
    lastModificationDate: Date;
    contentHash: string;
    getVersionedName(): string;
}
export type ArtifactsEmittedPerFile = Array<{
    file: ResolvedFile;
    artifactsEmitted: string[];
}>;
/**
 * Information about an npm library.
 */
export interface LibraryInfo {
    name: string;
    version: string;
}
/**
 * The content of a Solidity file. Including its raw content, its imports and
 * version pragma directives.
 */
export interface FileContent {
    rawContent: string;
    imports: string[];
    versionPragmas: string[];
}
/**
 * A CompilationJob includes all the necessary information to generate artifacts
 * from a group of files. This includes those files, their dependencies, and the
 * version and configuration of solc that should be used.
 */
export interface CompilationJob {
    emitsArtifacts(file: ResolvedFile): boolean;
    hasSolc9573Bug(): boolean;
    merge(other: CompilationJob): CompilationJob;
    getResolvedFiles(): ResolvedFile[];
    getSolcConfig(): SolcConfig;
}
/**
 * A DependencyGraph represents a group of files and how they depend on each
 * other.
 */
export interface DependencyGraph {
    getConnectedComponents(): DependencyGraph[];
    getDependencies(file: ResolvedFile): ResolvedFile[];
    getResolvedFiles(): ResolvedFile[];
    getTransitiveDependencies(file: ResolvedFile): TransitiveDependency[];
}
/**
 * Used as part of the return value of DependencyGraph.getTransitiveDependencies
 */
export interface TransitiveDependency {
    dependency: ResolvedFile;
    /**
     * The list of intermediate files between the file and the dependency
     * this is not guaranteed to be the shortest path
     */
    path: ResolvedFile[];
}
/**
 * An object with a list of successfully created compilation jobs and a list of
 * errors. The `errors` entry maps error codes (that come from the
 * CompilationJobCreationError enum) to the source names of the files that
 * caused that error.
 */
export interface CompilationJobsCreationResult {
    jobs: CompilationJob[];
    errors: CompilationJobCreationError[];
}
export interface CompilationJobCreationError {
    reason: CompilationJobCreationErrorReason;
    file: ResolvedFile;
    extra?: any;
}
export declare enum CompilationJobCreationErrorReason {
    OTHER_ERROR = "other",
    NO_COMPATIBLE_SOLC_VERSION_FOUND = "no-compatible-solc-version-found",
    INCOMPATIBLE_OVERRIDEN_SOLC_VERSION = "incompatible-overriden-solc-version",
    DIRECTLY_IMPORTS_INCOMPATIBLE_FILE = "directly-imports-incompatible-file",
    INDIRECTLY_IMPORTS_INCOMPATIBLE_FILE = "indirectly-imports-incompatible-file"
}
export interface SolcBuild {
    version: string;
    longVersion: string;
    compilerPath: string;
    isSolcJs: boolean;
}
//# sourceMappingURL=compile.d.ts.map