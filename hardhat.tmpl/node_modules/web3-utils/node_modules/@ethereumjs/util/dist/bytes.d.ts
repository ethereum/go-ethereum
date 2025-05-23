/// <reference types="node" />
import type { NestedBufferArray, NestedUint8Array, PrefixedHexString, TransformableToArray, TransformableToBuffer } from './types';
/**
 * Converts a `Number` into a hex `String`
 * @param {Number} i
 * @return {String}
 */
export declare const intToHex: (i: number) => string;
/**
 * Converts an `Number` to a `Buffer`
 * @param {Number} i
 * @return {Buffer}
 */
export declare const intToBuffer: (i: number) => Buffer;
/**
 * Returns a buffer filled with 0s.
 * @param bytes the number of bytes the buffer should be
 */
export declare const zeros: (bytes: number) => Buffer;
/**
 * Left Pads a `Buffer` with leading zeros till it has `length` bytes.
 * Or it truncates the beginning if it exceeds.
 * @param msg the value to pad (Buffer)
 * @param length the number of bytes the output should be
 * @return (Buffer)
 */
export declare const setLengthLeft: (msg: Buffer, length: number) => Buffer;
/**
 * Right Pads a `Buffer` with trailing zeros till it has `length` bytes.
 * it truncates the end if it exceeds.
 * @param msg the value to pad (Buffer)
 * @param length the number of bytes the output should be
 * @return (Buffer)
 */
export declare const setLengthRight: (msg: Buffer, length: number) => Buffer;
/**
 * Trims leading zeros from a `Buffer`.
 * @param a (Buffer)
 * @return (Buffer)
 */
export declare const unpadBuffer: (a: Buffer) => Buffer;
/**
 * Trims leading zeros from an `Array` (of numbers).
 * @param a (number[])
 * @return (number[])
 */
export declare const unpadArray: (a: number[]) => number[];
/**
 * Trims leading zeros from a hex-prefixed `String`.
 * @param a (String)
 * @return (String)
 */
export declare const unpadHexString: (a: string) => string;
export declare type ToBufferInputTypes = PrefixedHexString | number | bigint | Buffer | Uint8Array | number[] | TransformableToArray | TransformableToBuffer | null | undefined;
/**
 * Attempts to turn a value into a `Buffer`.
 * Inputs supported: `Buffer`, `String` (hex-prefixed), `Number`, null/undefined, `BigInt` and other objects
 * with a `toArray()` or `toBuffer()` method.
 * @param v the value
 */
export declare const toBuffer: (v: ToBufferInputTypes) => Buffer;
/**
 * Converts a `Buffer` into a `0x`-prefixed hex `String`.
 * @param buf `Buffer` object to convert
 */
export declare const bufferToHex: (buf: Buffer) => string;
/**
 * Converts a {@link Buffer} to a {@link bigint}
 */
export declare function bufferToBigInt(buf: Buffer): bigint;
/**
 * Converts a {@link bigint} to a {@link Buffer}
 */
export declare function bigIntToBuffer(num: bigint): Buffer;
/**
 * Converts a `Buffer` to a `Number`.
 * @param buf `Buffer` object to convert
 * @throws If the input number exceeds 53 bits.
 */
export declare const bufferToInt: (buf: Buffer) => number;
/**
 * Interprets a `Buffer` as a signed integer and returns a `BigInt`. Assumes 256-bit numbers.
 * @param num Signed integer value
 */
export declare const fromSigned: (num: Buffer) => bigint;
/**
 * Converts a `BigInt` to an unsigned integer and returns it as a `Buffer`. Assumes 256-bit numbers.
 * @param num
 */
export declare const toUnsigned: (num: bigint) => Buffer;
/**
 * Adds "0x" to a given `String` if it does not already start with "0x".
 */
export declare const addHexPrefix: (str: string) => string;
/**
 * Shortens a string  or buffer's hex string representation to maxLength (default 50).
 *
 * Examples:
 *
 * Input:  '657468657265756d000000000000000000000000000000000000000000000000'
 * Output: '657468657265756d0000000000000000000000000000000000…'
 */
export declare function short(buffer: Buffer | string, maxLength?: number): string;
/**
 * Returns the utf8 string representation from a hex string.
 *
 * Examples:
 *
 * Input 1: '657468657265756d000000000000000000000000000000000000000000000000'
 * Input 2: '657468657265756d'
 * Input 3: '000000000000000000000000000000000000000000000000657468657265756d'
 *
 * Output (all 3 input variants): 'ethereum'
 *
 * Note that this method is not intended to be used with hex strings
 * representing quantities in both big endian or little endian notation.
 *
 * @param string Hex string, should be `0x` prefixed
 * @return Utf8 string
 */
export declare const toUtf8: (hex: string) => string;
/**
 * Converts a `Buffer` or `Array` to JSON.
 * @param ba (Buffer|Array)
 * @return (Array|String|null)
 */
export declare const baToJSON: (ba: any) => any;
/**
 * Checks provided Buffers for leading zeroes and throws if found.
 *
 * Examples:
 *
 * Valid values: 0x1, 0x, 0x01, 0x1234
 * Invalid values: 0x0, 0x00, 0x001, 0x0001
 *
 * Note: This method is useful for validating that RLP encoded integers comply with the rule that all
 * integer values encoded to RLP must be in the most compact form and contain no leading zero bytes
 * @param values An object containing string keys and Buffer values
 * @throws if any provided value is found to have leading zero bytes
 */
export declare const validateNoLeadingZeroes: (values: {
    [key: string]: Buffer | undefined;
}) => void;
/**
 * Converts a {@link Uint8Array} or {@link NestedUint8Array} to {@link Buffer} or {@link NestedBufferArray}
 */
export declare function arrToBufArr(arr: Uint8Array): Buffer;
export declare function arrToBufArr(arr: NestedUint8Array): NestedBufferArray;
export declare function arrToBufArr(arr: Uint8Array | NestedUint8Array): Buffer | NestedBufferArray;
/**
 * Converts a {@link Buffer} or {@link NestedBufferArray} to {@link Uint8Array} or {@link NestedUint8Array}
 */
export declare function bufArrToArr(arr: Buffer): Uint8Array;
export declare function bufArrToArr(arr: NestedBufferArray): NestedUint8Array;
export declare function bufArrToArr(arr: Buffer | NestedBufferArray): Uint8Array | NestedUint8Array;
/**
 * Converts a {@link bigint} to a `0x` prefixed hex string
 */
export declare const bigIntToHex: (num: bigint) => string;
/**
 * Convert value from bigint to an unpadded Buffer
 * (useful for RLP transport)
 * @param value value to convert
 */
export declare function bigIntToUnpaddedBuffer(value: bigint): Buffer;
export declare function intToUnpaddedBuffer(value: number): Buffer;
//# sourceMappingURL=bytes.d.ts.map