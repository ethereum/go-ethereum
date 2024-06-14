"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const chalk_1 = __importDefault(require("chalk"));
const path_1 = __importDefault(require("path"));
const constants_1 = require("../internal/constants");
const config_env_1 = require("../internal/core/config/config-env");
const errors_1 = require("../internal/core/errors");
const errors_list_1 = require("../internal/core/errors-list");
const typescript_support_1 = require("../internal/core/typescript-support");
const disk_cache_1 = require("../internal/hardhat-network/provider/utils/disk-cache");
const fork_recomendations_banner_1 = require("../internal/hardhat-network/provider/utils/fork-recomendations-banner");
const strings_1 = require("../internal/util/strings");
const fs_utils_1 = require("../internal/util/fs-utils");
const packageInfo_1 = require("../internal/util/packageInfo");
const task_names_1 = require("./task-names");
(0, config_env_1.subtask)(task_names_1.TASK_TEST_GET_TEST_FILES)
    .addOptionalVariadicPositionalParam("testFiles", "An optional list of files to test", [])
    .setAction(async ({ testFiles }, { config }) => {
    if (testFiles.length !== 0) {
        const testFilesAbsolutePaths = testFiles.map((x) => path_1.default.resolve(process.cwd(), x));
        return testFilesAbsolutePaths;
    }
    const jsFiles = await (0, fs_utils_1.getAllFilesMatching)(config.paths.tests, typescript_support_1.isJavascriptFile);
    if (!(0, typescript_support_1.isRunningWithTypescript)(config)) {
        return jsFiles;
    }
    const tsFiles = await (0, fs_utils_1.getAllFilesMatching)(config.paths.tests, typescript_support_1.isTypescriptFile);
    return [...jsFiles, ...tsFiles];
});
(0, config_env_1.subtask)(task_names_1.TASK_TEST_SETUP_TEST_ENVIRONMENT, async () => { });
let testsAlreadyRun = false;
(0, config_env_1.subtask)(task_names_1.TASK_TEST_RUN_MOCHA_TESTS)
    .addFlag("parallel", "Run tests in parallel")
    .addFlag("bail", "Stop running tests after the first test failure")
    .addOptionalParam("grep", "Only run tests matching the given string or regexp")
    .addOptionalVariadicPositionalParam("testFiles", "An optional list of files to test", [])
    .setAction(async (taskArgs, { config }) => {
    const { default: Mocha } = await Promise.resolve().then(() => __importStar(require("mocha")));
    const mochaConfig = { ...config.mocha };
    if (taskArgs.grep !== undefined) {
        mochaConfig.grep = taskArgs.grep;
    }
    if (taskArgs.bail) {
        mochaConfig.bail = true;
    }
    if (taskArgs.parallel) {
        mochaConfig.parallel = true;
    }
    if (mochaConfig.parallel === true) {
        const mochaRequire = mochaConfig.require ?? [];
        if (!mochaRequire.includes("hardhat/register")) {
            mochaRequire.push("hardhat/register");
        }
        mochaConfig.require = mochaRequire;
    }
    const mocha = new Mocha(mochaConfig);
    taskArgs.testFiles.forEach((file) => mocha.addFile(file));
    // if the project is of type "module" or if there's some ESM test file,
    // we call loadFilesAsync to enable Mocha's ESM support
    const projectPackageJson = await (0, packageInfo_1.getProjectPackageJson)();
    const isTypeModule = projectPackageJson.type === "module";
    const hasEsmTest = taskArgs.testFiles.some((file) => file.endsWith(".mjs"));
    if (isTypeModule || hasEsmTest) {
        // Because of the way the ESM cache works, loadFilesAsync doesn't work
        // correctly if used twice within the same process, so we throw an error
        // in that case
        if (testsAlreadyRun) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.BUILTIN_TASKS.TEST_TASK_ESM_TESTS_RUN_TWICE);
        }
        testsAlreadyRun = true;
        // This instructs Mocha to use the more verbose file loading infrastructure
        // which supports both ESM and CJS
        await mocha.loadFilesAsync();
    }
    const testFailures = await new Promise((resolve) => {
        mocha.run(resolve);
    });
    mocha.dispose();
    return testFailures;
});
(0, config_env_1.subtask)(task_names_1.TASK_TEST_RUN_SHOW_FORK_RECOMMENDATIONS).setAction(async (_, { config, network }) => {
    if (network.name !== constants_1.HARDHAT_NETWORK_NAME) {
        return;
    }
    const forkCache = (0, disk_cache_1.getForkCacheDirPath)(config.paths);
    await (0, fork_recomendations_banner_1.showForkRecommendationsBannerIfNecessary)(network.config, forkCache);
});
(0, config_env_1.task)(task_names_1.TASK_TEST, "Runs mocha tests")
    .addOptionalVariadicPositionalParam("testFiles", "An optional list of files to test", [])
    .addFlag("noCompile", "Don't compile before running this task")
    .addFlag("parallel", "Run tests in parallel")
    .addFlag("bail", "Stop running tests after the first test failure")
    .addOptionalParam("grep", "Only run tests matching the given string or regexp")
    .setAction(async ({ testFiles, noCompile, parallel, bail, grep, }, { run, network }) => {
    if (!noCompile) {
        await run(task_names_1.TASK_COMPILE, { quiet: true });
    }
    const files = await run(task_names_1.TASK_TEST_GET_TEST_FILES, { testFiles });
    await run(task_names_1.TASK_TEST_SETUP_TEST_ENVIRONMENT);
    await run(task_names_1.TASK_TEST_RUN_SHOW_FORK_RECOMMENDATIONS);
    const testFailures = await run(task_names_1.TASK_TEST_RUN_MOCHA_TESTS, {
        testFiles: files,
        parallel,
        bail,
        grep,
    });
    if (network.name === constants_1.HARDHAT_NETWORK_NAME) {
        const stackTracesFailures = await network.provider.send("hardhat_getStackTraceFailuresCount");
        if (stackTracesFailures !== 0) {
            console.warn(chalk_1.default.yellow(`Failed to generate ${stackTracesFailures} ${(0, strings_1.pluralize)(stackTracesFailures, "stack trace")}. Run Hardhat with --verbose to learn more.`));
        }
    }
    process.exitCode = testFailures;
    return testFailures;
});
//# sourceMappingURL=test.js.map