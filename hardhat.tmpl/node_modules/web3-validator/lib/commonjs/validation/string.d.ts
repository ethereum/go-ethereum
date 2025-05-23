import { ValidInputTypes } from '../types.js';
/**
 * checks input if typeof data is valid string input
 */
export declare const isString: (value: ValidInputTypes) => boolean;
export declare const isHexStrict: (hex: ValidInputTypes) => boolean;
/**
 * Is the string a hex string.
 *
 * @param  value
 * @param  length
 * @returns  output the string is a hex string
 */
export declare function isHexString(value: string, length?: number): boolean;
export declare const isHex: (hex: ValidInputTypes) => boolean;
export declare const isHexString8Bytes: (value: string, prefixed?: boolean) => boolean;
export declare const isHexString32Bytes: (value: string, prefixed?: boolean) => boolean;
/**
 * Returns a `Boolean` on whether or not the a `String` starts with '0x'
 * @param str the string input value
 * @return a boolean if it is or is not hex prefixed
 * @throws if the str input is not a string
 */
export declare function isHexPrefixed(str: string): boolean;
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
