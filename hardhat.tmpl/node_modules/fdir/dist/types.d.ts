/// <reference types="node" />
import { Queue } from "./api/queue";
export type Counts = {
    files: number;
    directories: number;
    /**
     * @deprecated use `directories` instead. Will be removed in v7.0.
     */
    dirs: number;
};
export type Group = {
    directory: string;
    files: string[];
    /**
     * @deprecated use `directory` instead. Will be removed in v7.0.
     */
    dir: string;
};
export type GroupOutput = Group[];
export type OnlyCountsOutput = Counts;
export type PathsOutput = string[];
export type Output = OnlyCountsOutput | PathsOutput | GroupOutput;
export type WalkerState = {
    root: string;
    paths: string[];
    groups: Group[];
    counts: Counts;
    options: Options;
    queue: Queue;
    symlinks: Map<string, string>;
    visited: string[];
};
export type ResultCallback<TOutput extends Output> = (error: Error | null, output: TOutput) => void;
export type FilterPredicate = (path: string, isDirectory: boolean) => boolean;
export type ExcludePredicate = (dirName: string, dirPath: string) => boolean;
export type PathSeparator = "/" | "\\";
export type Options<TGlobFunction = unknown> = {
    includeBasePath?: boolean;
    includeDirs?: boolean;
    normalizePath?: boolean;
    maxDepth: number;
    maxFiles?: number;
    resolvePaths?: boolean;
    suppressErrors: boolean;
    group?: boolean;
    onlyCounts?: boolean;
    filters: FilterPredicate[];
    resolveSymlinks?: boolean;
    useRealPaths?: boolean;
    excludeFiles?: boolean;
    excludeSymlinks?: boolean;
    exclude?: ExcludePredicate;
    relativePaths?: boolean;
    pathSeparator: PathSeparator;
    signal?: AbortSignal;
    globFunction?: TGlobFunction;
};
export type GlobMatcher = (test: string) => boolean;
export type GlobFunction = (glob: string | string[], ...params: unknown[]) => GlobMatcher;
export type GlobParams<T> = T extends (globs: string | string[], ...params: infer TParams extends unknown[]) => GlobMatcher ? TParams : [];
