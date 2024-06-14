/// <reference types="node" />
import BN = require('bn.js');
/**
 * Returns a buffer filled with 0s.
 * @param bytes the number of bytes the buffer should be
 */
export declare const zeros: (bytes: number) => Buffer;
/**
 * Left Pads an `Array` or `Buffer` with leading zeros till it has `length` bytes.
 * Or it truncates the beginning if it exceeds.
 * @param msg the value to pad (Buffer|Array)
 * @param length the number of bytes the output should be
 * @param right whether to start padding form the left or right
 * @return (Buffer|Array)
 */
export declare const setLengthLeft: (msg: any, length: number, right?: boolean) => any;
export declare const setLength: (msg: any, length: number, right?: boolean) => any;
/**
 * Right Pads an `Array` or `Buffer` with leading zeros till it has `length` bytes.
 * Or it truncates the beginning if it exceeds.
 * @param msg the value to pad (Buffer|Array)
 * @param length the number of bytes the output should be
 * @return (Buffer|Array)
 */
export declare const setLengthRight: (msg: any, length: number) => any;
/**
 * Trims leading zeros from a `Buffer` or an `Array`.
 * @param a (Buffer|Array|String)
 * @return (Buffer|Array|String)
 */
export declare const unpad: (a: any) => any;
export declare const stripZeros: (a: any) => any;
/**
 * Attempts to turn a value into a `Buffer`. As input it supports `Buffer`, `String`, `Number`, null/undefined, `BN` and other objects with a `toArray()` method.
 * @param v the value
 */
export declare const toBuffer: (v: any) => Buffer;
/**
 * Converts a `Buffer` to a `Number`.
 * @param buf `Buffer` object to convert
 * @throws If the input number exceeds 53 bits.
 */
export declare const bufferToInt: (buf: Buffer) => number;
/**
 * Converts a `Buffer` into a `0x`-prefixed hex `String`.
 * @param buf `Buffer` object to convert
 */
export declare const bufferToHex: (buf: Buffer) => string;
/**
 * Interprets a `Buffer` as a signed integer and returns a `BN`. Assumes 256-bit numbers.
 * @param num Signed integer value
 */
export declare const fromSigned: (num: Buffer) => BN;
/**
 * Converts a `BN` to an unsigned integer and returns it as a `Buffer`. Assumes 256-bit numbers.
 * @param num
 */
export declare const toUnsigned: (num: BN) => Buffer;
/**
 * Adds "0x" to a given `String` if it does not already start with "0x".
 */
export declare const addHexPrefix: (str: string) => string;
/**
 * Converts a `Buffer` or `Array` to JSON.
 * @param ba (Buffer|Array)
 * @return (Array|String|null)
 */
export declare const baToJSON: (ba: any) => any;
