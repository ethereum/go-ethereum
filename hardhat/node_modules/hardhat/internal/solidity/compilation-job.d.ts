import { SolcConfig, SolidityConfig } from "../../types";
import * as taskTypes from "../../types/builtin-tasks";
import { CompilationJobCreationError, CompilationJobsCreationResult } from "../../types/builtin-tasks";
import { ResolvedFile } from "./resolver";
export declare class CompilationJob implements taskTypes.CompilationJob {
    solidityConfig: SolcConfig;
    private _filesToCompile;
    constructor(solidityConfig: SolcConfig);
    addFileToCompile(file: ResolvedFile, emitsArtifacts: boolean): void;
    hasSolc9573Bug(): boolean;
    merge(job: taskTypes.CompilationJob): CompilationJob;
    getSolcConfig(): SolcConfig;
    isEmpty(): boolean;
    getResolvedFiles(): ResolvedFile[];
    /**
     * Check if the given file emits artifacts.
     *
     * If no file is given, check if *some* file in the job emits artifacts.
     */
    emitsArtifacts(file: ResolvedFile): boolean;
}
/**
 * Creates a list of compilation jobs from a dependency graph. *This function
 * assumes that the given graph is a connected component*.
 * Returns the list of compilation jobs on success, and a list of
 * non-compilable files on failure.
 */
export declare function createCompilationJobsFromConnectedComponent(connectedComponent: taskTypes.DependencyGraph, getFromFile: (file: ResolvedFile) => Promise<taskTypes.CompilationJob | CompilationJobCreationError>): Promise<CompilationJobsCreationResult>;
export declare function createCompilationJobFromFile(dependencyGraph: taskTypes.DependencyGraph, file: ResolvedFile, solidityConfig: SolidityConfig): Promise<CompilationJob | CompilationJobCreationError>;
/**
 * Merge compilation jobs affected by the solc #9573 bug
 */
export declare function mergeCompilationJobsWithBug(compilationJobs: taskTypes.CompilationJob[]): taskTypes.CompilationJob[];
/**
 * Merge compilation jobs not affected by the solc #9573 bug
 */
export declare function mergeCompilationJobsWithoutBug(compilationJobs: taskTypes.CompilationJob[]): taskTypes.CompilationJob[];
//# sourceMappingURL=compilation-job.d.ts.map