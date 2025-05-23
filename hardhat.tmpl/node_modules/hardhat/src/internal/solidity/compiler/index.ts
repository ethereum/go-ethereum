import { execFile } from "child_process";
import * as fs from "fs";
import os from "node:os";
import path from "node:path";
import * as semver from "semver";
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

    const output: string = await new Promise((resolve, reject) => {
      try {
        const subprocess = execFile(
          process.execPath,
          [scriptPath, this._pathToSolcJs],
          {
            maxBuffer: 1024 * 1024 * 500,
          },
          (err, stdout) => {
            if (err !== null) {
              return reject(err);
            }
            resolve(stdout);
          }
        );

        subprocess.stdin!.write(JSON.stringify(input));
        subprocess.stdin!.end();
      } catch (e: any) {
        throw new HardhatError(
          ERRORS.SOLC.SOLCJS_ERROR,
          { error: e.message },
          e
        );
      }
    });

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

    const output: string = await new Promise((resolve, reject) => {
      try {
        const process = execFile(
          this._pathToSolc,
          args,
          {
            maxBuffer: 1024 * 1024 * 500,
          },
          (err, stdout) => {
            if (err !== null) {
              return reject(err);
            }
            resolve(stdout);
          }
        );

        process.stdin!.write(JSON.stringify(input));
        process.stdin!.end();
      } catch (e: any) {
        throw new HardhatError(ERRORS.SOLC.CANT_RUN_NATIVE_COMPILER, {}, e);
      }
    });

    return JSON.parse(output);
  }
}
