import type { PrefixedHexString, TransformabletoBytes } from './types.js';
export declare function _bytesToUnprefixedHex(bytes: Uint8Array): string;
/**
 * @deprecated
 */
export declare const bytesToUnprefixedHex: typeof _bytesToUnprefixedHex;
/**
 * @deprecated
 */
export declare const unprefixedHexToBytes: (inp: string) => Uint8Array;
export declare const bytesToHex: (bytes: Uint8Array) => string;
/**
 * Converts a {@link Uint8Array} to a {@link bigint}
 * @param {Uint8Array} bytes the bytes to convert
 * @returns {bigint}
 */
export declare const bytesToBigInt: (bytes: Uint8Array, littleEndian?: boolean) => bigint;
/**
 * Converts a {@link Uint8Array} to a {@link number}.
 * @param {Uint8Array} bytes the bytes to convert
 * @return  {number}
 * @throws If the input number exceeds 53 bits.
 */
export declare const bytesToInt: (bytes: Uint8Array) => number;
export declare const hexToBytes: (hex: string) => Uint8Array;
/******************************************/
/**
 * Converts a {@link number} into a {@link PrefixedHexString}
 * @param {number} i
 * @return {PrefixedHexString}
 */
export declare const intToHex: (i: number) => PrefixedHexString;
/**
 * Converts an {@link number} to a {@link Uint8Array}
 * @param {Number} i
 * @return {Uint8Array}
 */
export declare const intToBytes: (i: number) => Uint8Array;
/**
 * Converts a {@link bigint} to a {@link Uint8Array}
 *  * @param {bigint} num the bigint to convert
 * @returns {Uint8Array}
 */
export declare const bigIntToBytes: (num: bigint, littleEndian?: boolean) => Uint8Array;
/**
 * Returns a Uint8Array filled with 0s.
 * @param {number} bytes the number of bytes of the Uint8Array
 * @return {Uint8Array}
 */
export declare const zeros: (bytes: number) => Uint8Array;
/**
 * Left Pads a `Uint8Array` with leading zeros till it has `length` bytes.
 * Or it truncates the beginning if it exceeds.
 * @param {Uint8Array} msg the value to pad
 * @param {number} length the number of bytes the output should be
 * @return {Uint8Array}
 */
export declare const setLengthLeft: (msg: Uint8Array, length: number) => Uint8Array;
/**
 * Right Pads a `Uint8Array` with trailing zeros till it has `length` bytes.
 * it truncates the end if it exceeds.
 * @param {Uint8Array} msg the value to pad
 * @param {number} length the number of bytes the output should be
 * @return {Uint8Array}
 */
export declare const setLengthRight: (msg: Uint8Array, length: number) => Uint8Array;
/**
 * Trims leading zeros from a `Uint8Array`.
 * @param {Uint8Array} a
 * @return {Uint8Array}
 */
export declare const unpadBytes: (a: Uint8Array) => Uint8Array;
/**
 * Trims leading zeros from an `Array` (of numbers).
 * @param  {number[]} a
 * @return {number[]}
 */
export declare const unpadArray: (a: number[]) => number[];
/**
 * Trims leading zeros from a `PrefixedHexString`.
 * @param {PrefixedHexString} a
 * @return {PrefixedHexString}
 */
export declare const unpadHex: (a: string) => PrefixedHexString;
export declare type ToBytesInputTypes = PrefixedHexString | number | bigint | Uint8Array | number[] | TransformabletoBytes | null | undefined;
/**
 * Attempts to turn a value into a `Uint8Array`.
 * Inputs supported: `Buffer`, `Uint8Array`, `String` (hex-prefixed), `Number`, null/undefined, `BigInt` and other objects
 * with a `toArray()` or `toBytes()` method.
 * @param {ToBytesInputTypes} v the value
 * @return {Uint8Array}
 */
export declare const toBytes: (v: ToBytesInputTypes) => Uint8Array;
/**
 * Interprets a `Uint8Array` as a signed integer and returns a `BigInt`. Assumes 256-bit numbers.
 * @param {Uint8Array} num Signed integer value
 * @returns {bigint}
 */
export declare const fromSigned: (num: Uint8Array) => bigint;
/**
 * Converts a `BigInt` to an unsigned integer and returns it as a `Uint8Array`. Assumes 256-bit numbers.
 * @param {bigint} num
 * @returns {Uint8Array}
 */
export declare const toUnsigned: (num: bigint) => Uint8Array;
/**
 * Adds "0x" to a given `string` if it does not already start with "0x".
 * @param {string} str
 * @return {PrefixedHexString}
 */
export declare const addHexPrefix: (str: string) => PrefixedHexString;
/**
 * Shortens a string  or Uint8Array's hex string representation to maxLength (default 50).
 *
 * Examples:
 *
 * Input:  '657468657265756d000000000000000000000000000000000000000000000000'
 * Output: '657468657265756d0000000000000000000000000000000000…'
 * @param {Uint8Array | string} bytes
 * @param {number} maxLength
 * @return {string}
 */
export declare const short: (bytes: Uint8Array | string, maxLength?: number) => string;
/**
 * Checks provided Uint8Array for leading zeroes and throws if found.
 *
 * Examples:
 *
 * Valid values: 0x1, 0x, 0x01, 0x1234
 * Invalid values: 0x0, 0x00, 0x001, 0x0001
 *
 * Note: This method is useful for validating that RLP encoded integers comply with the rule that all
 * integer values encoded to RLP must be in the most compact form and contain no leading zero bytes
 * @param values An object containing string keys and Uint8Array values
 * @throws if any provided value is found to have leading zero bytes
 */
export declare const validateNoLeadingZeroes: (values: {
    [key: string]: Uint8Array | undefined;
}) => void;
/**
 * Converts a {@link bigint} to a `0x` prefixed hex string
 * @param {bigint} num the bigint to convert
 * @returns {PrefixedHexString}
 */
export declare const bigIntToHex: (num: bigint) => PrefixedHexString;
/**
 * Convert value from bigint to an unpadded Uint8Array
 * (useful for RLP transport)
 * @param {bigint} value the bigint to convert
 * @returns {Uint8Array}
 */
export declare const bigIntToUnpaddedBytes: (value: bigint) => Uint8Array;
/**
 * Convert value from number to an unpadded Uint8Array
 * (useful for RLP transport)
 * @param {number} value the bigint to convert
 * @returns {Uint8Array}
 */
export declare const intToUnpaddedBytes: (value: number) => Uint8Array;
/**
 * Compares two Uint8Arrays and returns a number indicating their order in a sorted array.
 *
 * @param {Uint8Array} value1 - The first Uint8Array to compare.
 * @param {Uint8Array} value2 - The second Uint8Array to compare.
 * @returns {number} A positive number if value1 is larger than value2,
 *                   A negative number if value1 is smaller than value2,
 *                   or 0 if value1 and value2 are equal.
 */
export declare const compareBytes: (value1: Uint8Array, value2: Uint8Array) => number;
/**
 * Generates a Uint8Array of random bytes of specified length.
 *
 * @param {number} length - The length of the Uint8Array.
 * @returns {Uint8Array} A Uint8Array of random bytes of specified length.
 */
export declare const randomBytes: (length: number) => Uint8Array;
/**
 * This mirrors the functionality of the `ethereum-cryptography` export except
 * it skips the check to validate that every element of `arrays` is indead a `uint8Array`
 * Can give small performance gains on large arrays
 * @param {Uint8Array[]} arrays an array of Uint8Arrays
 * @returns {Uint8Array} one Uint8Array with all the elements of the original set
 * works like `Buffer.concat`
 */
export declare const concatBytes: (...arrays: Uint8Array[]) => Uint8Array;
/**
 * @notice Convert a Uint8Array to a 32-bit integer
 * @param {Uint8Array} bytes The input Uint8Array from which to read the 32-bit integer.
 * @param {boolean} littleEndian True for little-endian, undefined or false for big-endian.
 * @return {number} The 32-bit integer read from the input Uint8Array.
 */
export declare function bytesToInt32(bytes: Uint8Array, littleEndian?: boolean): number;
/**
 * @notice Convert a Uint8Array to a 64-bit bigint
 * @param {Uint8Array} bytes The input Uint8Array from which to read the 64-bit bigint.
 * @param {boolean} littleEndian True for little-endian, undefined or false for big-endian.
 * @return {bigint} The 64-bit bigint read from the input Uint8Array.
 */
export declare function bytesToBigInt64(bytes: Uint8Array, littleEndian?: boolean): bigint;
/**
 * @notice Convert a 32-bit integer to a Uint8Array.
 * @param {number} value The 32-bit integer to convert.
 * @param {boolean} littleEndian True for little-endian, undefined or false for big-endian.
 * @return {Uint8Array} A Uint8Array of length 4 containing the integer.
 */
export declare function int32ToBytes(value: number, littleEndian?: boolean): Uint8Array;
/**
 * @notice Convert a 64-bit bigint to a Uint8Array.
 * @param {bigint} value The 64-bit bigint to convert.
 * @param {boolean} littleEndian True for little-endian, undefined or false for big-endian.
 * @return {Uint8Array} A Uint8Array of length 8 containing the bigint.
 */
export declare function bigInt64ToBytes(value: bigint, littleEndian?: boolean): Uint8Array;
export declare function equalsBytes(a: Uint8Array, b: Uint8Array): boolean;
export declare function bytesToUtf8(data: Uint8Array): string;
export declare function utf8ToBytes(str: string): Uint8Array;
//# sourceMappingURL=bytes.d.ts.map