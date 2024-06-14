"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const debug_1 = __importDefault(require("debug"));
const fs_extra_1 = __importDefault(require("fs-extra"));
const config_env_1 = require("../internal/core/config/config-env");
const errors_1 = require("../internal/core/errors");
const errors_list_1 = require("../internal/core/errors-list");
const scripts_runner_1 = require("../internal/util/scripts-runner");
const task_names_1 = require("./task-names");
const log = (0, debug_1.default)("hardhat:core:tasks:run");
(0, config_env_1.task)(task_names_1.TASK_RUN, "Runs a user-defined script after compiling the project")
    .addPositionalParam("script", "A js file to be run within hardhat's environment")
    .addFlag("noCompile", "Don't compile before running this task")
    .setAction(async ({ script, noCompile }, { run, hardhatArguments }) => {
    if (!(await fs_extra_1.default.pathExists(script))) {
        throw new errors_1.HardhatError(errors_list_1.ERRORS.BUILTIN_TASKS.RUN_FILE_NOT_FOUND, {
            script,
        });
    }
    if (!noCompile) {
        await run(task_names_1.TASK_COMPILE, { quiet: true });
    }
    log(`Running script ${script} in a subprocess so we can wait for it to complete`);
    try {
        process.exitCode = await (0, scripts_runner_1.runScriptWithHardhat)(hardhatArguments, script);
    }
    catch (error) {
        if (error instanceof Error) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.BUILTIN_TASKS.RUN_SCRIPT_ERROR, {
                script,
                error: error.message,
            }, error);
        }
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw error;
    }
});
//# sourceMappingURL=run.js.map