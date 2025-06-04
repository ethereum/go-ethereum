import { execFile } from "child_process";
import * as fs from "fs";
import os from "node:os";
import path from "node:path";
import * as semver from "semver";
import { ExecFileOptions } from "node:child_process";
import { CompilerInput, CompilerOutput } from "../../../types";
import { HardhatError } from "../../core/errors";
import { ERRORS } from "../../core/errors-list";

export interface ICompiler {
  compile(input: CompilerInput): Promise<CompilerOutput>;
}

export class Compiler implements ICompiler {
  constructor(private _pathToSolcJs: string) {}

  public async compile(input: CompilerInput) {
    const scriptPath = path.join(__dirname, "./solcjs-runner.js");

    let output: string;
    try {
      const { stdout } = await execFileWithInput(
        process.execPath,
        [scriptPath, this._pathToSolcJs],
        JSON.stringify(input),
        {
          maxBuffer: 1024 * 1024 * 500,
        }
      );

      output = stdout;
    } catch (e: any) {
      throw new HardhatError(ERRORS.SOLC.SOLCJS_ERROR, {}, e);
    }

    return JSON.parse(output);
  }
}

export class NativeCompiler implements ICompiler {
  constructor(private _pathToSolc: string, private _solcVersion?: string) {}

  public async compile(input: CompilerInput) {
    const args = ["--standard-json"];

    // Logic to make sure that solc default import callback is not being used.
    // If solcVersion is not defined or <= 0.6.8, do not add extra args.
    if (this._solcVersion !== undefined) {
      if (semver.gte(this._solcVersion, "0.8.22")) {
        // version >= 0.8.22
        args.push("--no-import-callback");
      } else if (semver.gte(this._solcVersion, "0.6.9")) {
        // version >= 0.6.9
        const tmpFolder = path.join(os.tmpdir(), "hardhat-solc");
        fs.mkdirSync(tmpFolder, { recursive: true });
        args.push(`--base-path`);
        args.push(tmpFolder);
      }
    }

    let output: string;
    try {
      const { stdout } = await execFileWithInput(
        this._pathToSolc,
        args,
        JSON.stringify(input),
        {
          maxBuffer: 1024 * 1024 * 500,
        }
      );

      output = stdout;
    } catch (e: any) {
      throw new HardhatError(ERRORS.SOLC.CANT_RUN_NATIVE_COMPILER, {}, e);
    }

    return JSON.parse(output);
  }
}

/**
 * Executes a command using execFile, writes provided input to stdin,
 * and returns a Promise that resolves with stdout and stderr.
 *
 * @param {string} file - The file to execute.
 * @param {readonly string[]} args - The arguments to pass to the file.
 * @param {ExecFileOptions} options - The options to pass to the exec function.
 * @returns {Promise<{stdout: string, stderr: string}>}
 */
export async function execFileWithInput(
  file: string,
  args: readonly string[],
  input: string,
  options: ExecFileOptions = {}
): Promise<{ stdout: string; stderr: string }> {
  return new Promise((resolve, reject) => {
    const child = execFile(file, args, options, (error, stdout, stderr) => {
      // `error` is any execution error. e.g. command not found, non-zero exit code, etc.
      if (error !== null) {
        reject(error);
      } else {
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
    } else {
      reject(new Error("No stdin on child process"));
    }
  });
}
