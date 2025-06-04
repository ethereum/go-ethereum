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
const debug_1 = __importDefault(require("debug"));
const fs_extra_1 = __importDefault(require("fs-extra"));
const path = __importStar(require("path"));
const semver = __importStar(require("semver"));
const config_env_1 = require("../internal/core/config/config-env");
const scripts_runner_1 = require("../internal/util/scripts-runner");
const task_names_1 = require("./task-names");
const log = (0, debug_1.default)("hardhat:core:tasks:console");
(0, config_env_1.task)(task_names_1.TASK_CONSOLE, "Opens a hardhat console")
    .addFlag("noCompile", "Don't compile before running this task")
    .setAction(async ({ noCompile }, { config, run, hardhatArguments }) => {
    if (!noCompile) {
        await run(task_names_1.TASK_COMPILE, { quiet: true });
    }
    await fs_extra_1.default.ensureDir(config.paths.cache);
    const historyFile = path.join(config.paths.cache, "console-history.txt");
    const nodeArgs = [];
    if (semver.gte(process.version, "10.0.0")) {
        nodeArgs.push("--experimental-repl-await");
    }
    log(`Creating a Node REPL subprocess with Hardhat's register so we can set some Node's flags`);
    // Running the script "" is like running `node`, so this starts the repl
    await (0, scripts_runner_1.runScriptWithHardhat)(hardhatArguments, "", [], nodeArgs, {
        NODE_REPL_HISTORY: historyFile,
    });
});
//# sourceMappingURL=console.js.map