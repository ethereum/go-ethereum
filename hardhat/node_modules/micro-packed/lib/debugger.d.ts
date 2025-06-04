import * as P from './index.ts';
export declare function table(data: any[]): void;
export declare function decode(coder: P.CoderType<any>, data: string | P.Bytes, forcePrint?: boolean): ReturnType<(typeof coder)['decode']>;
export declare function diff(coder: P.CoderType<any>, actual: string | P.Bytes, expected: string | P.Bytes, skipSame?: boolean): void;
/**
 * Wraps a CoderType with debug logging for encoding and decoding operations.
 * @param inner - Inner CoderType to wrap.
 * @returns Inner wrapped in debug prints via console.log.
 * @example
 * const debugInt = P.debug(P.U32LE); // Will print info to console on encoding/decoding
 */
export declare function debug<T>(inner: P.CoderType<T>): P.CoderType<T>;
//# sourceMappingURL=debugger.d.ts.map