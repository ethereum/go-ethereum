import os from "os";
import chalk from "chalk";
import debug from "debug";
import fsExtra from "fs-extra";
import semver from "semver";
import AggregateError from "aggregate-error";

import {
  Artifacts as ArtifactsImpl,
  getArtifactFromContractOutput,
} from "../internal/artifacts";
import { subtask, task, types } from "../internal/core/config/config-env";
import { assertHardhatInvariant, HardhatError } from "../internal/core/errors";
import { ERRORS } from "../internal/core/errors-list";
import {
  createCompilationJobFromFile,
  createCompilationJobsFromConnectedComponent,
  mergeCompilationJobsWithoutBug,
} from "../internal/solidity/compilation-job";
import { Compiler, NativeCompiler } from "../internal/solidity/compiler";
import { getInputFromCompilationJob } from "../internal/solidity/compiler/compiler-input";
import {
  CompilerDownloader,
  CompilerPlatform,
} from "../internal/solidity/compiler/downloader";
import { DependencyGraph } from "../internal/solidity/dependencyGraph";
import { Parser } from "../internal/solidity/parse";
import { ResolvedFile, Resolver } from "../internal/solidity/resolver";
import { getCompilersDir } from "../internal/util/global-dir";
import { pluralize } from "../internal/util/strings";
import { Artifacts, CompilerInput, CompilerOutput, SolcBuild } from "../types";
import * as taskTypes from "../types/builtin-tasks";
import {
  CompilationJob,
  CompilationJobCreationError,
  CompilationJobCreationErrorReason,
  CompilationJobsCreationResult,
} from "../types/builtin-tasks";
import { getFullyQualifiedName } from "../utils/contract-names";
import { localPathToSourceName } from "../utils/source-names";

import { getAllFilesMatching } from "../internal/util/fs-utils";
import { getEvmVersionFromSolcVersion } from "../internal/solidity/compiler/solc-info";
import {
  TASK_COMPILE,
  TASK_COMPILE_GET_COMPILATION_TASKS,
  TASK_COMPILE_REMOVE_OBSOLETE_ARTIFACTS,
  TASK_COMPILE_SOLIDITY,
  TASK_COMPILE_SOLIDITY_CHECK_ERRORS,
  TASK_COMPILE_SOLIDITY_COMPILE,
  TASK_COMPILE_SOLIDITY_COMPILE_JOB,
  TASK_COMPILE_SOLIDITY_COMPILE_JOBS,
  TASK_COMPILE_SOLIDITY_COMPILE_SOLC,
  TASK_COMPILE_SOLIDITY_EMIT_ARTIFACTS,
  TASK_COMPILE_SOLIDITY_FILTER_COMPILATION_JOBS,
  TASK_COMPILE_SOLIDITY_GET_ARTIFACT_FROM_COMPILATION_OUTPUT,
  TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOB_FOR_FILE,
  TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOBS,
  TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOBS_FAILURE_REASONS,
  TASK_COMPILE_SOLIDITY_GET_COMPILER_INPUT,
  TASK_COMPILE_SOLIDITY_GET_DEPENDENCY_GRAPH,
  TASK_COMPILE_SOLIDITY_GET_SOLC_BUILD,
  TASK_COMPILE_SOLIDITY_GET_SOURCE_NAMES,
  TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS,
  TASK_COMPILE_SOLIDITY_HANDLE_COMPILATION_JOBS_FAILURES,
  TASK_COMPILE_SOLIDITY_LOG_COMPILATION_ERRORS,
  TASK_COMPILE_SOLIDITY_LOG_COMPILATION_RESULT,
  TASK_COMPILE_SOLIDITY_LOG_DOWNLOAD_COMPILER_END,
  TASK_COMPILE_SOLIDITY_LOG_DOWNLOAD_COMPILER_START,
  TASK_COMPILE_SOLIDITY_LOG_NOTHING_TO_COMPILE,
  TASK_COMPILE_SOLIDITY_LOG_RUN_COMPILER_END,
  TASK_COMPILE_SOLIDITY_LOG_RUN_COMPILER_START,
  TASK_COMPILE_SOLIDITY_MERGE_COMPILATION_JOBS,
  TASK_COMPILE_SOLIDITY_READ_FILE,
  TASK_COMPILE_SOLIDITY_RUN_SOLC,
  TASK_COMPILE_SOLIDITY_RUN_SOLCJS,
  TASK_COMPILE_TRANSFORM_IMPORT_NAME,
  TASK_COMPILE_GET_REMAPPINGS,
} from "./task-names";
import {
  getSolidityFilesCachePath,
  SolidityFilesCache,
} from "./utils/solidity-files-cache";

type ArtifactsEmittedPerJob = Array<{
  compilationJob: CompilationJob;
  artifactsEmittedPerFile: taskTypes.ArtifactsEmittedPerFile;
}>;

function isConsoleLogError(error: any): boolean {
  const message = error.message;

  return (
    error.type === "TypeError" &&
    typeof message === "string" &&
    message.includes("log") &&
    message.includes("type(library console)")
  );
}

const log = debug("hardhat:core:tasks:compile");

const COMPILE_TASK_FIRST_SOLC_VERSION_SUPPORTED = "0.4.11";

const DEFAULT_CONCURRENCY_LEVEL = Math.max(os.cpus().length - 1, 1);

/**
 * Returns a list of absolute paths to all the solidity files in the project.
 * This list doesn't include dependencies, for example solidity files inside
 * node_modules.
 *
 * This is the right task to override to change how the solidity files of the
 * project are obtained.
 */
subtask(TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS)
  .addOptionalParam("sourcePath", undefined, undefined, types.string)
  .setAction(
    async (
      { sourcePath }: { sourcePath?: string },
      { config }
    ): Promise<string[]> => {
      return getAllFilesMatching(sourcePath ?? config.paths.sources, (f) =>
        f.endsWith(".sol")
      );
    }
  );

/**
 * Receives a list of absolute paths and returns a list of source names
 * corresponding to each path. For example, receives
 * ["/home/user/project/contracts/Foo.sol"] and returns
 * ["contracts/Foo.sol"]. These source names will be used when the solc input
 * is generated.
 */
subtask(TASK_COMPILE_SOLIDITY_GET_SOURCE_NAMES)
  .addOptionalParam("rootPath", undefined, undefined, types.string)
  .addParam("sourcePaths", undefined, undefined, types.any)
  .setAction(
    async (
      {
        rootPath,
        sourcePaths,
      }: {
        rootPath?: string;
        sourcePaths: string[];
      },
      { config }
    ): Promise<string[]> => {
      return Promise.all(
        sourcePaths.map((p) =>
          localPathToSourceName(rootPath ?? config.paths.root, p)
        )
      );
    }
  );

subtask(TASK_COMPILE_SOLIDITY_READ_FILE)
  .addParam("absolutePath", undefined, undefined, types.string)
  .setAction(
    async ({ absolutePath }: { absolutePath: string }): Promise<string> => {
      try {
        return await fsExtra.readFile(absolutePath, {
          encoding: "utf8",
        });
      } catch (e) {
        if (fsExtra.lstatSync(absolutePath).isDirectory()) {
          throw new HardhatError(ERRORS.GENERAL.INVALID_READ_OF_DIRECTORY, {
            absolutePath,
          });
        }

        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw e;
      }
    }
  );

/**
 * DEPRECATED: This subtask is deprecated and will be removed in the future.
 *
 * This task transform the string literal in an import directive.
 * By default it does nothing, but it can be overriden by plugins.
 */
subtask(TASK_COMPILE_TRANSFORM_IMPORT_NAME)
  .addParam("importName", undefined, undefined, types.string)
  .setAction(
    async ({ importName }: { importName: string }): Promise<string> => {
      return importName;
    }
  );

/**
 * This task returns a Record<string, string> representing remappings to be used
 * by the resolver.
 */
subtask(TASK_COMPILE_GET_REMAPPINGS).setAction(
  async (): Promise<Record<string, string>> => {
    return {};
  }
);

/**
 * Receives a list of source names and returns a dependency graph. This task
 * is responsible for both resolving dependencies (like getting files from
 * node_modules) and generating the graph.
 */
subtask(TASK_COMPILE_SOLIDITY_GET_DEPENDENCY_GRAPH)
  .addOptionalParam("rootPath", undefined, undefined, types.string)
  .addParam("sourceNames", undefined, undefined, types.any)
  .addOptionalParam("solidityFilesCache", undefined, undefined, types.any)
  .setAction(
    async (
      {
        rootPath,
        sourceNames,
        solidityFilesCache,
      }: {
        rootPath?: string;
        sourceNames: string[];
        solidityFilesCache?: SolidityFilesCache;
      },
      { config, run }
    ): Promise<taskTypes.DependencyGraph> => {
      const parser = new Parser(solidityFilesCache);
      const remappings = await run(TASK_COMPILE_GET_REMAPPINGS);
      const resolver = new Resolver(
        rootPath ?? config.paths.root,
        parser,
        remappings,
        (absolutePath: string) =>
          run(TASK_COMPILE_SOLIDITY_READ_FILE, { absolutePath }),
        (importName: string) =>
          run(TASK_COMPILE_TRANSFORM_IMPORT_NAME, {
            importName,
            deprecationCheck: true,
          })
      );

      const resolvedFiles = await Promise.all(
        sourceNames.map((sn) => resolver.resolveSourceName(sn))
      );

      return DependencyGraph.createFromResolvedFiles(resolver, resolvedFiles);
    }
  );

/**
 * Receives a dependency graph and a file in it, and returns the compilation
 * job for that file. The compilation job should have everything that is
 * necessary to compile that file: a compiler config to be used and a list of
 * files to use as input of the compilation.
 *
 * If the file cannot be compiled, a MatchingCompilerFailure should be
 * returned instead.
 *
 * This is the right task to override to change the compiler configuration.
 * For example, if you want to change the compiler settings when targetting
 * goerli, you could do something like this:
 *
 *   const compilationJob = await runSuper();
 *   if (config.network.name === 'goerli') {
 *     compilationJob.solidityConfig.settings = newSettings;
 *   }
 *   return compilationJob;
 *
 */
subtask(TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOB_FOR_FILE)
  .addParam("dependencyGraph", undefined, undefined, types.any)
  .addParam("file", undefined, undefined, types.any)
  .addOptionalParam("solidityFilesCache", undefined, undefined, types.any)
  .setAction(
    async (
      {
        dependencyGraph,
        file,
      }: {
        dependencyGraph: taskTypes.DependencyGraph;
        file: taskTypes.ResolvedFile;
        solidityFilesCache?: SolidityFilesCache;
      },
      { config }
    ): Promise<CompilationJob | CompilationJobCreationError> => {
      return createCompilationJobFromFile(
        dependencyGraph,
        file,
        config.solidity
      );
    }
  );

/**
 * Receives a dependency graph and returns a tuple with two arrays. The first
 * array is a list of CompilationJobsSuccess, where each item has a list of
 * compilation jobs. The second array is a list of CompilationJobsFailure,
 * where each item has a list of files that couldn't be compiled, grouped by
 * the reason for the failure.
 */
subtask(TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOBS)
  .addParam("dependencyGraph", undefined, undefined, types.any)
  .addOptionalParam("solidityFilesCache", undefined, undefined, types.any)
  .setAction(
    async (
      {
        dependencyGraph,
        solidityFilesCache,
      }: {
        dependencyGraph: taskTypes.DependencyGraph;
        solidityFilesCache?: SolidityFilesCache;
      },
      { run }
    ): Promise<CompilationJobsCreationResult> => {
      const connectedComponents = dependencyGraph.getConnectedComponents();

      log(
        `The dependency graph was divided in '${connectedComponents.length}' connected components`
      );

      const compilationJobsCreationResults = await Promise.all(
        connectedComponents.map((graph) =>
          createCompilationJobsFromConnectedComponent(
            graph,
            (file: taskTypes.ResolvedFile) =>
              run(TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOB_FOR_FILE, {
                file,
                dependencyGraph,
                solidityFilesCache,
              })
          )
        )
      );

      let jobs: CompilationJob[] = [];
      let errors: CompilationJobCreationError[] = [];

      for (const result of compilationJobsCreationResults) {
        jobs = jobs.concat(result.jobs);
        errors = errors.concat(result.errors);
      }

      return { jobs, errors };
    }
  );

/**
 * Receives a list of compilation jobs and returns a new list where some of
 * the compilation jobs might've been removed.
 *
 * This task can be overridden to change the way the cache is used, or to use
 * a different approach to filtering out compilation jobs.
 */
subtask(TASK_COMPILE_SOLIDITY_FILTER_COMPILATION_JOBS)
  .addParam("compilationJobs", undefined, undefined, types.any)
  .addParam("force", undefined, undefined, types.boolean)
  .addOptionalParam("solidityFilesCache", undefined, undefined, types.any)
  .setAction(
    async ({
      compilationJobs,
      force,
      solidityFilesCache,
    }: {
      compilationJobs: CompilationJob[];
      force: boolean;
      solidityFilesCache?: SolidityFilesCache;
    }): Promise<CompilationJob[]> => {
      assertHardhatInvariant(
        solidityFilesCache !== undefined,
        "The implementation of this task needs a defined solidityFilesCache"
      );

      if (force) {
        log(`force flag enabled, not filtering`);
        return compilationJobs;
      }

      const neededCompilationJobs = compilationJobs.filter((job) =>
        needsCompilation(job, solidityFilesCache)
      );

      const jobsFilteredOutCount =
        compilationJobs.length - neededCompilationJobs.length;
      log(`'${jobsFilteredOutCount}' jobs were filtered out`);

      return neededCompilationJobs;
    }
  );

/**
 * Receives a list of compilation jobs and returns a new list where some of
 * the jobs might've been merged.
 */
subtask(TASK_COMPILE_SOLIDITY_MERGE_COMPILATION_JOBS)
  .addParam("compilationJobs", undefined, undefined, types.any)
  .setAction(
    async ({
      compilationJobs,
    }: {
      compilationJobs: CompilationJob[];
    }): Promise<CompilationJob[]> => {
      return mergeCompilationJobsWithoutBug(compilationJobs);
    }
  );

/**
 * Prints a message when there's nothing to compile.
 */
subtask(TASK_COMPILE_SOLIDITY_LOG_NOTHING_TO_COMPILE)
  .addParam("quiet", undefined, undefined, types.boolean)
  .setAction(async ({ quiet }: { quiet: boolean }) => {
    if (!quiet) {
      console.log("Nothing to compile");
    }
  });

/**
 * Receives a list of compilation jobs and sends each one to be compiled.
 */
subtask(TASK_COMPILE_SOLIDITY_COMPILE_JOBS)
  .addParam("compilationJobs", undefined, undefined, types.any)
  .addParam("quiet", undefined, undefined, types.boolean)
  .addParam("concurrency", undefined, DEFAULT_CONCURRENCY_LEVEL, types.int)
  .setAction(
    async (
      {
        compilationJobs,
        quiet,
        concurrency,
      }: {
        compilationJobs: CompilationJob[];
        quiet: boolean;
        concurrency: number;
      },
      { run }
    ): Promise<{ artifactsEmittedPerJob: ArtifactsEmittedPerJob }> => {
      if (compilationJobs.length === 0) {
        log(`No compilation jobs to compile`);
        await run(TASK_COMPILE_SOLIDITY_LOG_NOTHING_TO_COMPILE, { quiet });
        return { artifactsEmittedPerJob: [] };
      }

      log(`Compiling ${compilationJobs.length} jobs`);

      for (const job of compilationJobs) {
        const solcVersion = job.getSolcConfig().version;

        // versions older than 0.4.11 don't work with hardhat
        // see issue https://github.com/nomiclabs/hardhat/issues/2004
        if (semver.lt(solcVersion, COMPILE_TASK_FIRST_SOLC_VERSION_SUPPORTED)) {
          throw new HardhatError(
            ERRORS.BUILTIN_TASKS.COMPILE_TASK_UNSUPPORTED_SOLC_VERSION,
            {
              version: solcVersion,
              firstSupportedVersion: COMPILE_TASK_FIRST_SOLC_VERSION_SUPPORTED,
            }
          );
        }
      }

      const { default: pMap } = await import("p-map");
      const pMapOptions = { concurrency, stopOnError: false };
      try {
        const artifactsEmittedPerJob: ArtifactsEmittedPerJob = await pMap(
          compilationJobs,
          async (compilationJob, compilationJobIndex) => {
            const result = await run(TASK_COMPILE_SOLIDITY_COMPILE_JOB, {
              compilationJob,
              compilationJobs,
              compilationJobIndex,
              quiet,
            });

            return {
              compilationJob: result.compilationJob,
              artifactsEmittedPerFile: result.artifactsEmittedPerFile,
            };
          },
          pMapOptions
        );

        return { artifactsEmittedPerJob };
      } catch (e) {
        if (!(e instanceof AggregateError)) {
          // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
          throw e;
        }

        for (const error of e) {
          if (
            !HardhatError.isHardhatErrorType(
              error,
              ERRORS.BUILTIN_TASKS.COMPILE_FAILURE
            )
          ) {
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw error;
          }
        }

        // error is an aggregate error, and all errors are compilation failures
        throw new HardhatError(ERRORS.BUILTIN_TASKS.COMPILE_FAILURE);
      }
    }
  );

/**
 * Receives a compilation job and returns a CompilerInput.
 *
 * It's not recommended to override this task to modify the solc
 * configuration, override
 * TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOB_FOR_FILE instead.
 */
subtask(TASK_COMPILE_SOLIDITY_GET_COMPILER_INPUT)
  .addParam("compilationJob", undefined, undefined, types.any)
  .setAction(
    async ({
      compilationJob,
    }: {
      compilationJob: CompilationJob;
    }): Promise<CompilerInput> => {
      return getInputFromCompilationJob(compilationJob);
    }
  );

subtask(TASK_COMPILE_SOLIDITY_LOG_DOWNLOAD_COMPILER_START)
  .addParam("isCompilerDownloaded", undefined, undefined, types.boolean)
  .addParam("quiet", undefined, undefined, types.boolean)
  .addParam("solcVersion", undefined, undefined, types.string)
  .setAction(
    async ({
      isCompilerDownloaded,
      solcVersion,
    }: {
      isCompilerDownloaded: boolean;
      quiet: boolean;
      solcVersion: string;
    }) => {
      if (isCompilerDownloaded) {
        return;
      }

      console.log(`Downloading compiler ${solcVersion}`);
    }
  );

subtask(TASK_COMPILE_SOLIDITY_LOG_DOWNLOAD_COMPILER_END)
  .addParam("isCompilerDownloaded", undefined, undefined, types.boolean)
  .addParam("quiet", undefined, undefined, types.boolean)
  .addParam("solcVersion", undefined, undefined, types.string)
  .setAction(
    async ({}: {
      isCompilerDownloaded: boolean;
      quiet: boolean;
      solcVersion: string;
    }) => {}
  );

/**
 * Receives a solc version and returns a path to a solc binary or to a
 * downloaded solcjs module. It also returns a flag indicating if the returned
 * path corresponds to solc or solcjs.
 */
subtask(TASK_COMPILE_SOLIDITY_GET_SOLC_BUILD)
  .addParam("quiet", undefined, undefined, types.boolean)
  .addParam("solcVersion", undefined, undefined, types.string)
  .setAction(
    async (
      {
        quiet,
        solcVersion,
      }: {
        quiet: boolean;
        solcVersion: string;
      },
      { run }
    ): Promise<SolcBuild> => {
      const compilersCache = await getCompilersDir();

      const compilerPlatform = CompilerDownloader.getCompilerPlatform();
      const downloader = CompilerDownloader.getConcurrencySafeDownloader(
        compilerPlatform,
        compilersCache
      );

      await downloader.downloadCompiler(
        solcVersion,
        // callback called before compiler download
        async (isCompilerDownloaded: boolean) => {
          await run(TASK_COMPILE_SOLIDITY_LOG_DOWNLOAD_COMPILER_START, {
            solcVersion,
            isCompilerDownloaded,
            quiet,
          });
        },
        // callback called after compiler download
        async (isCompilerDownloaded: boolean) => {
          await run(TASK_COMPILE_SOLIDITY_LOG_DOWNLOAD_COMPILER_END, {
            solcVersion,
            isCompilerDownloaded,
            quiet,
          });
        }
      );

      const compiler = await downloader.getCompiler(solcVersion);

      if (compiler !== undefined) {
        return compiler;
      }

      log(
        "Native solc binary doesn't work, using solcjs instead. Try running npx hardhat clean --global"
      );

      const wasmDownloader = CompilerDownloader.getConcurrencySafeDownloader(
        CompilerPlatform.WASM,
        compilersCache
      );

      await wasmDownloader.downloadCompiler(
        solcVersion,
        async (isCompilerDownloaded: boolean) => {
          // callback called before compiler download
          await run(TASK_COMPILE_SOLIDITY_LOG_DOWNLOAD_COMPILER_START, {
            solcVersion,
            isCompilerDownloaded,
            quiet,
          });
        },
        // callback called after compiler download
        async (isCompilerDownloaded: boolean) => {
          await run(TASK_COMPILE_SOLIDITY_LOG_DOWNLOAD_COMPILER_END, {
            solcVersion,
            isCompilerDownloaded,
            quiet,
          });
        }
      );

      const wasmCompiler = await wasmDownloader.getCompiler(solcVersion);

      assertHardhatInvariant(
        wasmCompiler !== undefined,
        `WASM build of solc ${solcVersion} isn't working`
      );

      return wasmCompiler;
    }
  );

/**
 * Receives an absolute path to a solcjs module and the input to be compiled,
 * and returns the generated output
 */
subtask(TASK_COMPILE_SOLIDITY_RUN_SOLCJS)
  .addParam("input", undefined, undefined, types.any)
  .addParam("solcJsPath", undefined, undefined, types.string)
  .setAction(
    async ({
      input,
      solcJsPath,
    }: {
      input: CompilerInput;
      solcJsPath: string;
    }) => {
      const compiler = new Compiler(solcJsPath);

      return compiler.compile(input);
    }
  );

/**
 * Receives an absolute path to a solc binary and the input to be compiled,
 * and returns the generated output
 */
subtask(TASK_COMPILE_SOLIDITY_RUN_SOLC)
  .addParam("input", undefined, undefined, types.any)
  .addParam("solcPath", undefined, undefined, types.string)
  .addOptionalParam("solcVersion", undefined, undefined, types.string)
  .setAction(
    async ({
      input,
      solcPath,
      solcVersion,
    }: {
      input: CompilerInput;
      solcPath: string;
      solcVersion?: string;
    }) => {
      if (solcVersion !== undefined && semver.valid(solcVersion) === null) {
        throw new HardhatError(ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
          value: solcVersion,
          name: "solcVersion",
          type: "string",
        });
      }

      const compiler = new NativeCompiler(solcPath, solcVersion);

      return compiler.compile(input);
    }
  );

/**
 * Receives a CompilerInput and a solc version, compiles the input using a native
 * solc binary or, if that's not possible, using solcjs. Returns the generated
 * output.
 *
 * This task can be overriden to change how solc is obtained or used.
 */
subtask(TASK_COMPILE_SOLIDITY_COMPILE_SOLC)
  .addParam("input", undefined, undefined, types.any)
  .addParam("quiet", undefined, undefined, types.boolean)
  .addParam("solcVersion", undefined, undefined, types.string)
  .addParam("compilationJob", undefined, undefined, types.any)
  .addParam("compilationJobs", undefined, undefined, types.any)
  .addParam("compilationJobIndex", undefined, undefined, types.int)
  .setAction(
    async (
      {
        input,
        quiet,
        solcVersion,
        compilationJob,
        compilationJobs,
        compilationJobIndex,
      }: {
        input: CompilerInput;
        quiet: boolean;
        solcVersion: string;
        compilationJob: CompilationJob;
        compilationJobs: CompilationJob[];
        compilationJobIndex: number;
      },
      { run }
    ): Promise<{ output: CompilerOutput; solcBuild: SolcBuild }> => {
      const solcBuild: SolcBuild = await run(
        TASK_COMPILE_SOLIDITY_GET_SOLC_BUILD,
        {
          quiet,
          solcVersion,
        }
      );

      await run(TASK_COMPILE_SOLIDITY_LOG_RUN_COMPILER_START, {
        compilationJob,
        compilationJobs,
        compilationJobIndex,
        quiet,
      });

      let output;
      if (solcBuild.isSolcJs) {
        output = await run(TASK_COMPILE_SOLIDITY_RUN_SOLCJS, {
          input,
          solcJsPath: solcBuild.compilerPath,
        });
      } else {
        output = await run(TASK_COMPILE_SOLIDITY_RUN_SOLC, {
          input,
          solcPath: solcBuild.compilerPath,
          solcVersion,
        });
      }

      await run(TASK_COMPILE_SOLIDITY_LOG_RUN_COMPILER_END, {
        compilationJob,
        compilationJobs,
        compilationJobIndex,
        output,
        quiet,
      });

      return { output, solcBuild };
    }
  );

/**
 * This task is just a proxy to the task that compiles with solc.
 *
 * Override this to use a different task to compile a job.
 */
subtask(TASK_COMPILE_SOLIDITY_COMPILE, async (taskArgs: any, { run }) => {
  return run(TASK_COMPILE_SOLIDITY_COMPILE_SOLC, taskArgs);
});

/**
 * Receives a compilation output and prints its errors and any other
 * information useful to the user.
 */
subtask(TASK_COMPILE_SOLIDITY_LOG_COMPILATION_ERRORS)
  .addParam("output", undefined, undefined, types.any)
  .addParam("quiet", undefined, undefined, types.boolean)
  .setAction(async ({ output }: { output: any; quiet: boolean }) => {
    if (output?.errors === undefined) {
      return;
    }

    for (const error of output.errors) {
      if (error.severity === "error") {
        const errorMessage: string =
          getFormattedInternalCompilerErrorMessage(error) ??
          error.formattedMessage;

        console.error(errorMessage.replace(/^\w+:/, (t) => chalk.red.bold(t)));
      } else {
        console.warn(
          (error.formattedMessage as string).replace(/^\w+:/, (t) =>
            chalk.yellow.bold(t)
          )
        );
      }
    }

    const hasConsoleErrors: boolean = output.errors.some(isConsoleLogError);
    if (hasConsoleErrors) {
      console.error(
        chalk.red(
          `The console.log call you made isn’t supported. See https://hardhat.org/console-log for the list of supported methods.`
        )
      );
      console.log();
    }
  });

/**
 * Receives a solc output and checks if there are errors. Throws if there are
 * errors.
 *
 * Override this task to avoid interrupting the compilation process if some
 * job has compilation errors.
 */
subtask(TASK_COMPILE_SOLIDITY_CHECK_ERRORS)
  .addParam("output", undefined, undefined, types.any)
  .addParam("quiet", undefined, undefined, types.boolean)
  .setAction(
    async ({ output, quiet }: { output: any; quiet: boolean }, { run }) => {
      await run(TASK_COMPILE_SOLIDITY_LOG_COMPILATION_ERRORS, {
        output,
        quiet,
      });

      if (hasCompilationErrors(output)) {
        throw new HardhatError(ERRORS.BUILTIN_TASKS.COMPILE_FAILURE);
      }
    }
  );

/**
 * Saves to disk the artifacts for a compilation job. These artifacts
 * include the main artifacts, the debug files, and the build info.
 */
subtask(TASK_COMPILE_SOLIDITY_EMIT_ARTIFACTS)
  .addParam("compilationJob", undefined, undefined, types.any)
  .addParam("input", undefined, undefined, types.any)
  .addParam("output", undefined, undefined, types.any)
  .addParam("solcBuild", undefined, undefined, types.any)
  .setAction(
    async (
      {
        compilationJob,
        input,
        output,
        solcBuild,
      }: {
        compilationJob: CompilationJob;
        input: CompilerInput;
        output: CompilerOutput;
        solcBuild: SolcBuild;
      },
      { artifacts, run }
    ): Promise<{
      artifactsEmittedPerFile: taskTypes.ArtifactsEmittedPerFile;
    }> => {
      const pathToBuildInfo = await artifacts.saveBuildInfo(
        compilationJob.getSolcConfig().version,
        solcBuild.longVersion,
        input,
        output
      );

      const artifactsEmittedPerFile: taskTypes.ArtifactsEmittedPerFile =
        await Promise.all(
          compilationJob
            .getResolvedFiles()
            .filter((f) => compilationJob.emitsArtifacts(f))
            .map(async (file) => {
              const artifactsEmitted = await Promise.all(
                Object.entries(output.contracts?.[file.sourceName] ?? {}).map(
                  async ([contractName, contractOutput]) => {
                    log(`Emitting artifact for contract '${contractName}'`);
                    const artifact = await run(
                      TASK_COMPILE_SOLIDITY_GET_ARTIFACT_FROM_COMPILATION_OUTPUT,
                      {
                        sourceName: file.sourceName,
                        contractName,
                        contractOutput,
                      }
                    );

                    await artifacts.saveArtifactAndDebugFile(
                      artifact,
                      pathToBuildInfo
                    );

                    return artifact.contractName;
                  }
                )
              );

              return {
                file,
                artifactsEmitted,
              };
            })
        );

      return { artifactsEmittedPerFile };
    }
  );

/**
 * Generates the artifact for contract `contractName` given its compilation
 * output.
 */
subtask(TASK_COMPILE_SOLIDITY_GET_ARTIFACT_FROM_COMPILATION_OUTPUT)
  .addParam("sourceName", undefined, undefined, types.string)
  .addParam("contractName", undefined, undefined, types.string)
  .addParam("contractOutput", undefined, undefined, types.any)
  .setAction(
    async ({
      sourceName,
      contractName,
      contractOutput,
    }: {
      sourceName: string;
      contractName: string;
      contractOutput: any;
    }): Promise<any> => {
      return getArtifactFromContractOutput(
        sourceName,
        contractName,
        contractOutput
      );
    }
  );

/**
 * Prints a message before running soljs with some input.
 */
subtask(TASK_COMPILE_SOLIDITY_LOG_RUN_COMPILER_START)
  .addParam("compilationJob", undefined, undefined, types.any)
  .addParam("compilationJobs", undefined, undefined, types.any)
  .addParam("compilationJobIndex", undefined, undefined, types.int)
  .addParam("quiet", undefined, undefined, types.boolean)
  .setAction(
    async ({}: {
      compilationJob: CompilationJob;
      compilationJobs: CompilationJob[];
      compilationJobIndex: number;
    }) => {}
  );

/**
 * Prints a message after compiling some input
 */
subtask(TASK_COMPILE_SOLIDITY_LOG_RUN_COMPILER_END)
  .addParam("compilationJob", undefined, undefined, types.any)
  .addParam("compilationJobs", undefined, undefined, types.any)
  .addParam("compilationJobIndex", undefined, undefined, types.int)
  .addParam("output", undefined, undefined, types.any)
  .addParam("quiet", undefined, undefined, types.boolean)
  .setAction(
    async ({}: {
      compilationJob: CompilationJob;
      compilationJobs: CompilationJob[];
      compilationJobIndex: number;
      output: any;
      quiet: boolean;
    }) => {}
  );

/**
 * This is an orchestrator task that uses other subtasks to compile a
 * compilation job.
 */
subtask(TASK_COMPILE_SOLIDITY_COMPILE_JOB)
  .addParam("compilationJob", undefined, undefined, types.any)
  .addParam("compilationJobs", undefined, undefined, types.any)
  .addParam("compilationJobIndex", undefined, undefined, types.int)
  .addParam("quiet", undefined, undefined, types.boolean)
  .addOptionalParam("emitsArtifacts", undefined, true, types.boolean)
  .setAction(
    async (
      {
        compilationJob,
        compilationJobs,
        compilationJobIndex,
        quiet,
        emitsArtifacts,
      }: {
        compilationJob: CompilationJob;
        compilationJobs: CompilationJob[];
        compilationJobIndex: number;
        quiet: boolean;
        emitsArtifacts: boolean;
      },
      { run }
    ): Promise<{
      artifactsEmittedPerFile: taskTypes.ArtifactsEmittedPerFile;
      compilationJob: taskTypes.CompilationJob;
      input: CompilerInput;
      output: CompilerOutput;
      solcBuild: any;
    }> => {
      log(
        `Compiling job with version '${compilationJob.getSolcConfig().version}'`
      );
      const input: CompilerInput = await run(
        TASK_COMPILE_SOLIDITY_GET_COMPILER_INPUT,
        {
          compilationJob,
        }
      );

      const { output, solcBuild } = await run(TASK_COMPILE_SOLIDITY_COMPILE, {
        solcVersion: compilationJob.getSolcConfig().version,
        input,
        quiet,
        compilationJob,
        compilationJobs,
        compilationJobIndex,
      });

      await run(TASK_COMPILE_SOLIDITY_CHECK_ERRORS, { output, quiet });

      let artifactsEmittedPerFile = [];
      if (emitsArtifacts) {
        artifactsEmittedPerFile = (
          await run(TASK_COMPILE_SOLIDITY_EMIT_ARTIFACTS, {
            compilationJob,
            input,
            output,
            solcBuild,
          })
        ).artifactsEmittedPerFile;
      }

      return {
        artifactsEmittedPerFile,
        compilationJob,
        input,
        output,
        solcBuild,
      };
    }
  );

/**
 * Receives a list of CompilationJobsFailure and throws an error if it's not
 * empty.
 *
 * This task could be overriden to avoid interrupting the compilation if
 * there's some part of the project that can't be compiled.
 */
subtask(TASK_COMPILE_SOLIDITY_HANDLE_COMPILATION_JOBS_FAILURES)
  .addParam("compilationJobsCreationErrors", undefined, undefined, types.any)
  .setAction(
    async (
      {
        compilationJobsCreationErrors,
      }: {
        compilationJobsCreationErrors: CompilationJobCreationError[];
      },
      { run }
    ) => {
      const hasErrors = compilationJobsCreationErrors.length > 0;

      if (hasErrors) {
        log(`There were errors creating the compilation jobs, throwing`);
        const reasons: string = await run(
          TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOBS_FAILURE_REASONS,
          { compilationJobsCreationErrors }
        );

        throw new HardhatError(
          ERRORS.BUILTIN_TASKS.COMPILATION_JOBS_CREATION_FAILURE,
          {
            reasons,
          }
        );
      }
    }
  );

/**
 * Receives a list of CompilationJobsFailure and returns an error message
 * that describes the failure.
 */
subtask(TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOBS_FAILURE_REASONS)
  .addParam("compilationJobsCreationErrors", undefined, undefined, types.any)
  .setAction(
    async ({
      compilationJobsCreationErrors: errors,
    }: {
      compilationJobsCreationErrors: CompilationJobCreationError[];
    }): Promise<string> => {
      const noCompatibleSolc: CompilationJobCreationError[] = [];
      const incompatibleOverridenSolc: CompilationJobCreationError[] = [];
      const directlyImportsIncompatibleFile: CompilationJobCreationError[] = [];
      const indirectlyImportsIncompatibleFile: CompilationJobCreationError[] =
        [];
      const other: CompilationJobCreationError[] = [];

      for (const error of errors) {
        if (
          error.reason ===
          CompilationJobCreationErrorReason.NO_COMPATIBLE_SOLC_VERSION_FOUND
        ) {
          noCompatibleSolc.push(error);
        } else if (
          error.reason ===
          CompilationJobCreationErrorReason.INCOMPATIBLE_OVERRIDEN_SOLC_VERSION
        ) {
          incompatibleOverridenSolc.push(error);
        } else if (
          error.reason ===
          CompilationJobCreationErrorReason.DIRECTLY_IMPORTS_INCOMPATIBLE_FILE
        ) {
          directlyImportsIncompatibleFile.push(error);
        } else if (
          error.reason ===
          CompilationJobCreationErrorReason.INDIRECTLY_IMPORTS_INCOMPATIBLE_FILE
        ) {
          indirectlyImportsIncompatibleFile.push(error);
        } else if (
          error.reason === CompilationJobCreationErrorReason.OTHER_ERROR
        ) {
          other.push(error);
        } else {
          // add unrecognized errors to `other`
          other.push(error);
        }
      }

      let errorMessage = "";
      if (incompatibleOverridenSolc.length > 0) {
        errorMessage += `The compiler version for the following files is fixed through an override in your config file to a version that is incompatible with their Solidity version pragmas.

`;

        for (const error of incompatibleOverridenSolc) {
          const { sourceName } = error.file;
          const { versionPragmas } = error.file.content;
          const versionsRange = versionPragmas.join(" ");

          log(`File ${sourceName} has an incompatible overridden compiler`);

          errorMessage += `  * ${sourceName} (${versionsRange})\n`;
        }

        errorMessage += "\n";
      }

      if (noCompatibleSolc.length > 0) {
        errorMessage += `The Solidity version pragma statement in these files doesn't match any of the configured compilers in your config. Change the pragma or configure additional compiler versions in your hardhat config.

`;

        for (const error of noCompatibleSolc) {
          const { sourceName } = error.file;
          const { versionPragmas } = error.file.content;
          const versionsRange = versionPragmas.join(" ");

          log(
            `File ${sourceName} doesn't match any of the configured compilers`
          );

          errorMessage += `  * ${sourceName} (${versionsRange})\n`;
        }

        errorMessage += "\n";
      }

      if (directlyImportsIncompatibleFile.length > 0) {
        errorMessage += `These files import other files that use a different and incompatible version of Solidity:

`;

        for (const error of directlyImportsIncompatibleFile) {
          const { sourceName } = error.file;
          const { versionPragmas } = error.file.content;
          const versionsRange = versionPragmas.join(" ");

          const incompatibleDirectImportsFiles: ResolvedFile[] =
            error.extra?.incompatibleDirectImports ?? [];

          const incompatibleDirectImports = incompatibleDirectImportsFiles.map(
            (x: ResolvedFile) =>
              `${x.sourceName} (${x.content.versionPragmas.join(" ")})`
          );

          log(
            `File ${sourceName} imports files ${incompatibleDirectImportsFiles
              .map((x) => x.sourceName)
              .join(", ")} that use an incompatible version of Solidity`
          );

          let directImportsText = "";
          if (incompatibleDirectImports.length === 1) {
            directImportsText = ` imports ${incompatibleDirectImports[0]}`;
          } else if (incompatibleDirectImports.length === 2) {
            directImportsText = ` imports ${incompatibleDirectImports[0]} and ${incompatibleDirectImports[1]}`;
          } else if (incompatibleDirectImports.length > 2) {
            const otherImportsCount = incompatibleDirectImports.length - 2;
            directImportsText = ` imports ${incompatibleDirectImports[0]}, ${
              incompatibleDirectImports[1]
            } and ${otherImportsCount} other ${pluralize(
              otherImportsCount,
              "file"
            )}. Use --verbose to see the full list.`;
          }

          errorMessage += `  * ${sourceName} (${versionsRange})${directImportsText}\n`;
        }

        errorMessage += "\n";
      }

      if (indirectlyImportsIncompatibleFile.length > 0) {
        errorMessage += `These files depend on other files that use a different and incompatible version of Solidity:

`;

        for (const error of indirectlyImportsIncompatibleFile) {
          const { sourceName } = error.file;
          const { versionPragmas } = error.file.content;
          const versionsRange = versionPragmas.join(" ");

          const incompatibleIndirectImports: taskTypes.TransitiveDependency[] =
            error.extra?.incompatibleIndirectImports ?? [];

          const incompatibleImports = incompatibleIndirectImports.map(
            ({ dependency }) =>
              `${
                dependency.sourceName
              } (${dependency.content.versionPragmas.join(" ")})`
          );

          for (const {
            dependency,
            path: dependencyPath,
          } of incompatibleIndirectImports) {
            const dependencyPathText = [
              sourceName,
              ...dependencyPath.map((x) => x.sourceName),
              dependency.sourceName,
            ].join(" -> ");

            log(
              `File ${sourceName} depends on file ${dependency.sourceName} that uses an incompatible version of Solidity
The dependency path is ${dependencyPathText}
`
            );
          }

          let indirectImportsText = "";
          if (incompatibleImports.length === 1) {
            indirectImportsText = ` depends on ${incompatibleImports[0]}`;
          } else if (incompatibleImports.length === 2) {
            indirectImportsText = ` depends on ${incompatibleImports[0]} and ${incompatibleImports[1]}`;
          } else if (incompatibleImports.length > 2) {
            const otherImportsCount = incompatibleImports.length - 2;
            indirectImportsText = ` depends on ${incompatibleImports[0]}, ${
              incompatibleImports[1]
            } and ${otherImportsCount} other ${pluralize(
              otherImportsCount,
              "file"
            )}. Use --verbose to see the full list.`;
          }

          errorMessage += `  * ${sourceName} (${versionsRange})${indirectImportsText}\n`;
        }

        errorMessage += "\n";
      }

      if (other.length > 0) {
        errorMessage += `These files and its dependencies cannot be compiled with your config. This can happen because they have incompatible Solidity pragmas, or don't match any of your configured Solidity compilers.

${other.map((x) => `  * ${x.file.sourceName}`).join("\n")}

`;
      }

      errorMessage += `To learn more, run the command again with --verbose

Read about compiler configuration at https://hardhat.org/config
`;

      return errorMessage;
    }
  );

subtask(TASK_COMPILE_SOLIDITY_LOG_COMPILATION_RESULT)
  .addParam("compilationJobs", undefined, undefined, types.any)
  .addParam("quiet", undefined, undefined, types.boolean)
  .setAction(
    async ({ compilationJobs }: { compilationJobs: CompilationJob[] }) => {
      let count = 0;
      const evmVersions = new Set<string>();
      const unknownEvmVersions = new Set<string>();

      for (const job of compilationJobs) {
        count += job
          .getResolvedFiles()
          .filter((file) => job.emitsArtifacts(file)).length;

        const solcVersion = job.getSolcConfig().version;
        const evmTarget =
          job.getSolcConfig().settings?.evmVersion ??
          getEvmVersionFromSolcVersion(solcVersion);

        if (evmTarget !== undefined) {
          evmVersions.add(evmTarget);
        } else {
          unknownEvmVersions.add(
            `unknown evm version for solc version ${solcVersion}`
          );
        }
      }

      const targetVersionsList = Array.from(evmVersions)
        // Alphabetically sort evm versions. The unknown ones are added at the end
        .sort()
        .concat(Array.from(unknownEvmVersions).sort());

      if (count > 0) {
        console.log(
          `Compiled ${count} Solidity ${pluralize(
            count,
            "file"
          )} successfully (evm ${pluralize(
            targetVersionsList.length,
            "target",
            "targets"
          )}: ${targetVersionsList.join(", ")}).`
        );
      }
    }
  );

/**
 * Main task for compiling the solidity files in the project.
 *
 * The main responsibility of this task is to orchestrate and connect most of
 * the subtasks related to compiling solidity.
 */
subtask(TASK_COMPILE_SOLIDITY)
  .addParam("force", undefined, undefined, types.boolean)
  .addParam("quiet", undefined, undefined, types.boolean)
  .addParam("concurrency", undefined, DEFAULT_CONCURRENCY_LEVEL, types.int)
  .setAction(
    async (
      {
        force,
        quiet,
        concurrency,
      }: { force: boolean; quiet: boolean; concurrency: number },
      { artifacts, config, run }
    ) => {
      const rootPath = config.paths.root;

      const sourcePaths: string[] = await run(
        TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS,
        { sourcePath: config.paths.sources }
      );
      const sourceNames: string[] = await run(
        TASK_COMPILE_SOLIDITY_GET_SOURCE_NAMES,
        {
          rootPath,
          sourcePaths,
        }
      );

      const solidityFilesCachePath = getSolidityFilesCachePath(config.paths);
      let solidityFilesCache = await SolidityFilesCache.readFromFile(
        solidityFilesCachePath
      );

      const dependencyGraph: taskTypes.DependencyGraph = await run(
        TASK_COMPILE_SOLIDITY_GET_DEPENDENCY_GRAPH,
        { rootPath, sourceNames, solidityFilesCache }
      );

      solidityFilesCache = await invalidateCacheMissingArtifacts(
        solidityFilesCache,
        artifacts,
        dependencyGraph.getResolvedFiles()
      );

      const compilationJobsCreationResult: CompilationJobsCreationResult =
        await run(TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOBS, {
          dependencyGraph,
          solidityFilesCache,
        });

      await run(TASK_COMPILE_SOLIDITY_HANDLE_COMPILATION_JOBS_FAILURES, {
        compilationJobsCreationErrors: compilationJobsCreationResult.errors,
      });

      const compilationJobs = compilationJobsCreationResult.jobs;

      const filteredCompilationJobs: CompilationJob[] = await run(
        TASK_COMPILE_SOLIDITY_FILTER_COMPILATION_JOBS,
        { compilationJobs, force, solidityFilesCache }
      );

      const mergedCompilationJobs: CompilationJob[] = await run(
        TASK_COMPILE_SOLIDITY_MERGE_COMPILATION_JOBS,
        { compilationJobs: filteredCompilationJobs }
      );

      const {
        artifactsEmittedPerJob,
      }: { artifactsEmittedPerJob: ArtifactsEmittedPerJob } = await run(
        TASK_COMPILE_SOLIDITY_COMPILE_JOBS,
        {
          compilationJobs: mergedCompilationJobs,
          quiet,
          concurrency,
        }
      );

      // update cache using the information about the emitted artifacts
      for (const {
        compilationJob: compilationJob,
        artifactsEmittedPerFile: artifactsEmittedPerFile,
      } of artifactsEmittedPerJob) {
        for (const { file, artifactsEmitted } of artifactsEmittedPerFile) {
          solidityFilesCache.addFile(file.absolutePath, {
            lastModificationDate: file.lastModificationDate.valueOf(),
            contentHash: file.contentHash,
            sourceName: file.sourceName,
            solcConfig: compilationJob.getSolcConfig(),
            imports: file.content.imports,
            versionPragmas: file.content.versionPragmas,
            artifacts: artifactsEmitted,
          });
        }
      }

      const allArtifactsEmittedPerFile = solidityFilesCache.getEntries();

      // We know this is the actual implementation, so we use some
      // non-public methods here.
      const artifactsImpl = artifacts as ArtifactsImpl;
      artifactsImpl.addValidArtifacts(allArtifactsEmittedPerFile);

      await solidityFilesCache.writeToFile(solidityFilesCachePath);

      await run(TASK_COMPILE_SOLIDITY_LOG_COMPILATION_RESULT, {
        compilationJobs: mergedCompilationJobs,
        quiet,
      });
    }
  );

subtask(TASK_COMPILE_REMOVE_OBSOLETE_ARTIFACTS, async (_, { artifacts }) => {
  // We know this is the actual implementation, so we use some
  // non-public methods here.
  const artifactsImpl = artifacts as ArtifactsImpl;
  await artifactsImpl.removeObsoleteArtifacts();
});

/**
 * Returns a list of compilation tasks.
 *
 * This is the task to override to add support for other languages.
 */
subtask(TASK_COMPILE_GET_COMPILATION_TASKS, async (): Promise<string[]> => {
  return [TASK_COMPILE_SOLIDITY];
});

/**
 * Main compile task.
 *
 * This is a meta-task that just gets all the compilation tasks and runs them.
 * Right now there's only a "compile solidity" task.
 */
task(TASK_COMPILE, "Compiles the entire project, building all artifacts")
  .addFlag("force", "Force compilation ignoring cache")
  .addFlag("quiet", "Makes the compilation process less verbose")
  .addParam(
    "concurrency",
    "Number of compilation jobs executed in parallel. Defaults to the number of CPU cores - 1",
    DEFAULT_CONCURRENCY_LEVEL,
    types.int
  )
  .setAction(async (compilationArgs: any, { run }) => {
    const compilationTasks: string[] = await run(
      TASK_COMPILE_GET_COMPILATION_TASKS
    );

    for (const compilationTask of compilationTasks) {
      await run(compilationTask, compilationArgs);
    }

    await run(TASK_COMPILE_REMOVE_OBSOLETE_ARTIFACTS);
  });

/**
 * If a file is present in the cache, but some of its artifacts are missing on
 * disk, we remove it from the cache to force it to be recompiled.
 */
async function invalidateCacheMissingArtifacts(
  solidityFilesCache: SolidityFilesCache,
  artifacts: Artifacts,
  resolvedFiles: ResolvedFile[]
): Promise<SolidityFilesCache> {
  const paths = new Set(await artifacts.getArtifactPaths());

  for (const file of resolvedFiles) {
    const cacheEntry = solidityFilesCache.getEntry(file.absolutePath);

    if (cacheEntry === undefined) {
      continue;
    }

    const { artifacts: emittedArtifacts } = cacheEntry;
    for (const emittedArtifact of emittedArtifacts) {
      const fqn = getFullyQualifiedName(file.sourceName, emittedArtifact);
      const path = artifacts.formArtifactPathFromFullyQualifiedName(fqn);

      if (!paths.has(path)) {
        log(
          `Invalidate cache for '${file.absolutePath}' because artifact '${fqn}' doesn't exist`
        );

        solidityFilesCache.removeEntry(file.absolutePath);
        break;
      }
    }
  }

  artifacts.clearCache?.();

  return solidityFilesCache;
}

/**
 * Checks if the given compilation job needs to be done.
 */
function needsCompilation(
  job: taskTypes.CompilationJob,
  cache: SolidityFilesCache
): boolean {
  for (const file of job.getResolvedFiles()) {
    const hasChanged = cache.hasFileChanged(
      file.absolutePath,
      file.contentHash,
      // we only check if the solcConfig is different for files that
      // emit artifacts
      job.emitsArtifacts(file) ? job.getSolcConfig() : undefined
    );

    if (hasChanged) {
      return true;
    }
  }

  return false;
}

function hasCompilationErrors(output: any): boolean {
  return output.errors?.some((x: any) => x.severity === "error");
}

/**
 * This function returns a properly formatted Internal Compiler Error message.
 *
 * This is present due to a bug in Solidity. See: https://github.com/ethereum/solidity/issues/9926
 *
 * If the error is not an ICE, or if it's properly formatted, this function returns undefined.
 */
function getFormattedInternalCompilerErrorMessage(error: {
  formattedMessage: string;
  message: string;
  type: string;
}): string | undefined {
  if (error.formattedMessage.trim() !== "InternalCompilerError:") {
    return;
  }

  // We trim any final `:`, as we found some at the end of the error messages,
  // and then trim just in case a blank space was left
  return `${error.type}: ${error.message}`.replace(/[:\s]*$/g, "").trim();
}
