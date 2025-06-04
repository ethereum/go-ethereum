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
exports.execFileWithInput = exports.NativeCompiler = exports.Compiler = void 0;
const child_process_1 = require("child_process");
const fs = __importStar(require("fs"));
const node_os_1 = __importDefault(require("node:os"));
const node_path_1 = __importDefault(require("node:path"));
const semver = __importStar(require("semver"));
const errors_1 = require("../../core/errors");
const errors_list_1 = require("../../core/errors-list");
class Compiler {
    constructor(_pathToSolcJs) {
        this._pathToSolcJs = _pathToSolcJs;
    }
    async compile(input) {
        const scriptPath = node_path_1.default.join(__dirname, "./solcjs-runner.js");
        let output;
        try {
            const { stdout } = await execFileWithInput(process.execPath, [scriptPath, this._pathToSolcJs], JSON.stringify(input), {
                maxBuffer: 1024 * 1024 * 500,
            });
            output = stdout;
        }
        catch (e) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.SOLC.SOLCJS_ERROR, {}, e);
        }
        return JSON.parse(output);
    }
}
exports.Compiler = Compiler;
class NativeCompiler {
    constructor(_pathToSolc, _solcVersion) {
        this._pathToSolc = _pathToSolc;
        this._solcVersion = _solcVersion;
    }
    async compile(input) {
        const args = ["--standard-json"];
        // Logic to make sure that solc default import callback is not being used.
        // If solcVersion is not defined or <= 0.6.8, do not add extra args.
        if (this._solcVersion !== undefined) {
            if (semver.gte(this._solcVersion, "0.8.22")) {
                // version >= 0.8.22
                args.push("--no-import-callback");
            }
            else if (semver.gte(this._solcVersion, "0.6.9")) {
                // version >= 0.6.9
                const tmpFolder = node_path_1.default.join(node_os_1.default.tmpdir(), "hardhat-solc");
                fs.mkdirSync(tmpFolder, { recursive: true });
                args.push(`--base-path`);
                args.push(tmpFolder);
            }
        }
        let output;
        try {
            const { stdout } = await execFileWithInput(this._pathToSolc, args, JSON.stringify(input), {
                maxBuffer: 1024 * 1024 * 500,
            });
            output = stdout;
        }
        catch (e) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.SOLC.CANT_RUN_NATIVE_COMPILER, {}, e);
        }
        return JSON.parse(output);
    }
}
exports.NativeCompiler = NativeCompiler;
/**
 * Executes a command using execFile, writes provided input to stdin,
 * and returns a Promise that resolves with stdout and stderr.
 *
 * @param {string} file - The file to execute.
 * @param {readonly string[]} args - The arguments to pass to the file.
 * @param {ExecFileOptions} options - The options to pass to the exec function.
 * @returns {Promise<{stdout: string, stderr: string}>}
 */
async function execFileWithInput(file, args, input, options = {}) {
    return new Promise((resolve, reject) => {
        const child = (0, child_process_1.execFile)(file, args, options, (error, stdout, stderr) => {
            // `error` is any execution error. e.g. command not found, non-zero exit code, etc.
            if (error !== null) {
                reject(error);
            }
            else {
                resolve({ stdout, stderr });
            }
        });
        // This could be triggered if node fails to spawn the child process
        child.on("error", (err) => {
            reject(err);
        });
        const stdin = child.stdin;
        if (stdin !== null) {
            stdin.on("error", (err) => {
                // This captures EPIPE error
                reject(err);
            });
            child.once("spawn", () => {
                if (!stdin.writable || child.killed) {
                    return reject(new Error("Failed to write to unwritable stdin"));
                }
                stdin.write(input, (error) => {
                    if (error !== null && error !== undefined) {
                        reject(error);
                    }
                    stdin.end();
                });
            });
        }
        else {
            reject(new Error("No stdin on child process"));
        }
    });
}
exports.execFileWithInput = execFileWithInput;
//# sourceMappingURL=index.js.map