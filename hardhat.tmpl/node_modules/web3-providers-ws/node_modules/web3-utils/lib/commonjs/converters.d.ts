import { utf8ToBytes as ecUtf8ToBytes } from 'ethereum-cryptography/utils.js';
import { Address, Bytes, HexString, Numbers, ValueTypes } from 'web3-types';
/** @internal */
export declare const ethUnitMap: {
    noether: bigint;
    wei: bigint;
    kwei: bigint;
    Kwei: bigint;
    babbage: bigint;
    femtoether: bigint;
    mwei: bigint;
    Mwei: bigint;
    lovelace: bigint;
    picoether: bigint;
    gwei: bigint;
    Gwei: bigint;
    shannon: bigint;
    nanoether: bigint;
    nano: bigint;
    szabo: bigint;
    microether: bigint;
    micro: bigint;
    finney: bigint;
    milliether: bigint;
    milli: bigint;
    ether: bigint;
    kether: bigint;
    grand: bigint;
    mether: bigint;
    gether: bigint;
    tether: bigint;
};
export type EtherUnits = keyof typeof ethUnitMap;
/**
 * Convert a value from bytes to Uint8Array
 * @param data - Data to be converted
 * @returns - The Uint8Array representation of the input data
 *
 * @example
 * ```ts
 * console.log(web3.utils.bytesToUint8Array("0xab")));
 * > Uint8Array(1) [ 171 ]
 * ```
 */
export declare const bytesToUint8Array: (data: Bytes) => Uint8Array | never;
/**
 * Convert a byte array to a hex string
 * @param bytes - Byte array to be converted
 * @returns - The hex string representation of the input byte array
 *
 * @example
 * ```ts
 * console.log(web3.utils.bytesToHex(new Uint8Array([72, 12])));
 * > "0x480c"
 *
 */
export declare const bytesToHex: (bytes: Bytes) => HexString;
/**
 * Convert a hex string to a byte array
 * @param hex - Hex string to be converted
 * @returns - The byte array representation of the input hex string
 *
 * @example
 * ```ts
 * console.log(web3.utils.hexToBytes('0x74657374'));
 * > Uint8Array(4) [ 116, 101, 115, 116 ]
 * ```
 */
export declare const hexToBytes: (bytes: HexString) => Uint8Array;
/**
 * Converts value to it's number representation
 * @param value - Hex string to be converted
 * @returns - The number representation of the input value
 *
 * @example
 * ```ts
 * conoslle.log(web3.utils.hexToNumber('0xa'));
 * > 10
 * ```
 */
export declare const hexToNumber: (value: HexString) => bigint | number;
/**
 * Converts value to it's number representation @alias `hexToNumber`
 */
export declare const toDecimal: (value: HexString) => bigint | number;
/**
 * Converts value to it's hex representation
 * @param value - Value to be converted
 * @param hexstrict - Add padding to converted value if odd, to make it hexstrict
 * @returns - The hex representation of the input value
 *
 * @example
 * ```ts
 * console.log(web3.utils.numberToHex(10));
 * > "0xa"
 * ```
 */
export declare const numberToHex: (value: Numbers, hexstrict?: boolean) => HexString;
/**
 * Converts value to it's hex representation @alias `numberToHex`
 *
 */
export declare const fromDecimal: (value: Numbers, hexstrict?: boolean) => HexString;
/**
 * Converts value to it's decimal representation in string
 * @param value - Hex string to be converted
 * @returns - The decimal representation of the input value
 *
 * @example
 * ```ts
 * console.log(web3.utils.hexToNumberString('0xa'));
 * > "10"
 * ```
 */
export declare const hexToNumberString: (data: HexString) => string;
/**
 * Should be called to get hex representation (prefixed by 0x) of utf8 string
 * @param str - Utf8 string to be converted
 * @returns - The hex representation of the input string
 *
 * @example
 * ```ts
 * console.log(utf8ToHex('web3.js'));
 * > "0x776562332e6a73"
 * ```
 *
 */
export declare const utf8ToHex: (str: string) => HexString;
/**
 * @alias utf8ToHex
 */
export declare const fromUtf8: (str: string) => HexString;
/**
 * @alias utf8ToHex
 */
export declare const stringToHex: (str: string) => HexString;
/**
 * Should be called to get utf8 from it's hex representation
 * @param str - Hex string to be converted
 * @returns - Utf8 string
 *
 * @example
 * ```ts
 * console.log(web3.utils.hexToUtf8('0x48656c6c6f20576f726c64'));
 * > Hello World
 * ```
 */
export declare const hexToUtf8: (str: HexString) => string;
/**
 * @alias hexToUtf8
 */
export declare const toUtf8: (input: HexString | Uint8Array) => string;
export declare const utf8ToBytes: typeof ecUtf8ToBytes;
/**
 * @alias hexToUtf8
 */
export declare const hexToString: (str: HexString) => string;
/**
 * Should be called to get hex representation (prefixed by 0x) of ascii string
 * @param str - String to be converted to hex
 * @returns - Hex string
 *
 * @example
 * ```ts
 * console.log(web3.utils.asciiToHex('Hello World'));
 * > 0x48656c6c6f20576f726c64
 * ```
 */
export declare const asciiToHex: (str: string) => HexString;
/**
 * @alias asciiToHex
 */
export declare const fromAscii: (str: string) => HexString;
/**
 * Should be called to get ascii from it's hex representation
 * @param str - Hex string to be converted to ascii
 * @returns - Ascii string
 *
 * @example
 * ```ts
 * console.log(web3.utils.hexToAscii('0x48656c6c6f20576f726c64'));
 * > Hello World
 * ```
 */
export declare const hexToAscii: (str: HexString) => string;
/**
 * @alias hexToAscii
 */
export declare const toAscii: (str: HexString) => string;
/**
 * Auto converts any given value into it's hex representation.
 * @param value - Value to be converted to hex
 * @param returnType - If true, it will return the type of the value
 *
 * @example
 * ```ts
 * console.log(web3.utils.toHex(10));
 * > 0xa
 *
 * console.log(web3.utils.toHex('0x123', true));
 * > bytes
 *```
 */
export declare const toHex: (value: Numbers | Bytes | Address | boolean | object, returnType?: boolean) => HexString | ValueTypes;
/**
 * Converts any given value into it's number representation, if possible, else into it's bigint representation.
 * @param value - The value to convert
 * @returns - Returns the value in number or bigint representation
 *
 * @example
 * ```ts
 * console.log(web3.utils.toNumber(1));
 * > 1
 * console.log(web3.utils.toNumber(Number.MAX_SAFE_INTEGER));
 * > 9007199254740991
 *
 * console.log(web3.utils.toNumber(BigInt(Number.MAX_SAFE_INTEGER)));
 * > 9007199254740991
 *
 * console.log(web3.utils.toNumber(BigInt(Number.MAX_SAFE_INTEGER) + BigInt(1)));
 * > 9007199254740992n
 *
 * ```
 */
export declare const toNumber: (value: Numbers) => number | bigint;
/**
 * Auto converts any given value into it's bigint representation
 *
 * @param value - The value to convert
 * @returns - Returns the value in bigint representation

 * @example
 * ```ts
 * console.log(web3.utils.toBigInt(1));
 * > 1n
 * ```
 */
export declare const toBigInt: (value: unknown) => bigint;
/**
 * Takes a number of wei and converts it to any other ether unit.
 * @param number - The value in wei
 * @param unit - The unit to convert to
 * @returns - Returns the converted value in the given unit
 *
 * @example
 * ```ts
 * console.log(web3.utils.fromWei("1", "ether"));
 * > 0.000000000000000001
 *
 * console.log(web3.utils.fromWei("1", "shannon"));
 * > 0.000000001
 * ```
 */
export declare const fromWei: (number: Numbers, unit: EtherUnits | number) => string;
/**
 * Takes a number of a unit and converts it to wei.
 *
 * @param number - The number to convert.
 * @param unit - {@link EtherUnits} The unit of the number passed.
 * @returns The number converted to wei.
 *
 * @example
 * ```ts
 * console.log(web3.utils.toWei("0.001", "ether"));
 * > 1000000000000000 //(wei)
 * ```
 */
export declare const toWei: (number: Numbers, unit: EtherUnits | number) => string;
/**
 * Will convert an upper or lowercase Ethereum address to a checksum address.
 * @param address - An address string
 * @returns	The checksum address
 * @example
 * ```ts
 * web3.utils.toChecksumAddress('0xc1912fee45d61c87cc5ea59dae31190fffff232d');
 * > "0xc1912fEE45d61C87Cc5EA59DaE31190FFFFf232d"
 * ```
 */
export declare const toChecksumAddress: (address: Address) => string;
export declare const toBool: (value: boolean | string | number | unknown) => boolean;
