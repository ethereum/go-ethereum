/// <reference types="node" />
import * as fs from 'fs';
import { sync as mkdirp } from 'mkdirp';
import * as prettier from 'prettier';
import { MarkOptional } from 'ts-essentials';
export interface Config {
    cwd: string;
    target: string;
    outDir?: string | undefined;
    prettier?: object | undefined;
    filesToProcess: string[];
    allFiles: string[];
    /**
     * Optional path to directory with ABI files.
     * If not specified, inferred to be lowest common path of all input files.
     */
    inputDir: string;
    flags: CodegenConfig;
}
export interface CodegenConfig {
    alwaysGenerateOverloads: boolean;
    discriminateTypes: boolean;
    tsNocheck?: boolean;
    node16Modules?: boolean;
    environment: 'hardhat' | undefined;
}
export type PublicConfig = MarkOptional<Config, 'flags' | 'inputDir'>;
export declare abstract class TypeChainTarget {
    readonly cfg: Config;
    abstract readonly name: string;
    constructor(cfg: Config);
    beforeRun(): Output | Promise<Output>;
    afterRun(): Output | Promise<Output>;
    abstract transformFile(file: FileDescription): Output | Promise<Output>;
}
export type Output = void | FileDescription | FileDescription[];
export interface FileDescription {
    path: string;
    contents: string;
}
export interface Services {
    fs: typeof fs;
    prettier: typeof prettier;
    mkdirp: typeof mkdirp;
}
