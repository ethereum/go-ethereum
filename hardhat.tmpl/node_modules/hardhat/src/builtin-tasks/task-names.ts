export const TASK_CHECK = "check";

export const TASK_CLEAN = "clean";
export const TASK_CLEAN_GLOBAL = "clean:global";

export const TASK_COMPILE = "compile";
export const TASK_COMPILE_GET_COMPILATION_TASKS =
  "compile:get-compilation-tasks";
export const TASK_COMPILE_SOLIDITY = "compile:solidity";
export const TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS =
  "compile:solidity:get-source-paths";
export const TASK_COMPILE_SOLIDITY_GET_SOURCE_NAMES =
  "compile:solidity:get-source-names";
export const TASK_COMPILE_SOLIDITY_READ_FILE = "compile:solidity:read-file";
export const TASK_COMPILE_TRANSFORM_IMPORT_NAME =
  "compile:solidity:transform-import-name";
export const TASK_COMPILE_GET_REMAPPINGS = "compile:solidity:get-remappings";
export const TASK_COMPILE_SOLIDITY_GET_DEPENDENCY_GRAPH =
  "compile:solidity:get-dependency-graph";
export const TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOBS =
  "compile:solidity:get-compilation-jobs";
export const TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOB_FOR_FILE =
  "compile:solidity:get-compilation-job-for-file";
export const TASK_COMPILE_SOLIDITY_FILTER_COMPILATION_JOBS =
  "compile:solidity:filter-compilation-jobs";
export const TASK_COMPILE_SOLIDITY_MERGE_COMPILATION_JOBS =
  "compile:solidity:merge-compilation-jobs";
export const TASK_COMPILE_SOLIDITY_LOG_NOTHING_TO_COMPILE =
  "compile:solidity:log:nothing-to-compile";
export const TASK_COMPILE_SOLIDITY_COMPILE_JOB = "compile:solidity:compile-job";
export const TASK_COMPILE_SOLIDITY_LOG_RUN_COMPILER_START =
  "compile:solidity:log:run-compiler-start";
export const TASK_COMPILE_SOLIDITY_LOG_RUN_COMPILER_END =
  "compile:solidity:log:run-compiler-end";
export const TASK_COMPILE_SOLIDITY_COMPILE_JOBS =
  "compile:solidity:compile-jobs";
export const TASK_COMPILE_SOLIDITY_GET_COMPILER_INPUT =
  "compile:solidity:get-compiler-input";
export const TASK_COMPILE_SOLIDITY_COMPILE = "compile:solidity:compile";
export const TASK_COMPILE_SOLIDITY_COMPILE_SOLC =
  "compile:solidity:solc:compile";
export const TASK_COMPILE_SOLIDITY_GET_SOLC_BUILD =
  "compile:solidity:solc:get-build";
export const TASK_COMPILE_SOLIDITY_LOG_DOWNLOAD_COMPILER_START =
  "compile:solidity:log:download-compiler-start";
export const TASK_COMPILE_SOLIDITY_LOG_DOWNLOAD_COMPILER_END =
  "compile:solidity:log:download-compiler-end";
export const TASK_COMPILE_SOLIDITY_RUN_SOLCJS = "compile:solidity:solcjs:run";
export const TASK_COMPILE_SOLIDITY_RUN_SOLC = "compile:solidity:solc:run";
export const TASK_COMPILE_SOLIDITY_CHECK_ERRORS =
  "compile:solidity:check-errors";
export const TASK_COMPILE_SOLIDITY_LOG_COMPILATION_ERRORS =
  "compile:solidity:log:compilation-errors";
export const TASK_COMPILE_SOLIDITY_EMIT_ARTIFACTS =
  "compile:solidity:emit-artifacts";
export const TASK_COMPILE_SOLIDITY_GET_ARTIFACT_FROM_COMPILATION_OUTPUT =
  "compile:solidity:get-artifact-from-compilation-output";
export const TASK_COMPILE_SOLIDITY_HANDLE_COMPILATION_JOBS_FAILURES =
  "compile:solidity:handle-compilation-jobs-failures";
export const TASK_COMPILE_SOLIDITY_GET_COMPILATION_JOBS_FAILURE_REASONS =
  "compile:solidity:get-compilation-jobs-failure-reasons";
export const TASK_COMPILE_SOLIDITY_LOG_COMPILATION_RESULT =
  "compile:solidity:log:compilation-result";
export const TASK_COMPILE_REMOVE_OBSOLETE_ARTIFACTS =
  "compile:remove-obsolete-artifacts";

export const TASK_CONSOLE = "console";

export const TASK_FLATTEN = "flatten";
export const TASK_FLATTEN_GET_FLATTENED_SOURCE =
  "flatten:get-flattened-sources";
export const TASK_FLATTEN_GET_FLATTENED_SOURCE_AND_METADATA =
  "flatten:get-flattened-sources-and-metadata";
export const TASK_FLATTEN_GET_DEPENDENCY_GRAPH = "flatten:get-dependency-graph";

export const TASK_HELP = "help";

export const TASK_RUN = "run";

export const TASK_NODE = "node";
export const TASK_NODE_GET_PROVIDER = "node:get-provider";
export const TASK_NODE_CREATE_SERVER = "node:create-server";
export const TASK_NODE_SERVER_CREATED = "node:server-created";
export const TASK_NODE_SERVER_READY = "node:server-ready";

export const TASK_TEST = "test";

export const TASK_TEST_RUN_SHOW_FORK_RECOMMENDATIONS =
  "test:show-fork-recommendations";
export const TASK_TEST_RUN_MOCHA_TESTS = "test:run-mocha-tests";
export const TASK_TEST_GET_TEST_FILES = "test:get-test-files";
export const TASK_TEST_SETUP_TEST_ENVIRONMENT = "test:setup-test-environment";
