/**
 * stack-trace - Parses node.js stack traces
 *
 * This was originally forked to fix this issue:
 * https://github.com/felixge/node-stack-trace/issues/31
 *
 * Mar 19,2019 - #4fd379e
 *
 * https://github.com/felixge/node-stack-trace/
 * @license MIT
 */
/** Decoded StackFrame */
export interface StackFrame {
    fileName: string;
    lineNumber: number;
    functionName: string;
    typeName: string;
    methodName: string;
    native: boolean;
    columnNumber: number;
}
/** Extracts StackFrames from the Error */
export declare function parse(err: Error): StackFrame[];
//# sourceMappingURL=stacktrace.d.ts.map