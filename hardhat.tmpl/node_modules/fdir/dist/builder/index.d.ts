/// <reference types="node" />
import { Output, OnlyCountsOutput, GroupOutput, PathsOutput, Options, FilterPredicate, ExcludePredicate, GlobParams } from "../types";
import { APIBuilder } from "./api-builder";
import type picomatch from "picomatch";
export declare class Builder<TReturnType extends Output = PathsOutput, TGlobFunction = typeof picomatch> {
    private readonly globCache;
    private options;
    private globFunction?;
    constructor(options?: Partial<Options<TGlobFunction>>);
    group(): Builder<GroupOutput, TGlobFunction>;
    withPathSeparator(separator: "/" | "\\"): this;
    withBasePath(): this;
    withRelativePaths(): this;
    withDirs(): this;
    withMaxDepth(depth: number): this;
    withMaxFiles(limit: number): this;
    withFullPaths(): this;
    withErrors(): this;
    withSymlinks({ resolvePaths }?: {
        resolvePaths?: boolean | undefined;
    }): this;
    withAbortSignal(signal: AbortSignal): this;
    normalize(): this;
    filter(predicate: FilterPredicate): this;
    onlyDirs(): this;
    exclude(predicate: ExcludePredicate): this;
    onlyCounts(): Builder<OnlyCountsOutput, TGlobFunction>;
    crawl(root?: string): APIBuilder<TReturnType>;
    withGlobFunction<TFunc>(fn: TFunc): Builder<TReturnType, TFunc>;
    /**
     * @deprecated Pass options using the constructor instead:
     * ```ts
     * new fdir(options).crawl("/path/to/root");
     * ```
     * This method will be removed in v7.0
     */
    crawlWithOptions(root: string, options: Partial<Options<TGlobFunction>>): APIBuilder<TReturnType>;
    glob(...patterns: string[]): Builder<TReturnType, TGlobFunction>;
    globWithOptions(patterns: string[]): Builder<TReturnType, TGlobFunction>;
    globWithOptions(patterns: string[], ...options: GlobParams<TGlobFunction>): Builder<TReturnType, TGlobFunction>;
}
