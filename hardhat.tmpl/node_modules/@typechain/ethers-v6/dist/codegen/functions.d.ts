import { AbiParameter, CodegenConfig, EventArgDeclaration, FunctionDeclaration } from 'typechain';
interface GenerateFunctionOptions {
    overrideOutput?: string;
    codegenConfig: CodegenConfig;
}
export declare function codegenFunctions(options: GenerateFunctionOptions, fns: FunctionDeclaration[]): string;
export declare function codegenForOverloadedFunctions(fns: FunctionDeclaration[]): string;
export declare function generateInterfaceFunctionDescription(fn: FunctionDeclaration): string;
export declare function generateFunctionNameOrSignature(fn: FunctionDeclaration, useSignature: boolean): string;
export declare function generateGetFunctionForInterface(args: string[]): string;
export declare function generateGetFunctionForContract(fn: FunctionDeclaration, useSignature: boolean): string;
export declare function generateEncodeFunctionDataOverload(fn: FunctionDeclaration, useSignature: boolean): string;
export declare function generateDecodeFunctionResultOverload(fn: FunctionDeclaration, useSignature: boolean): string;
export declare function generateParamNames(params: Array<AbiParameter | EventArgDeclaration>): string;
export declare const FUNCTION_IMPORTS: string[];
export {};
