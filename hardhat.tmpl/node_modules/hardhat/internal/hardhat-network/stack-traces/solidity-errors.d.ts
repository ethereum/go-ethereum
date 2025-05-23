/// <reference types="node" />
import { SolidityStackTrace } from "./solidity-stack-trace";
declare const inspect: unique symbol;
export declare function getCurrentStack(): NodeJS.CallSite[];
export declare function wrapWithSolidityErrorsCorrection(f: () => Promise<any>, stackFramesToRemove: number): Promise<any>;
export declare function encodeSolidityStackTrace(fallbackMessage: string, stackTrace: SolidityStackTrace, previousStack?: NodeJS.CallSite[]): SolidityError;
export declare class SolidityError extends Error {
    readonly stackTrace: SolidityStackTrace;
    constructor(message: string, stackTrace: SolidityStackTrace);
    [inspect](): string;
    inspect(): string;
}
export {};
//# sourceMappingURL=solidity-errors.d.ts.map