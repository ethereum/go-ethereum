/// <reference types="node" />
import { ExecFileOptions } from "node:child_process";
import { CompilerInput, CompilerOutput } from "../../../types";
export interface ICompiler {
    compile(input: CompilerInput): Promise<CompilerOutput>;
}
export declare class Compiler implements ICompiler {
    private _pathToSolcJs;
    constructor(_pathToSolcJs: string);
    compile(input: CompilerInput): Promise<any>;
}
export declare class NativeCompiler implements ICompiler {
    private _pathToSolc;
    private _solcVersion?;
    constructor(_pathToSolc: string, _solcVersion?: string | undefined);
    compile(input: CompilerInput): Promise<any>;
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
export declare function execFileWithInput(file: string, args: readonly string[], input: string, options?: ExecFileOptions): Promise<{
    stdout: string;
    stderr: string;
}>;
//# sourceMappingURL=index.d.ts.map