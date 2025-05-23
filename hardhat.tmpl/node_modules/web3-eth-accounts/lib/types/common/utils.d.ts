import { ToBytesInputTypes, TypeOutput, TypeOutputReturnType } from './types.js';
type ConfigHardfork = {
    name: string;
    block: null;
    timestamp: number;
} | {
    name: string;
    block: number;
    timestamp?: number;
};
/**
 * Removes '0x' from a given `String` if present
 * @param str the string value
 * @returns the string without 0x prefix
 */
export declare const stripHexPrefix: (str: string) => string;
/**
 * Parses a genesis.json exported from Geth into parameters for Common instance
 * @param json representing the Geth genesis file
 * @param name optional chain name
 * @returns parsed params
 */
export declare function parseGethGenesis(json: any, name?: string, mergeForkIdPostMerge?: boolean): {
    name: string;
    chainId: number;
    networkId: number;
    genesis: {
        timestamp: string;
        gasLimit: number;
        difficulty: number;
        nonce: string;
        extraData: string;
        mixHash: string;
        coinbase: string;
        baseFeePerGas: string;
    };
    hardfork: string | undefined;
    hardforks: ConfigHardfork[];
    bootstrapNodes: never[];
    consensus: {
        type: string;
        algorithm: string;
        clique: {
            period: any;
            epoch: any;
        };
        ethash?: undefined;
    } | {
        type: string;
        algorithm: string;
        ethash: {};
        clique?: undefined;
    };
};
/**
 * Pads a `String` to have an even length
 * @param value
 * @return output
 */
export declare function padToEven(value: string): string;
/**
 * Converts an `Number` to a `Uint8Array`
 * @param {Number} i
 * @return {Uint8Array}
 */
export declare const intToUint8Array: (i: number) => Uint8Array;
/**
 * Attempts to turn a value into a `Uint8Array`.
 * Inputs supported: `Uint8Array` `String` (hex-prefixed), `Number`, null/undefined, `BigInt` and other objects
 * with a `toArray()` or `toUint8Array()` method.
 * @param v the value
 */
export declare const toUint8Array: (v: ToBytesInputTypes) => Uint8Array;
/**
 * Converts a {@link Uint8Array} to a {@link bigint}
 */
export declare function uint8ArrayToBigInt(buf: Uint8Array): bigint;
/**
 * Converts a {@link bigint} to a {@link Uint8Array}
 */
export declare function bigIntToUint8Array(num: bigint): Uint8Array;
/**
 * Returns a Uint8Array filled with 0s.
 * @param bytes the number of bytes the Uint8Array should be
 */
export declare const zeros: (bytes: number) => Uint8Array;
/**
 * Throws if input is not a Uint8Array
 * @param {Uint8Array} input value to check
 */
export declare function assertIsUint8Array(input: unknown): asserts input is Uint8Array;
/**
 * Left Pads a `Uint8Array` with leading zeros till it has `length` bytes.
 * Or it truncates the beginning if it exceeds.
 * @param msg the value to pad (Uint8Array)
 * @param length the number of bytes the output should be
 * @return (Uint8Array)
 */
export declare const setLengthLeft: (msg: Uint8Array, length: number) => Uint8Array;
/**
 * Trims leading zeros from a `Uint8Array`, `String` or `Number[]`.
 * @param a (Uint8Array|Array|String)
 * @return (Uint8Array|Array|String)
 */
export declare function stripZeros<T extends Uint8Array | number[] | string>(a: T): T;
/**
 * Trims leading zeros from a `Uint8Array`.
 * @param a (Uint8Array)
 * @return (Uint8Array)
 */
export declare const unpadUint8Array: (a: Uint8Array) => Uint8Array;
/**
 * Converts a {@link bigint} to a `0x` prefixed hex string
 */
export declare const bigIntToHex: (num: bigint) => string;
/**
 * Convert value from bigint to an unpadded Uint8Array
 * (useful for RLP transport)
 * @param value value to convert
 */
export declare function bigIntToUnpaddedUint8Array(value: bigint): Uint8Array;
/**
 * ECDSA public key recovery from signature.
 * NOTE: Accepts `v === 0 | v === 1` for EIP1559 transactions
 * @returns Recovered public key
 */
export declare const ecrecover: (msgHash: Uint8Array, v: bigint, r: Uint8Array, s: Uint8Array, chainId?: bigint) => Uint8Array;
/**
 * Convert an input to a specified type.
 * Input of null/undefined returns null/undefined regardless of the output type.
 * @param input value to convert
 * @param outputType type to output
 */
export declare function toType<T extends TypeOutput>(input: null, outputType: T): null;
export declare function toType<T extends TypeOutput>(input: undefined, outputType: T): undefined;
export declare function toType<T extends TypeOutput>(input: ToBytesInputTypes, outputType: T): TypeOutputReturnType[T];
export {};
//# sourceMappingURL=utils.d.ts.map