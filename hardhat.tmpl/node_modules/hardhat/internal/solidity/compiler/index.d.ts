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
//# sourceMappingURL=index.d.ts.map