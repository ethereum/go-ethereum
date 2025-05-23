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
exports.getStorageSlotNumForLongString = exports.soliditySha3Raw = exports.soliditySha3 = exports.encodePacked = exports.processSolidityEncodePackedArgs = exports.sha3Raw = exports.sha3 = exports.keccak256 = exports.keccak256Wrapper = void 0;
/**
 * This package provides utility functions for Ethereum dapps and other web3.js packages.
 *
 * For using Utils functions, first install Web3 package using `npm i web3` or `yarn add web3`.
 * After that, Web3 Utils functions will be available as mentioned below.
 * ```ts
 * import { Web3 } from 'web3';
 * const web3 = new Web3();
 *
 * const value = web3.utils.fromWei("1", "ether")
 *
 * ```
 *
 * For using individual package install `web3-utils` package using `npm i web3-utils` or `yarn add web3-utils` and only import required functions.
 * This is more efficient approach for building lightweight applications.
 * ```ts
 * import { fromWei, soliditySha3Raw } from 'web3-utils';
 *
 * console.log(fromWei("1", "ether"));
 * console.log(soliditySha3Raw({ type: "string", value: "helloworld" }))
 *
 * ```
 * @module Utils
 */
const keccak_js_1 = require("ethereum-cryptography/keccak.js");
const utils_js_1 = require("ethereum-cryptography/utils.js");
const web3_errors_1 = require("web3-errors");
const web3_validator_1 = require("web3-validator");
const converters_js_1 = require("./converters.js");
const string_manipulation_js_1 = require("./string_manipulation.js");
const SHA3_EMPTY_BYTES = '0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470';
/**
 * A wrapper for ethereum-cryptography/keccak256 to allow hashing a `string` and a `bigint` in addition to `UInt8Array`
 * @param data - the input to hash
 * @returns - the Keccak-256 hash of the input
 *
 * @example
 * ```ts
 * console.log(web3.utils.keccak256Wrapper('web3.js'));
 * > 0x63667efb1961039c9bb0d6ea7a5abdd223a3aca7daa5044ad894226e1f83919a
 *
 * console.log(web3.utils.keccak256Wrapper(1));
 * > 0xc89efdaa54c0f20c7adf612882df0950f5a951637e0307cdcb4c672f298b8bc6
 *
 * console.log(web3.utils.keccak256Wrapper(0xaf12fd));
 * > 0x358640fd4719fa923525d74ab5ae80a594301aba5543e3492b052bf4598b794c
 * ```
 */
const keccak256Wrapper = (data) => {
    let processedData;
    if (typeof data === 'bigint' || typeof data === 'number') {
        processedData = (0, utils_js_1.utf8ToBytes)(data.toString());
    }
    else if (Array.isArray(data)) {
        processedData = new Uint8Array(data);
    }
    else if (typeof data === 'string' && !(0, web3_validator_1.isHexStrict)(data)) {
        processedData = (0, utils_js_1.utf8ToBytes)(data);
    }
    else {
        processedData = (0, converters_js_1.bytesToUint8Array)(data);
    }
    return (0, converters_js_1.bytesToHex)((0, keccak_js_1.keccak256)(web3_validator_1.utils.ensureIfUint8Array(processedData)));
};
exports.keccak256Wrapper = keccak256Wrapper;
exports.keccak256 = exports.keccak256Wrapper;
/**
 * computes the Keccak-256 hash of the input and returns a hexstring
 * @param data - the input to hash
 * @returns - the Keccak-256 hash of the input
 *
 * @example
 * ```ts
 * console.log(web3.utils.sha3('web3.js'));
 * > 0x63667efb1961039c9bb0d6ea7a5abdd223a3aca7daa5044ad894226e1f83919a
 *
 * console.log(web3.utils.sha3(''));
 * > undefined
 * ```
 */
const sha3 = (data) => {
    let updatedData;
    if (typeof data === 'string') {
        if (data.startsWith('0x') && (0, web3_validator_1.isHexStrict)(data)) {
            updatedData = (0, converters_js_1.hexToBytes)(data);
        }
        else {
            updatedData = (0, utils_js_1.utf8ToBytes)(data);
        }
    }
    else {
        updatedData = data;
    }
    const hash = (0, exports.keccak256Wrapper)(updatedData);
    // EIP-1052 if hash is equal to c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470, keccak was given empty data
    return hash === SHA3_EMPTY_BYTES ? undefined : hash;
};
exports.sha3 = sha3;
/**
 * Will calculate the sha3 of the input but does return the hash value instead of null if for example a empty string is passed.
 * @param data - the input to hash
 * @returns - the Keccak-256 hash of the input
 *
 * @example
 * ```ts
 * conosle.log(web3.utils.sha3Raw('web3.js'));
 * > 0x63667efb1961039c9bb0d6ea7a5abdd223a3aca7daa5044ad894226e1f83919a
 *
 * console.log(web3.utils.sha3Raw(''));
 * > 0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470
 * ```
 */
const sha3Raw = (data) => {
    const hash = (0, exports.sha3)(data);
    if ((0, web3_validator_1.isNullish)(hash)) {
        return SHA3_EMPTY_BYTES;
    }
    return hash;
};
exports.sha3Raw = sha3Raw;
/**
 * returns type and value
 * @param arg - the input to return the type and value
 * @returns - the type and value of the input
 */
const getType = (arg) => {
    if (Array.isArray(arg)) {
        throw new Error('Autodetection of array types is not supported.');
    }
    let type;
    let value;
    // if type is given
    if (typeof arg === 'object' &&
        ('t' in arg || 'type' in arg) &&
        ('v' in arg || 'value' in arg)) {
        type = 't' in arg ? arg.t : arg.type;
        value = 'v' in arg ? arg.v : arg.value;
        type = type.toLowerCase() === 'bigint' ? 'int' : type;
    }
    else if (typeof arg === 'bigint') {
        return ['int', arg];
    }
    // otherwise try to guess the type
    else {
        type = (0, converters_js_1.toHex)(arg, true);
        value = (0, converters_js_1.toHex)(arg);
        if (!type.startsWith('int') && !type.startsWith('uint')) {
            type = 'bytes';
        }
    }
    if ((type.startsWith('int') || type.startsWith('uint')) &&
        typeof value === 'string' &&
        !/^(-)?0x/i.test(value)) {
        value = (0, converters_js_1.toBigInt)(value);
    }
    return [type, value];
};
/**
 * returns the type with size if uint or int
 * @param name - the input to return the type with size
 * @returns - the type with size of the input
 */
const elementaryName = (name) => {
    if (name.startsWith('int[')) {
        return `int256${name.slice(3)}`;
    }
    if (name === 'int') {
        return 'int256';
    }
    if (name.startsWith('uint[')) {
        return `uint256'${name.slice(4)}`;
    }
    if (name === 'uint') {
        return 'uint256';
    }
    return name;
};
/**
 * returns the size of the value of type 'byte'
 */
const parseTypeN = (value, typeLength) => {
    const typesize = /^(\d+).*$/.exec(value.slice(typeLength));
    return typesize ? parseInt(typesize[1], 10) : 0;
};
/**
 * returns the bit length of the value
 * @param value - the input to return the bit length
 * @returns - the bit length of the input
 */
const bitLength = (value) => {
    const updatedVal = value.toString(2);
    return updatedVal.length;
};
/**
 * Pads the value based on size and type
 * returns a string of the padded value
 * @param type - the input to pad
 * @returns = the padded value
 */
const solidityPack = (type, val) => {
    const value = val.toString();
    if (type === 'string') {
        if (typeof val === 'string')
            return (0, converters_js_1.utf8ToHex)(val);
        throw new web3_errors_1.InvalidStringError(val);
    }
    if (type === 'bool' || type === 'boolean') {
        if (typeof val === 'boolean')
            return val ? '01' : '00';
        throw new web3_errors_1.InvalidBooleanError(val);
    }
    if (type === 'address') {
        if (!(0, web3_validator_1.isAddress)(value)) {
            throw new web3_errors_1.InvalidAddressError(value);
        }
        return value;
    }
    const name = elementaryName(type);
    if (type.startsWith('uint')) {
        const size = parseTypeN(name, 'uint'.length);
        if (size % 8 || size < 8 || size > 256) {
            throw new web3_errors_1.InvalidSizeError(value);
        }
        const num = (0, converters_js_1.toNumber)(value);
        if (bitLength(num) > size) {
            throw new web3_errors_1.InvalidLargeValueError(value);
        }
        if (num < BigInt(0)) {
            throw new web3_errors_1.InvalidUnsignedIntegerError(value);
        }
        return size ? (0, string_manipulation_js_1.leftPad)(num.toString(16), (size / 8) * 2) : num.toString(16);
    }
    if (type.startsWith('int')) {
        const size = parseTypeN(name, 'int'.length);
        if (size % 8 || size < 8 || size > 256) {
            throw new web3_errors_1.InvalidSizeError(type);
        }
        const num = (0, converters_js_1.toNumber)(value);
        if (bitLength(num) > size) {
            throw new web3_errors_1.InvalidLargeValueError(value);
        }
        if (num < BigInt(0)) {
            return (0, string_manipulation_js_1.toTwosComplement)(num.toString(), (size / 8) * 2);
        }
        return size ? (0, string_manipulation_js_1.leftPad)(num.toString(16), size / 4) : num.toString(16);
    }
    if (name === 'bytes') {
        if (value.replace(/^0x/i, '').length % 2 !== 0) {
            throw new web3_errors_1.InvalidBytesError(value);
        }
        return value;
    }
    if (type.startsWith('bytes')) {
        if (value.replace(/^0x/i, '').length % 2 !== 0) {
            throw new web3_errors_1.InvalidBytesError(value);
        }
        const size = parseTypeN(type, 'bytes'.length);
        if (!size || size < 1 || size > 64 || size < value.replace(/^0x/i, '').length / 2) {
            throw new web3_errors_1.InvalidBytesError(value);
        }
        return (0, string_manipulation_js_1.rightPad)(value, size * 2);
    }
    return '';
};
/**
 * returns a string of the tightly packed value given based on the type
 * @param arg - the input to return the tightly packed value
 * @returns - the tightly packed value
 */
const processSolidityEncodePackedArgs = (arg) => {
    const [type, val] = getType(arg);
    // array case
    if (Array.isArray(val)) {
        // go through each element of the array and use map function to create new hexarg list
        const hexArg = val.map((v) => solidityPack(type, v).replace('0x', ''));
        return hexArg.join('');
    }
    const hexArg = solidityPack(type, val);
    return hexArg.replace('0x', '');
};
exports.processSolidityEncodePackedArgs = processSolidityEncodePackedArgs;
/**
 * Encode packed arguments to a hexstring
 */
const encodePacked = (...values) => {
    const hexArgs = values.map(exports.processSolidityEncodePackedArgs);
    return `0x${hexArgs.join('').toLowerCase()}`;
};
exports.encodePacked = encodePacked;
/**
 * Will tightly pack values given in the same way solidity would then hash.
 * returns a hash string, or null if input is empty
 * @param values - the input to return the tightly packed values
 * @returns - the keccack246 of the tightly packed values
 *
 * @example
 * ```ts
 * console.log(web3.utils.soliditySha3({ type: "string", value: "31323334" }));
 * > 0xf15f8da2ad27e486d632dc37d24912f634398918d6f9913a0a0ff84e388be62b
 * ```
 */
const soliditySha3 = (...values) => (0, exports.sha3)((0, exports.encodePacked)(...values));
exports.soliditySha3 = soliditySha3;
/**
 * Will tightly pack values given in the same way solidity would then hash.
 * returns a hash string, if input is empty will return `0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470`
 * @param values - the input to return the tightly packed values
 * @returns - the keccack246 of the tightly packed values
 *
 * @example
 * ```ts
 * console.log(web3.utils.soliditySha3Raw({ type: "string", value: "helloworld" }))
 * > 0xfa26db7ca85ead399216e7c6316bc50ed24393c3122b582735e7f3b0f91b93f0
 * ```
 */
const soliditySha3Raw = (...values) => (0, exports.sha3Raw)((0, exports.encodePacked)(...values));
exports.soliditySha3Raw = soliditySha3Raw;
/**
 * Get slot number for storage long string in contract. Basically for getStorage method
 * returns slotNumber where will data placed
 * @param mainSlotNumber - the slot number where will be stored hash of long string
 * @returns - the slot number where will be stored long string
 */
const getStorageSlotNumForLongString = (mainSlotNumber) => (0, exports.sha3)(`0x${(typeof mainSlotNumber === 'number'
    ? mainSlotNumber.toString()
    : mainSlotNumber).padStart(64, '0')}`);
exports.getStorageSlotNumForLongString = getStorageSlotNumForLongString;
//# sourceMappingURL=hash.js.map