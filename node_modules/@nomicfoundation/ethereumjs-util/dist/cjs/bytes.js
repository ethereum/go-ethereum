"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.utf8ToBytes = exports.bytesToUtf8 = exports.equalsBytes = exports.bigInt64ToBytes = exports.int32ToBytes = exports.bytesToBigInt64 = exports.bytesToInt32 = exports.concatBytes = exports.randomBytes = exports.compareBytes = exports.intToUnpaddedBytes = exports.bigIntToUnpaddedBytes = exports.bigIntToHex = exports.validateNoLeadingZeroes = exports.short = exports.addHexPrefix = exports.toUnsigned = exports.fromSigned = exports.toBytes = exports.unpadHex = exports.unpadArray = exports.unpadBytes = exports.setLengthRight = exports.setLengthLeft = exports.zeros = exports.bigIntToBytes = exports.intToBytes = exports.intToHex = exports.hexToBytes = exports.bytesToInt = exports.bytesToBigInt = exports.bytesToHex = exports.unprefixedHexToBytes = exports.bytesToUnprefixedHex = exports._bytesToUnprefixedHex = void 0;
const random_js_1 = require("ethereum-cryptography/random.js");
const helpers_js_1 = require("./helpers.js");
const internal_js_1 = require("./internal.js");
const BIGINT_0 = BigInt(0);
function isBytes(a) {
    return (a instanceof Uint8Array ||
        // eslint-disable-next-line eqeqeq
        (a != null && typeof a === 'object' && a.constructor.name === 'Uint8Array'));
}
const hexes = Array.from({ length: 256 }, (_, i) => i.toString(16).padStart(2, '0'));
function _bytesToUnprefixedHex(bytes) {
    if (!isBytes(bytes))
        throw new Error('Uint8Array expected');
    // pre-caching improves the speed 6x
    let hex = '';
    for (let i = 0; i < bytes.length; i++) {
        hex += hexes[bytes[i]];
    }
    return hex;
}
exports._bytesToUnprefixedHex = _bytesToUnprefixedHex;
/**
 * @deprecated
 */
exports.bytesToUnprefixedHex = _bytesToUnprefixedHex;
// hexToBytes cache
const hexToBytesMapFirstKey = {};
const hexToBytesMapSecondKey = {};
for (let i = 0; i < 16; i++) {
    const vSecondKey = i;
    const vFirstKey = i * 16;
    const key = i.toString(16).toLowerCase();
    hexToBytesMapSecondKey[key] = vSecondKey;
    hexToBytesMapSecondKey[key.toUpperCase()] = vSecondKey;
    hexToBytesMapFirstKey[key] = vFirstKey;
    hexToBytesMapFirstKey[key.toUpperCase()] = vFirstKey;
}
/**
 * NOTE: only use this function if the string is even, and only consists of hex characters
 * If this is not the case, this function could return weird results
 * @deprecated
 */
function _unprefixedHexToBytes(hex) {
    const byteLen = hex.length;
    const bytes = new Uint8Array(byteLen / 2);
    for (let i = 0; i < byteLen; i += 2) {
        bytes[i / 2] = hexToBytesMapFirstKey[hex[i]] + hexToBytesMapSecondKey[hex[i + 1]];
    }
    return bytes;
}
/**
 * @deprecated
 */
const unprefixedHexToBytes = (inp) => {
    if (inp.slice(0, 2) === '0x') {
        throw new Error('hex string is prefixed with 0x, should be unprefixed');
    }
    else {
        return _unprefixedHexToBytes((0, internal_js_1.padToEven)(inp));
    }
};
exports.unprefixedHexToBytes = unprefixedHexToBytes;
/****************  Borrowed from @chainsafe/ssz */
// Caching this info costs about ~1000 bytes and speeds up toHexString() by x6
const hexByByte = Array.from({ length: 256 }, (v, i) => i.toString(16).padStart(2, '0'));
const bytesToHex = (bytes) => {
    let hex = '0x';
    if (bytes === undefined || bytes.length === 0)
        return hex;
    for (const byte of bytes) {
        hex += hexByByte[byte];
    }
    return hex;
};
exports.bytesToHex = bytesToHex;
// BigInt cache for the numbers 0 - 256*256-1 (two-byte bytes)
const BIGINT_CACHE = [];
for (let i = 0; i <= 256 * 256 - 1; i++) {
    BIGINT_CACHE[i] = BigInt(i);
}
/**
 * Converts a {@link Uint8Array} to a {@link bigint}
 * @param {Uint8Array} bytes the bytes to convert
 * @returns {bigint}
 */
const bytesToBigInt = (bytes, littleEndian = false) => {
    if (littleEndian) {
        bytes.reverse();
    }
    const hex = (0, exports.bytesToHex)(bytes);
    if (hex === '0x') {
        return BIGINT_0;
    }
    if (hex.length === 4) {
        // If the byte length is 1 (this is faster than checking `bytes.length === 1`)
        return BIGINT_CACHE[bytes[0]];
    }
    if (hex.length === 6) {
        return BIGINT_CACHE[bytes[0] * 256 + bytes[1]];
    }
    return BigInt(hex);
};
exports.bytesToBigInt = bytesToBigInt;
/**
 * Converts a {@link Uint8Array} to a {@link number}.
 * @param {Uint8Array} bytes the bytes to convert
 * @return  {number}
 * @throws If the input number exceeds 53 bits.
 */
const bytesToInt = (bytes) => {
    const res = Number((0, exports.bytesToBigInt)(bytes));
    if (!Number.isSafeInteger(res))
        throw new Error('Number exceeds 53 bits');
    return res;
};
exports.bytesToInt = bytesToInt;
const hexToBytes = (hex) => {
    if (typeof hex !== 'string') {
        throw new Error(`hex argument type ${typeof hex} must be of type string`);
    }
    if (!/^0x[0-9a-fA-F]*$/.test(hex)) {
        throw new Error(`Input must be a 0x-prefixed hexadecimal string, got ${hex}`);
    }
    hex = hex.slice(2);
    if (hex.length % 2 !== 0) {
        hex = (0, internal_js_1.padToEven)(hex);
    }
    return _unprefixedHexToBytes(hex);
};
exports.hexToBytes = hexToBytes;
/******************************************/
/**
 * Converts a {@link number} into a {@link PrefixedHexString}
 * @param {number} i
 * @return {PrefixedHexString}
 */
const intToHex = (i) => {
    if (!Number.isSafeInteger(i) || i < 0) {
        throw new Error(`Received an invalid integer type: ${i}`);
    }
    return `0x${i.toString(16)}`;
};
exports.intToHex = intToHex;
/**
 * Converts an {@link number} to a {@link Uint8Array}
 * @param {Number} i
 * @return {Uint8Array}
 */
const intToBytes = (i) => {
    const hex = (0, exports.intToHex)(i);
    return (0, exports.hexToBytes)(hex);
};
exports.intToBytes = intToBytes;
/**
 * Converts a {@link bigint} to a {@link Uint8Array}
 *  * @param {bigint} num the bigint to convert
 * @returns {Uint8Array}
 */
const bigIntToBytes = (num, littleEndian = false) => {
    // eslint-disable-next-line @typescript-eslint/no-use-before-define
    const bytes = (0, exports.toBytes)('0x' + (0, internal_js_1.padToEven)(num.toString(16)));
    return littleEndian ? bytes.reverse() : bytes;
};
exports.bigIntToBytes = bigIntToBytes;
/**
 * Returns a Uint8Array filled with 0s.
 * @param {number} bytes the number of bytes of the Uint8Array
 * @return {Uint8Array}
 */
const zeros = (bytes) => {
    return new Uint8Array(bytes);
};
exports.zeros = zeros;
/**
 * Pads a `Uint8Array` with zeros till it has `length` bytes.
 * Truncates the beginning or end of input if its length exceeds `length`.
 * @param {Uint8Array} msg the value to pad
 * @param {number} length the number of bytes the output should be
 * @param {boolean} right whether to start padding form the left or right
 * @return {Uint8Array}
 */
const setLength = (msg, length, right) => {
    if (right) {
        if (msg.length < length) {
            return new Uint8Array([...msg, ...(0, exports.zeros)(length - msg.length)]);
        }
        return msg.subarray(0, length);
    }
    else {
        if (msg.length < length) {
            return new Uint8Array([...(0, exports.zeros)(length - msg.length), ...msg]);
        }
        return msg.subarray(-length);
    }
};
/**
 * Left Pads a `Uint8Array` with leading zeros till it has `length` bytes.
 * Or it truncates the beginning if it exceeds.
 * @param {Uint8Array} msg the value to pad
 * @param {number} length the number of bytes the output should be
 * @return {Uint8Array}
 */
const setLengthLeft = (msg, length) => {
    (0, helpers_js_1.assertIsBytes)(msg);
    return setLength(msg, length, false);
};
exports.setLengthLeft = setLengthLeft;
/**
 * Right Pads a `Uint8Array` with trailing zeros till it has `length` bytes.
 * it truncates the end if it exceeds.
 * @param {Uint8Array} msg the value to pad
 * @param {number} length the number of bytes the output should be
 * @return {Uint8Array}
 */
const setLengthRight = (msg, length) => {
    (0, helpers_js_1.assertIsBytes)(msg);
    return setLength(msg, length, true);
};
exports.setLengthRight = setLengthRight;
/**
 * Trims leading zeros from a `Uint8Array`, `number[]` or PrefixedHexString`.
 * @param {Uint8Array|number[]|PrefixedHexString} a
 * @return {Uint8Array|number[]|PrefixedHexString}
 */
const stripZeros = (a) => {
    let first = a[0];
    while (a.length > 0 && first.toString() === '0') {
        a = a.slice(1);
        first = a[0];
    }
    return a;
};
/**
 * Trims leading zeros from a `Uint8Array`.
 * @param {Uint8Array} a
 * @return {Uint8Array}
 */
const unpadBytes = (a) => {
    (0, helpers_js_1.assertIsBytes)(a);
    return stripZeros(a);
};
exports.unpadBytes = unpadBytes;
/**
 * Trims leading zeros from an `Array` (of numbers).
 * @param  {number[]} a
 * @return {number[]}
 */
const unpadArray = (a) => {
    (0, helpers_js_1.assertIsArray)(a);
    return stripZeros(a);
};
exports.unpadArray = unpadArray;
/**
 * Trims leading zeros from a `PrefixedHexString`.
 * @param {PrefixedHexString} a
 * @return {PrefixedHexString}
 */
const unpadHex = (a) => {
    (0, helpers_js_1.assertIsHexString)(a);
    a = (0, internal_js_1.stripHexPrefix)(a);
    return '0x' + stripZeros(a);
};
exports.unpadHex = unpadHex;
/**
 * Attempts to turn a value into a `Uint8Array`.
 * Inputs supported: `Buffer`, `Uint8Array`, `String` (hex-prefixed), `Number`, null/undefined, `BigInt` and other objects
 * with a `toArray()` or `toBytes()` method.
 * @param {ToBytesInputTypes} v the value
 * @return {Uint8Array}
 */
const toBytes = (v) => {
    if (v === null || v === undefined) {
        return new Uint8Array();
    }
    if (Array.isArray(v) || v instanceof Uint8Array) {
        return Uint8Array.from(v);
    }
    if (typeof v === 'string') {
        if (!(0, internal_js_1.isHexString)(v)) {
            throw new Error(`Cannot convert string to Uint8Array. toBytes only supports 0x-prefixed hex strings and this string was given: ${v}`);
        }
        return (0, exports.hexToBytes)(v);
    }
    if (typeof v === 'number') {
        return (0, exports.intToBytes)(v);
    }
    if (typeof v === 'bigint') {
        if (v < BIGINT_0) {
            throw new Error(`Cannot convert negative bigint to Uint8Array. Given: ${v}`);
        }
        let n = v.toString(16);
        if (n.length % 2)
            n = '0' + n;
        return (0, exports.unprefixedHexToBytes)(n);
    }
    if (v.toBytes !== undefined) {
        // converts a `TransformableToBytes` object to a Uint8Array
        return v.toBytes();
    }
    throw new Error('invalid type');
};
exports.toBytes = toBytes;
/**
 * Interprets a `Uint8Array` as a signed integer and returns a `BigInt`. Assumes 256-bit numbers.
 * @param {Uint8Array} num Signed integer value
 * @returns {bigint}
 */
const fromSigned = (num) => {
    return BigInt.asIntN(256, (0, exports.bytesToBigInt)(num));
};
exports.fromSigned = fromSigned;
/**
 * Converts a `BigInt` to an unsigned integer and returns it as a `Uint8Array`. Assumes 256-bit numbers.
 * @param {bigint} num
 * @returns {Uint8Array}
 */
const toUnsigned = (num) => {
    return (0, exports.bigIntToBytes)(BigInt.asUintN(256, num));
};
exports.toUnsigned = toUnsigned;
/**
 * Adds "0x" to a given `string` if it does not already start with "0x".
 * @param {string} str
 * @return {PrefixedHexString}
 */
const addHexPrefix = (str) => {
    if (typeof str !== 'string') {
        return str;
    }
    return (0, internal_js_1.isHexPrefixed)(str) ? str : '0x' + str;
};
exports.addHexPrefix = addHexPrefix;
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
const short = (bytes, maxLength = 50) => {
    const byteStr = bytes instanceof Uint8Array ? (0, exports.bytesToHex)(bytes) : bytes;
    const len = byteStr.slice(0, 2) === '0x' ? maxLength + 2 : maxLength;
    if (byteStr.length <= len) {
        return byteStr;
    }
    return byteStr.slice(0, len) + '…';
};
exports.short = short;
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
const validateNoLeadingZeroes = (values) => {
    for (const [k, v] of Object.entries(values)) {
        if (v !== undefined && v.length > 0 && v[0] === 0) {
            throw new Error(`${k} cannot have leading zeroes, received: ${(0, exports.bytesToHex)(v)}`);
        }
    }
};
exports.validateNoLeadingZeroes = validateNoLeadingZeroes;
/**
 * Converts a {@link bigint} to a `0x` prefixed hex string
 * @param {bigint} num the bigint to convert
 * @returns {PrefixedHexString}
 */
const bigIntToHex = (num) => {
    return '0x' + num.toString(16);
};
exports.bigIntToHex = bigIntToHex;
/**
 * Convert value from bigint to an unpadded Uint8Array
 * (useful for RLP transport)
 * @param {bigint} value the bigint to convert
 * @returns {Uint8Array}
 */
const bigIntToUnpaddedBytes = (value) => {
    return (0, exports.unpadBytes)((0, exports.bigIntToBytes)(value));
};
exports.bigIntToUnpaddedBytes = bigIntToUnpaddedBytes;
/**
 * Convert value from number to an unpadded Uint8Array
 * (useful for RLP transport)
 * @param {number} value the bigint to convert
 * @returns {Uint8Array}
 */
const intToUnpaddedBytes = (value) => {
    return (0, exports.unpadBytes)((0, exports.intToBytes)(value));
};
exports.intToUnpaddedBytes = intToUnpaddedBytes;
/**
 * Compares two Uint8Arrays and returns a number indicating their order in a sorted array.
 *
 * @param {Uint8Array} value1 - The first Uint8Array to compare.
 * @param {Uint8Array} value2 - The second Uint8Array to compare.
 * @returns {number} A positive number if value1 is larger than value2,
 *                   A negative number if value1 is smaller than value2,
 *                   or 0 if value1 and value2 are equal.
 */
const compareBytes = (value1, value2) => {
    const bigIntValue1 = (0, exports.bytesToBigInt)(value1);
    const bigIntValue2 = (0, exports.bytesToBigInt)(value2);
    return bigIntValue1 > bigIntValue2 ? 1 : bigIntValue1 < bigIntValue2 ? -1 : 0;
};
exports.compareBytes = compareBytes;
/**
 * Generates a Uint8Array of random bytes of specified length.
 *
 * @param {number} length - The length of the Uint8Array.
 * @returns {Uint8Array} A Uint8Array of random bytes of specified length.
 */
const randomBytes = (length) => {
    return (0, random_js_1.getRandomBytesSync)(length);
};
exports.randomBytes = randomBytes;
/**
 * This mirrors the functionality of the `ethereum-cryptography` export except
 * it skips the check to validate that every element of `arrays` is indead a `uint8Array`
 * Can give small performance gains on large arrays
 * @param {Uint8Array[]} arrays an array of Uint8Arrays
 * @returns {Uint8Array} one Uint8Array with all the elements of the original set
 * works like `Buffer.concat`
 */
const concatBytes = (...arrays) => {
    if (arrays.length === 1)
        return arrays[0];
    const length = arrays.reduce((a, arr) => a + arr.length, 0);
    const result = new Uint8Array(length);
    for (let i = 0, pad = 0; i < arrays.length; i++) {
        const arr = arrays[i];
        result.set(arr, pad);
        pad += arr.length;
    }
    return result;
};
exports.concatBytes = concatBytes;
/**
 * @notice Convert a Uint8Array to a 32-bit integer
 * @param {Uint8Array} bytes The input Uint8Array from which to read the 32-bit integer.
 * @param {boolean} littleEndian True for little-endian, undefined or false for big-endian.
 * @return {number} The 32-bit integer read from the input Uint8Array.
 */
function bytesToInt32(bytes, littleEndian = false) {
    if (bytes.length < 4) {
        bytes = setLength(bytes, 4, littleEndian);
    }
    const dataView = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
    return dataView.getUint32(0, littleEndian);
}
exports.bytesToInt32 = bytesToInt32;
/**
 * @notice Convert a Uint8Array to a 64-bit bigint
 * @param {Uint8Array} bytes The input Uint8Array from which to read the 64-bit bigint.
 * @param {boolean} littleEndian True for little-endian, undefined or false for big-endian.
 * @return {bigint} The 64-bit bigint read from the input Uint8Array.
 */
function bytesToBigInt64(bytes, littleEndian = false) {
    if (bytes.length < 8) {
        bytes = setLength(bytes, 8, littleEndian);
    }
    const dataView = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
    return dataView.getBigUint64(0, littleEndian);
}
exports.bytesToBigInt64 = bytesToBigInt64;
/**
 * @notice Convert a 32-bit integer to a Uint8Array.
 * @param {number} value The 32-bit integer to convert.
 * @param {boolean} littleEndian True for little-endian, undefined or false for big-endian.
 * @return {Uint8Array} A Uint8Array of length 4 containing the integer.
 */
function int32ToBytes(value, littleEndian = false) {
    const buffer = new ArrayBuffer(4);
    const dataView = new DataView(buffer);
    dataView.setUint32(0, value, littleEndian);
    return new Uint8Array(buffer);
}
exports.int32ToBytes = int32ToBytes;
/**
 * @notice Convert a 64-bit bigint to a Uint8Array.
 * @param {bigint} value The 64-bit bigint to convert.
 * @param {boolean} littleEndian True for little-endian, undefined or false for big-endian.
 * @return {Uint8Array} A Uint8Array of length 8 containing the bigint.
 */
function bigInt64ToBytes(value, littleEndian = false) {
    const buffer = new ArrayBuffer(8);
    const dataView = new DataView(buffer);
    dataView.setBigUint64(0, value, littleEndian);
    return new Uint8Array(buffer);
}
exports.bigInt64ToBytes = bigInt64ToBytes;
function equalsBytes(a, b) {
    if (a.length !== b.length) {
        return false;
    }
    for (let i = 0; i < a.length; i++) {
        if (a[i] !== b[i]) {
            return false;
        }
    }
    return true;
}
exports.equalsBytes = equalsBytes;
function bytesToUtf8(data) {
    if (!(data instanceof Uint8Array)) {
        throw new TypeError(`bytesToUtf8 expected Uint8Array, got ${typeof data}`);
    }
    return new TextDecoder().decode(data);
}
exports.bytesToUtf8 = bytesToUtf8;
function utf8ToBytes(str) {
    if (typeof str !== 'string')
        throw new Error(`utf8ToBytes expected string, got ${typeof str}`);
    return new Uint8Array(new TextEncoder().encode(str)); // https://bugzil.la/1681809
}
exports.utf8ToBytes = utf8ToBytes;
//# sourceMappingURL=bytes.js.map