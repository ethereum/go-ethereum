"use strict";
/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
Object.defineProperty(exports, "__esModule", { value: true });
exports.toBool = exports.toChecksumAddress = exports.toWei = exports.fromWei = exports.toBigInt = exports.toNumber = exports.toHex = exports.toAscii = exports.hexToAscii = exports.fromAscii = exports.asciiToHex = exports.hexToString = exports.utf8ToBytes = exports.toUtf8 = exports.hexToUtf8 = exports.stringToHex = exports.fromUtf8 = exports.utf8ToHex = exports.hexToNumberString = exports.fromDecimal = exports.numberToHex = exports.toDecimal = exports.hexToNumber = exports.hexToBytes = exports.bytesToHex = exports.bytesToUint8Array = exports.ethUnitMap = void 0;
/**
 * @module Utils
 */
const keccak_js_1 = require("ethereum-cryptography/keccak.js");
const utils_js_1 = require("ethereum-cryptography/utils.js");
const web3_validator_1 = require("web3-validator");
const web3_errors_1 = require("web3-errors");
const uint8array_js_1 = require("./uint8array.js");
// Ref: https://ethdocs.org/en/latest/ether.html
// Note: this could be simplified using ** operator, but babel does not handle it well (https://github.com/babel/babel/issues/13109)
/** @internal */
exports.ethUnitMap = {
    noether: BigInt(0),
    wei: BigInt(1),
    kwei: BigInt(1000),
    Kwei: BigInt(1000),
    babbage: BigInt(1000),
    femtoether: BigInt(1000),
    mwei: BigInt(1000000),
    Mwei: BigInt(1000000),
    lovelace: BigInt(1000000),
    picoether: BigInt(1000000),
    gwei: BigInt(1000000000),
    Gwei: BigInt(1000000000),
    shannon: BigInt(1000000000),
    nanoether: BigInt(1000000000),
    nano: BigInt(1000000000),
    szabo: BigInt(1000000000000),
    microether: BigInt(1000000000000),
    micro: BigInt(1000000000000),
    finney: BigInt(1000000000000000),
    milliether: BigInt(1000000000000000),
    milli: BigInt(1000000000000000),
    ether: BigInt('1000000000000000000'),
    kether: BigInt('1000000000000000000000'),
    grand: BigInt('1000000000000000000000'),
    mether: BigInt('1000000000000000000000000'),
    gether: BigInt('1000000000000000000000000000'),
    tether: BigInt('1000000000000000000000000000000'),
};
const PrecisionLossWarning = 'Warning: Using type `number` with values that are large or contain many decimals may cause loss of precision, it is recommended to use type `string` or `BigInt` when using conversion methods';
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
const bytesToUint8Array = (data) => {
    web3_validator_1.validator.validate(['bytes'], [data]);
    if ((0, uint8array_js_1.isUint8Array)(data)) {
        return data;
    }
    if (Array.isArray(data)) {
        return new Uint8Array(data);
    }
    if (typeof data === 'string') {
        return web3_validator_1.utils.hexToUint8Array(data);
    }
    throw new web3_errors_1.InvalidBytesError(data);
};
exports.bytesToUint8Array = bytesToUint8Array;
/**
 * @internal
 */
const { uint8ArrayToHexString } = web3_validator_1.utils;
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
const bytesToHex = (bytes) => uint8ArrayToHexString((0, exports.bytesToUint8Array)(bytes));
exports.bytesToHex = bytesToHex;
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
const hexToBytes = (bytes) => {
    if (typeof bytes === 'string' && bytes.slice(0, 2).toLowerCase() !== '0x') {
        return (0, exports.bytesToUint8Array)(`0x${bytes}`);
    }
    return (0, exports.bytesToUint8Array)(bytes);
};
exports.hexToBytes = hexToBytes;
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
const hexToNumber = (value) => {
    web3_validator_1.validator.validate(['hex'], [value]);
    // To avoid duplicate code and circular dependency we will
    // use `hexToNumber` implementation from `web3-validator`
    return web3_validator_1.utils.hexToNumber(value);
};
exports.hexToNumber = hexToNumber;
/**
 * Converts value to it's number representation @alias `hexToNumber`
 */
exports.toDecimal = exports.hexToNumber;
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
const numberToHex = (value, hexstrict) => {
    if (typeof value !== 'bigint')
        web3_validator_1.validator.validate(['int'], [value]);
    // To avoid duplicate code and circular dependency we will
    // use `numberToHex` implementation from `web3-validator`
    let updatedValue = web3_validator_1.utils.numberToHex(value);
    if (hexstrict) {
        if (!updatedValue.startsWith('-') && updatedValue.length % 2 === 1) {
            // To avoid duplicate a circular dependency we will not be using the padLeft method
            updatedValue = '0x0'.concat(updatedValue.slice(2));
        }
        else if (updatedValue.length % 2 === 0 && updatedValue.startsWith('-'))
            updatedValue = '-0x0'.concat(updatedValue.slice(3));
    }
    return updatedValue;
};
exports.numberToHex = numberToHex;
/**
 * Converts value to it's hex representation @alias `numberToHex`
 *
 */
exports.fromDecimal = exports.numberToHex;
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
const hexToNumberString = (data) => (0, exports.hexToNumber)(data).toString();
exports.hexToNumberString = hexToNumberString;
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
const utf8ToHex = (str) => {
    web3_validator_1.validator.validate(['string'], [str]);
    // To be compatible with 1.x trim null character
    // eslint-disable-next-line no-control-regex
    let strWithoutNullCharacter = str.replace(/^(?:\u0000)/, '');
    // eslint-disable-next-line no-control-regex
    strWithoutNullCharacter = strWithoutNullCharacter.replace(/(?:\u0000)$/, '');
    return (0, exports.bytesToHex)(new TextEncoder().encode(strWithoutNullCharacter));
};
exports.utf8ToHex = utf8ToHex;
/**
 * @alias utf8ToHex
 */
exports.fromUtf8 = exports.utf8ToHex;
/**
 * @alias utf8ToHex
 */
exports.stringToHex = exports.utf8ToHex;
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
const hexToUtf8 = (str) => (0, utils_js_1.bytesToUtf8)((0, exports.hexToBytes)(str));
exports.hexToUtf8 = hexToUtf8;
/**
 * @alias hexToUtf8
 */
const toUtf8 = (input) => {
    if (typeof input === 'string') {
        return (0, exports.hexToUtf8)(input);
    }
    web3_validator_1.validator.validate(['bytes'], [input]);
    return (0, utils_js_1.bytesToUtf8)(input);
};
exports.toUtf8 = toUtf8;
exports.utf8ToBytes = utils_js_1.utf8ToBytes;
/**
 * @alias hexToUtf8
 */
exports.hexToString = exports.hexToUtf8;
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
const asciiToHex = (str) => {
    web3_validator_1.validator.validate(['string'], [str]);
    let hexString = '';
    for (let i = 0; i < str.length; i += 1) {
        const hexCharCode = str.charCodeAt(i).toString(16);
        // might need a leading 0
        hexString += hexCharCode.length % 2 !== 0 ? `0${hexCharCode}` : hexCharCode;
    }
    return `0x${hexString}`;
};
exports.asciiToHex = asciiToHex;
/**
 * @alias asciiToHex
 */
exports.fromAscii = exports.asciiToHex;
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
const hexToAscii = (str) => {
    const decoder = new TextDecoder('ascii');
    return decoder.decode((0, exports.hexToBytes)(str));
};
exports.hexToAscii = hexToAscii;
/**
 * @alias hexToAscii
 */
exports.toAscii = exports.hexToAscii;
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
const toHex = (value, returnType) => {
    if (typeof value === 'string' && (0, web3_validator_1.isAddress)(value)) {
        return returnType ? 'address' : `0x${value.toLowerCase().replace(/^0x/i, '')}`;
    }
    if (typeof value === 'boolean') {
        // eslint-disable-next-line no-nested-ternary
        return returnType ? 'bool' : value ? '0x01' : '0x00';
    }
    if (typeof value === 'number') {
        // eslint-disable-next-line no-nested-ternary
        return returnType ? (value < 0 ? 'int256' : 'uint256') : (0, exports.numberToHex)(value);
    }
    if (typeof value === 'bigint') {
        return returnType ? 'bigint' : (0, exports.numberToHex)(value);
    }
    if ((0, uint8array_js_1.isUint8Array)(value)) {
        return returnType ? 'bytes' : (0, exports.bytesToHex)(value);
    }
    if (typeof value === 'object' && !!value) {
        return returnType ? 'string' : (0, exports.utf8ToHex)(JSON.stringify(value));
    }
    if (typeof value === 'string') {
        if (value.startsWith('-0x') || value.startsWith('-0X')) {
            return returnType ? 'int256' : (0, exports.numberToHex)(value);
        }
        if ((0, web3_validator_1.isHexStrict)(value)) {
            return returnType ? 'bytes' : value;
        }
        if ((0, web3_validator_1.isHex)(value) && !(0, web3_validator_1.isInt)(value) && !(0, web3_validator_1.isUInt)(value)) {
            return returnType ? 'bytes' : `0x${value}`;
        }
        if ((0, web3_validator_1.isHex)(value) && !(0, web3_validator_1.isInt)(value) && (0, web3_validator_1.isUInt)(value)) {
            // This condition seems problematic because meeting
            // both conditions `!isInt(value) && isUInt(value)` should be impossible.
            // But a value pass for those conditions: "101611154195520776335741463917853444671577865378275924493376429267637792638729"
            // Note that according to the docs: it is supposed to be treated as a string (https://docs.web3js.org/guides/web3_upgrade_guide/x/web3_utils_migration_guide#conversion-to-hex)
            // In short, the strange is that isInt(value) is false but isUInt(value) is true for the value above.
            // TODO: isUInt(value) should be investigated.
            // However, if `toHex('101611154195520776335741463917853444671577865378275924493376429267637792638729', true)` is called, it will return `true`.
            // But, if `toHex('101611154195520776335741463917853444671577865378275924493376429267637792638729')` is called, it will throw inside `numberToHex`.
            return returnType ? 'uint' : (0, exports.numberToHex)(value);
        }
        if (!Number.isFinite(value)) {
            return returnType ? 'string' : (0, exports.utf8ToHex)(value);
        }
    }
    throw new web3_errors_1.HexProcessingError(value);
};
exports.toHex = toHex;
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
const toNumber = (value) => {
    if (typeof value === 'number') {
        if (value > 1e20) {
            console.warn(PrecisionLossWarning);
            // JavaScript converts numbers >= 10^21 to scientific notation when coerced to strings,
            // leading to potential parsing errors and incorrect representations.
            // For instance, String(10000000000000000000000) yields '1e+22'.
            // Using BigInt prevents this
            return BigInt(value);
        }
        return value;
    }
    if (typeof value === 'bigint') {
        return value >= Number.MIN_SAFE_INTEGER && value <= Number.MAX_SAFE_INTEGER
            ? Number(value)
            : value;
    }
    if (typeof value === 'string' && (0, web3_validator_1.isHexStrict)(value)) {
        return (0, exports.hexToNumber)(value);
    }
    try {
        return (0, exports.toNumber)(BigInt(value));
    }
    catch (_a) {
        throw new web3_errors_1.InvalidNumberError(value);
    }
};
exports.toNumber = toNumber;
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
const toBigInt = (value) => {
    if (typeof value === 'number') {
        return BigInt(value);
    }
    if (typeof value === 'bigint') {
        return value;
    }
    // isHex passes for dec, too
    if (typeof value === 'string' && (0, web3_validator_1.isHex)(value)) {
        if (value.startsWith('-')) {
            return -BigInt(value.substring(1));
        }
        return BigInt(value);
    }
    throw new web3_errors_1.InvalidNumberError(value);
};
exports.toBigInt = toBigInt;
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
const fromWei = (number, unit) => {
    let denomination;
    if (typeof unit === 'string') {
        denomination = exports.ethUnitMap[unit];
        if (!denomination) {
            throw new web3_errors_1.InvalidUnitError(unit);
        }
    }
    else {
        if (unit < 0 || !Number.isInteger(unit)) {
            throw new web3_errors_1.InvalidIntegerError(unit);
        }
        denomination = (0, web3_validator_1.bigintPower)(BigInt(10), BigInt(unit));
    }
    // value in wei would always be integer
    // 13456789, 1234
    const value = String((0, exports.toNumber)(number));
    // count number of zeros in denomination
    // 1000000 -> 6
    const numberOfZerosInDenomination = denomination.toString().length - 1;
    if (numberOfZerosInDenomination <= 0) {
        return value.toString();
    }
    // pad the value with required zeros
    // 13456789 -> 13456789, 1234 -> 001234
    const zeroPaddedValue = value.padStart(numberOfZerosInDenomination, '0');
    // get the integer part of value by counting number of zeros from start
    // 13456789 -> '13'
    // 001234 -> ''
    const integer = zeroPaddedValue.slice(0, -numberOfZerosInDenomination);
    // get the fraction part of value by counting number of zeros backward
    // 13456789 -> '456789'
    // 001234 -> '001234'
    const fraction = zeroPaddedValue.slice(-numberOfZerosInDenomination).replace(/\.?0+$/, '');
    if (integer === '') {
        return fraction ? `0.${fraction}` : '0';
    }
    if (fraction === '') {
        return integer;
    }
    const updatedValue = `${integer}.${fraction}`;
    return updatedValue.slice(0, integer.length + numberOfZerosInDenomination + 1);
};
exports.fromWei = fromWei;
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
// todo in 1.x unit defaults to 'ether'
const toWei = (number, unit) => {
    web3_validator_1.validator.validate(['number'], [number]);
    let denomination;
    if (typeof unit === 'string') {
        denomination = exports.ethUnitMap[unit];
        if (!denomination) {
            throw new web3_errors_1.InvalidUnitError(unit);
        }
    }
    else {
        if (unit < 0 || !Number.isInteger(unit)) {
            throw new web3_errors_1.InvalidIntegerError(unit);
        }
        denomination = (0, web3_validator_1.bigintPower)(BigInt(10), BigInt(unit));
    }
    let parsedNumber = number;
    if (typeof parsedNumber === 'number') {
        if (parsedNumber < 1e-15) {
            console.warn(PrecisionLossWarning);
        }
        if (parsedNumber > 1e20) {
            console.warn(PrecisionLossWarning);
            parsedNumber = BigInt(parsedNumber);
        }
        else {
            // in case there is a decimal point, we need to convert it to string
            parsedNumber = parsedNumber.toLocaleString('fullwide', {
                useGrouping: false,
                maximumFractionDigits: 20,
            });
        }
    }
    // if value is decimal e.g. 24.56 extract `integer` and `fraction` part
    // to avoid `fraction` to be null use `concat` with empty string
    const [integer, fraction] = String(typeof parsedNumber === 'string' && !(0, web3_validator_1.isHexStrict)(parsedNumber)
        ? parsedNumber
        : (0, exports.toNumber)(parsedNumber))
        .split('.')
        .concat('');
    // join the value removing `.` from
    // 24.56 -> 2456
    const value = BigInt(`${integer}${fraction}`);
    // multiply value with denomination
    // 2456 * 1000000 -> 2456000000
    const updatedValue = value * denomination;
    // check if whole number was passed in
    const decimals = fraction.length;
    if (decimals === 0) {
        return updatedValue.toString();
    }
    // trim the value to remove extra zeros
    return updatedValue.toString().slice(0, -decimals);
};
exports.toWei = toWei;
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
const toChecksumAddress = (address) => {
    if (!(0, web3_validator_1.isAddress)(address, false)) {
        throw new web3_errors_1.InvalidAddressError(address);
    }
    const lowerCaseAddress = address.toLowerCase().replace(/^0x/i, '');
    // calling `Uint8Array.from` because `noble-hashes` checks with `instanceof Uint8Array` that fails in some edge cases:
    // 	https://github.com/paulmillr/noble-hashes/issues/25#issuecomment-1750106284
    const hash = web3_validator_1.utils.uint8ArrayToHexString((0, keccak_js_1.keccak256)(web3_validator_1.utils.ensureIfUint8Array((0, exports.utf8ToBytes)(lowerCaseAddress))));
    if ((0, web3_validator_1.isNullish)(hash) ||
        hash === '0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470')
        return ''; // // EIP-1052 if hash is equal to c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470, keccak was given empty data
    let checksumAddress = '0x';
    const addressHash = hash.replace(/^0x/i, '');
    for (let i = 0; i < lowerCaseAddress.length; i += 1) {
        // If ith character is 8 to f then make it uppercase
        if (parseInt(addressHash[i], 16) > 7) {
            checksumAddress += lowerCaseAddress[i].toUpperCase();
        }
        else {
            checksumAddress += lowerCaseAddress[i];
        }
    }
    return checksumAddress;
};
exports.toChecksumAddress = toChecksumAddress;
const toBool = (value) => {
    if (typeof value === 'boolean') {
        return value;
    }
    if (typeof value === 'number' && (value === 0 || value === 1)) {
        return Boolean(value);
    }
    if (typeof value === 'bigint' && (value === BigInt(0) || value === BigInt(1))) {
        return Boolean(value);
    }
    if (typeof value === 'string' &&
        !(0, web3_validator_1.isHexStrict)(value) &&
        (value === '1' || value === '0' || value === 'false' || value === 'true')) {
        if (value === 'true') {
            return true;
        }
        if (value === 'false') {
            return false;
        }
        return Boolean(Number(value));
    }
    if (typeof value === 'string' && (0, web3_validator_1.isHexStrict)(value) && (value === '0x1' || value === '0x0')) {
        return Boolean((0, exports.toNumber)(value));
    }
    throw new web3_errors_1.InvalidBooleanError(value);
};
exports.toBool = toBool;
//# sourceMappingURL=converters.js.map