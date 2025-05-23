/**
 * @module Utils
 */
import { Numbers } from 'web3-types';
/**
 * Adds a padding on the left of a string, if value is a integer or bigInt will be converted to a hex string.
 * @param value - The value to be padded.
 * @param characterAmount - The amount of characters the string should have.
 * @param sign - The sign to be added (default is 0).
 * @returns The padded string.
 *
 * @example
 * ```ts
 *
 * console.log(web3.utils.padLeft('0x123', 10));
 * >0x0000000123
 * ```
 */
export declare const padLeft: (value: Numbers, characterAmount: number, sign?: string) => string;
/**
 * Adds a padding on the right of a string, if value is a integer or bigInt will be converted to a hex string.
 * @param value - The value to be padded.
 * @param characterAmount - The amount of characters the string should have.
 * @param sign - The sign to be added (default is 0).
 * @returns The padded string.
 *
 * @example
 * ```ts
 * console.log(web3.utils.padRight('0x123', 10));
 * > 0x1230000000
 *
 * console.log(web3.utils.padRight('0x123', 10, '1'));
 * > 0x1231111111
 * ```
 */
export declare const padRight: (value: Numbers, characterAmount: number, sign?: string) => string;
/**
 * Adds a padding on the right of a string, if value is a integer or bigInt will be converted to a hex string. @alias `padRight`
 */
export declare const rightPad: (value: Numbers, characterAmount: number, sign?: string) => string;
/**
 * Adds a padding on the left of a string, if value is a integer or bigInt will be converted to a hex string. @alias `padLeft`
 */
export declare const leftPad: (value: Numbers, characterAmount: number, sign?: string) => string;
/**
 * Converts a negative number into the two’s complement and return a hexstring of 64 nibbles.
 * @param value - The value to be converted.
 * @param nibbleWidth - The nibble width of the hex string (default is 64).
 *
 * @returns The hex string of the two’s complement.
 *
 * @example
 * ```ts
 * console.log(web3.utils.toTwosComplement(13, 32));
 * > 0x0000000000000000000000000000000d
 *
 * console.log(web3.utils.toTwosComplement('-0x1', 32));
 * > 0xffffffffffffffffffffffffffffffff
 *
 * console.log(web3.utils.toTwosComplement(BigInt('9007199254740992'), 32));
 * > 0x00000000000000000020000000000000
 * ```
 */
export declare const toTwosComplement: (value: Numbers, nibbleWidth?: number) => string;
/**
 * Converts the twos complement into a decimal number or big int.
 * @param value - The value to be converted.
 * @param nibbleWidth - The nibble width of the hex string (default is 64).
 * @returns The decimal number or big int.
 *
 * @example
 * ```ts
 * console.log(web3.utils.fromTwosComplement('0x0000000000000000000000000000000d', 32'));
 * > 13
 *
 * console.log(web3.utils.fromTwosComplement('0x00000000000000000020000000000000', 32));
 * > 9007199254740992n
 * ```
 */
export declare const fromTwosComplement: (value: Numbers, nibbleWidth?: number) => number | bigint;
//# sourceMappingURL=string_manipulation.d.ts.map