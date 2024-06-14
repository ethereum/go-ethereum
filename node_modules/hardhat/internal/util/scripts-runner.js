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
exports.runScriptWithHardhat = exports.runScript = void 0;
const debug_1 = __importDefault(require("debug"));
const path_1 = __importDefault(require("path"));
const execution_mode_1 = require("../core/execution-mode");
const env_variables_1 = require("../core/params/env-variables");
const log = (0, debug_1.default)("hardhat:core:scripts-runner");
async function runScript(scriptPath, scriptArgs = [], extraNodeArgs = [], extraEnvVars = {}) {
    const { fork } = await Promise.resolve().then(() => __importStar(require("child_process")));
    return new Promise((resolve, reject) => {
        const processExecArgv = withFixedInspectArg(process.execArgv);
        const nodeArgs = [
            ...processExecArgv,
            ...getTsNodeArgsIfNeeded(scriptPath, extraEnvVars.HARDHAT_TYPECHECK === "true"),
            ...extraNodeArgs,
        ];
        const envVars = { ...process.env, ...extraEnvVars };
        const childProcess = fork(scriptPath, scriptArgs, {
            stdio: "inherit",
            execArgv: nodeArgs,
            env: envVars,
        });
        childProcess.once("close", (status) => {
            log(`Script ${scriptPath} exited with status code ${status ?? "null"}`);
            resolve(status);
        });
        childProcess.once("error", reject);
    });
}
exports.runScript = runScript;
async function runScriptWithHardhat(hardhatArguments, scriptPath, scriptArgs = [], extraNodeArgs = [], extraEnvVars = {}) {
    log(`Creating Hardhat subprocess to run ${scriptPath}`);
    return runScript(scriptPath, scriptArgs, [
        ...extraNodeArgs,
        "--require",
        path_1.default.join(__dirname, "..", "..", "register"),
    ], {
        ...(0, env_variables_1.getEnvVariablesMap)(hardhatArguments),
        ...extraEnvVars,
    });
}
exports.runScriptWithHardhat = runScriptWithHardhat;
/**
 * Fix debugger "inspect" arg from process.argv, if present.
 *
 * When running this process with a debugger, a debugger port
 * is specified via the "--inspect-brk=" arg param in some IDEs/setups.
 *
 * This normally works, but if we do a fork afterwards, we'll get an error stating
 * that the port is already in use (since the fork would also use the same args,
 * therefore the same port number). To prevent this issue, we could replace the port number with
 * a different free one, or simply use the port-agnostic --inspect" flag, and leave the debugger
 * port selection to the Node process itself, which will pick an empty AND valid one.
 *
 * This way, we can properly use the debugger for this process AND for the executed
 * script itself - even if it's compiled using ts-node.
 */
function withFixedInspectArg(argv) {
    const fixIfInspectArg = (arg) => {
        if (arg.toLowerCase().includes("--inspect-brk=")) {
            return "--inspect";
        }
        return arg;
    };
    return argv.map(fixIfInspectArg);
}
function getTsNodeArgsIfNeeded(scriptPath, shouldTypecheck) {
    if (process.execArgv.includes("ts-node/register")) {
        return [];
    }
    // if we are running the tests we only want to transpile, or these tests
    // take forever
    if ((0, execution_mode_1.isRunningHardhatCoreTests)()) {
        return ["--require", "ts-node/register/transpile-only"];
    }
    // If the script we are going to run is .ts we need ts-node
    if (/\.tsx?$/i.test(scriptPath)) {
        return [
            "--require",
            `ts-node/register${shouldTypecheck ? "" : "/transpile-only"}`,
        ];
    }
    return [];
}
//# sourceMappingURL=scripts-runner.js.map