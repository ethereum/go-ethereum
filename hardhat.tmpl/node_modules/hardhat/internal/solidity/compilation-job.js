"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.mergeCompilationJobsWithoutBug = exports.mergeCompilationJobsWithBug = exports.createCompilationJobFromFile = exports.createCompilationJobsFromConnectedComponent = exports.CompilationJob = void 0;
const debug_1 = __importDefault(require("debug"));
const semver_1 = __importDefault(require("semver"));
const builtin_tasks_1 = require("../../types/builtin-tasks");
const errors_1 = require("../core/errors");
const log = (0, debug_1.default)("hardhat:core:compilation-job");
// this should have a proper version range when it's fixed
const SOLC_BUG_9573_VERSIONS = "<0.8.0";
function isCompilationJobCreationError(x) {
    return "reason" in x;
}
class CompilationJob {
    constructor(solidityConfig) {
        this.solidityConfig = solidityConfig;
        this._filesToCompile = new Map();
    }
    addFileToCompile(file, emitsArtifacts) {
        const fileToCompile = this._filesToCompile.get(file.sourceName);
        // if the file doesn't exist, we add it
        // we also add it if emitsArtifacts is true, to override it in case it was
        // previously added but with a false emitsArtifacts
        if (fileToCompile === undefined || emitsArtifacts) {
            this._filesToCompile.set(file.sourceName, { file, emitsArtifacts });
        }
    }
    hasSolc9573Bug() {
        return (this.solidityConfig?.settings?.optimizer?.enabled === true &&
            semver_1.default.satisfies(this.solidityConfig.version, SOLC_BUG_9573_VERSIONS));
    }
    merge(job) {
        const isEqual = require("lodash/isEqual");
        (0, errors_1.assertHardhatInvariant)(isEqual(this.solidityConfig, job.getSolcConfig()), "Merging jobs with different solidity configurations");
        const mergedJobs = new CompilationJob(job.getSolcConfig());
        for (const file of this.getResolvedFiles()) {
            mergedJobs.addFileToCompile(file, this.emitsArtifacts(file));
        }
        for (const file of job.getResolvedFiles()) {
            mergedJobs.addFileToCompile(file, job.emitsArtifacts(file));
        }
        return mergedJobs;
    }
    getSolcConfig() {
        return this.solidityConfig;
    }
    isEmpty() {
        return this._filesToCompile.size === 0;
    }
    getResolvedFiles() {
        return [...this._filesToCompile.values()].map((x) => x.file);
    }
    /**
     * Check if the given file emits artifacts.
     *
     * If no file is given, check if *some* file in the job emits artifacts.
     */
    emitsArtifacts(file) {
        const fileToCompile = this._filesToCompile.get(file.sourceName);
        (0, errors_1.assertHardhatInvariant)(fileToCompile !== undefined, `File '${file.sourceName}' does not exist in this compilation job`);
        return fileToCompile.emitsArtifacts;
    }
}
exports.CompilationJob = CompilationJob;
function mergeCompilationJobs(jobs, isMergeable) {
    const jobsMap = new Map();
    for (const job of jobs) {
        const mergedJobs = jobsMap.get(job.getSolcConfig());
        if (isMergeable(job)) {
            if (mergedJobs === undefined) {
                jobsMap.set(job.getSolcConfig(), [job]);
            }
            else if (mergedJobs.length === 1) {
                const newJob = mergedJobs[0].merge(job);
                jobsMap.set(job.getSolcConfig(), [newJob]);
            }
            else {
                (0, errors_1.assertHardhatInvariant)(false, "More than one mergeable job was added for the same configuration");
            }
        }
        else {
            if (mergedJobs === undefined) {
                jobsMap.set(job.getSolcConfig(), [job]);
            }
            else {
                jobsMap.set(job.getSolcConfig(), [...mergedJobs, job]);
            }
        }
    }
    // Array#flat This method defaults to depth limit 1
    return [...jobsMap.values()].flat(1000000);
}
/**
 * Creates a list of compilation jobs from a dependency graph. *This function
 * assumes that the given graph is a connected component*.
 * Returns the list of compilation jobs on success, and a list of
 * non-compilable files on failure.
 */
async function createCompilationJobsFromConnectedComponent(connectedComponent, getFromFile) {
    const compilationJobs = [];
    const errors = [];
    for (const file of connectedComponent.getResolvedFiles()) {
        const compilationJobOrError = await getFromFile(file);
        if (isCompilationJobCreationError(compilationJobOrError)) {
            log(`'${file.absolutePath}' couldn't be compiled. Reason: '${compilationJobOrError}'`);
            errors.push(compilationJobOrError);
            continue;
        }
        compilationJobs.push(compilationJobOrError);
    }
    const jobs = mergeCompilationJobsWithBug(compilationJobs);
    return { jobs, errors };
}
exports.createCompilationJobsFromConnectedComponent = createCompilationJobsFromConnectedComponent;
async function createCompilationJobFromFile(dependencyGraph, file, solidityConfig) {
    const directDependencies = dependencyGraph.getDependencies(file);
    const transitiveDependencies = dependencyGraph.getTransitiveDependencies(file);
    const compilerConfig = getCompilerConfigForFile(file, directDependencies, transitiveDependencies, solidityConfig);
    // if the config cannot be obtained, we just return the failure
    if (isCompilationJobCreationError(compilerConfig)) {
        return compilerConfig;
    }
    log(`File '${file.absolutePath}' will be compiled with version '${compilerConfig.version}'`);
    const compilationJob = new CompilationJob(compilerConfig);
    compilationJob.addFileToCompile(file, true);
    for (const { dependency } of transitiveDependencies) {
        log(`File '${dependency.absolutePath}' added as dependency of '${file.absolutePath}'`);
        compilationJob.addFileToCompile(dependency, false);
    }
    return compilationJob;
}
exports.createCompilationJobFromFile = createCompilationJobFromFile;
/**
 * Merge compilation jobs affected by the solc #9573 bug
 */
function mergeCompilationJobsWithBug(compilationJobs) {
    return mergeCompilationJobs(compilationJobs, (job) => job.hasSolc9573Bug());
}
exports.mergeCompilationJobsWithBug = mergeCompilationJobsWithBug;
/**
 * Merge compilation jobs not affected by the solc #9573 bug
 */
function mergeCompilationJobsWithoutBug(compilationJobs) {
    return mergeCompilationJobs(compilationJobs, (job) => !job.hasSolc9573Bug());
}
exports.mergeCompilationJobsWithoutBug = mergeCompilationJobsWithoutBug;
/**
 * Return the compiler config with the newest version that satisfies the given
 * version ranges, or a value indicating why the compiler couldn't be obtained.
 */
function getCompilerConfigForFile(file, directDependencies, transitiveDependencies, solidityConfig) {
    const transitiveDependenciesVersionPragmas = transitiveDependencies
        .map(({ dependency }) => dependency.content.versionPragmas)
        .flat();
    const versionRange = Array.from(new Set([
        ...file.content.versionPragmas,
        ...transitiveDependenciesVersionPragmas,
    ])).join(" ");
    const overrides = solidityConfig.overrides ?? {};
    const overriddenCompiler = overrides[file.sourceName];
    // if there's an override, we only check that
    if (overriddenCompiler !== undefined) {
        if (!semver_1.default.satisfies(overriddenCompiler.version, versionRange)) {
            return getCompilationJobCreationError(file, directDependencies, transitiveDependencies, [overriddenCompiler.version], true);
        }
        return overriddenCompiler;
    }
    // if there's no override, we find a compiler that matches the version range
    const compilerVersions = solidityConfig.compilers.map((x) => x.version);
    const matchingVersion = semver_1.default.maxSatisfying(compilerVersions, versionRange);
    if (matchingVersion === null) {
        return getCompilationJobCreationError(file, directDependencies, transitiveDependencies, compilerVersions, false);
    }
    const matchingConfig = solidityConfig.compilers.find((x) => x.version === matchingVersion);
    return matchingConfig;
}
function getCompilationJobCreationError(file, directDependencies, transitiveDependencies, compilerVersions, overriden) {
    const fileVersionRange = file.content.versionPragmas.join(" ");
    if (semver_1.default.maxSatisfying(compilerVersions, fileVersionRange) === null) {
        const reason = overriden
            ? builtin_tasks_1.CompilationJobCreationErrorReason.INCOMPATIBLE_OVERRIDEN_SOLC_VERSION
            : builtin_tasks_1.CompilationJobCreationErrorReason.NO_COMPATIBLE_SOLC_VERSION_FOUND;
        return { reason, file };
    }
    const incompatibleDirectImports = [];
    for (const dependency of directDependencies) {
        const dependencyVersionRange = dependency.content.versionPragmas.join(" ");
        if (!semver_1.default.intersects(fileVersionRange, dependencyVersionRange)) {
            incompatibleDirectImports.push(dependency);
        }
    }
    if (incompatibleDirectImports.length > 0) {
        return {
            reason: builtin_tasks_1.CompilationJobCreationErrorReason.DIRECTLY_IMPORTS_INCOMPATIBLE_FILE,
            file,
            extra: {
                incompatibleDirectImports,
            },
        };
    }
    const incompatibleIndirectImports = [];
    for (const transitiveDependency of transitiveDependencies) {
        const { dependency } = transitiveDependency;
        const dependencyVersionRange = dependency.content.versionPragmas.join(" ");
        if (!semver_1.default.intersects(fileVersionRange, dependencyVersionRange)) {
            incompatibleIndirectImports.push(transitiveDependency);
        }
    }
    if (incompatibleIndirectImports.length > 0) {
        return {
            reason: builtin_tasks_1.CompilationJobCreationErrorReason.INDIRECTLY_IMPORTS_INCOMPATIBLE_FILE,
            file,
            extra: {
                incompatibleIndirectImports,
            },
        };
    }
    return { reason: builtin_tasks_1.CompilationJobCreationErrorReason.OTHER_ERROR, file };
}
//# sourceMappingURL=compilation-job.js.map