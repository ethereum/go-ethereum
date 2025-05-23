import * as taskTypes from "../../types/builtin-tasks";
import { ResolvedFile, Resolver } from "./resolver";
export declare class DependencyGraph implements taskTypes.DependencyGraph {
    static createFromResolvedFiles(resolver: Resolver, resolvedFiles: ResolvedFile[]): Promise<DependencyGraph>;
    private _resolvedFiles;
    private _dependenciesPerFile;
    private readonly _visitedFiles;
    private constructor();
    getResolvedFiles(): ResolvedFile[];
    has(file: ResolvedFile): boolean;
    isEmpty(): boolean;
    entries(): Array<[ResolvedFile, Set<ResolvedFile>]>;
    getDependencies(file: ResolvedFile): ResolvedFile[];
    getTransitiveDependencies(file: ResolvedFile): taskTypes.TransitiveDependency[];
    getConnectedComponents(): DependencyGraph[];
    private _getTransitiveDependencies;
    private _addDependenciesFrom;
}
//# sourceMappingURL=dependencyGraph.d.ts.map