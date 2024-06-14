/**
 * Returns a `Boolean` on whether or not the a `String` starts with '0x'
 * @param str the string input value
 * @return a boolean if it is or is not hex prefixed
 * @throws if the str input is not a string
 */
export declare function isHexPrefixed(str: string): boolean;
/**
 * Removes '0x' from a given `String` if present
 * @param str the string value
 * @returns the string without 0x prefix
 */
export declare const stripHexPrefix: (str: string) => string;
/**
 * Pads a `String` to have an even length
 * @param value
 * @return output
 */
export declare function padToEven(value: string): string;
/**
 * Get the binary size of a string
 * @param str
 * @returns the number of bytes contained within the string
 */
export declare function getBinarySize(str: string): number;
/**
 * Returns TRUE if the first specified array contains all elements
 * from the second one. FALSE otherwise.
 *
 * @param superset
 * @param subset
 *
 */
export declare function arrayContainsArray(superset: unknown[], subset: unknown[], some?: boolean): boolean;
/**
 * Should be called to get ascii from its hex representation
 *
 * @param string in hex
 * @returns ascii string representation of hex value
 */
export declare function toAscii(hex: string): string;
/**
 * Should be called to get hex representation (prefixed by 0x) of utf8 string.
 * Strips leading and trailing 0's.
 *
 * @param string
 * @param optional padding
 * @returns hex representation of input string
 */
export declare function fromUtf8(stringValue: string): string;
/**
 * Should be called to get hex representation (prefixed by 0x) of ascii string
 *
 * @param  string
 * @param  optional padding
 * @returns  hex representation of input string
 */
export declare function fromAscii(stringValue: string): string;
/**
 * Returns the keys from an array of objects.
 * @example
 * ```js
 * getKeys([{a: '1', b: '2'}, {a: '3', b: '4'}], 'a') => ['1', '3']
 *````
 * @param  params
 * @param  key
 * @param  allowEmpty
 * @returns output just a simple array of output keys
 */
export declare function getKeys(params: Record<string, string>[], key: string, allowEmpty?: boolean): string[];
/**
 * Is the string a hex string.
 *
 * @param  value
 * @param  length
 * @returns  output the string is a hex string
 */
export declare function isHexString(value: string, length?: number): boolean;
//# sourceMappingURL=internal.d.ts.map