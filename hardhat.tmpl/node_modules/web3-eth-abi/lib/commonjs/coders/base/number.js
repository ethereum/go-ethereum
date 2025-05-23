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
exports.encodeNumber = encodeNumber;
exports.decodeNumber = decodeNumber;
const web3_errors_1 = require("web3-errors");
const web3_utils_1 = require("web3-utils");
const web3_validator_1 = require("web3-validator");
const utils_js_1 = require("../utils.js");
const numbersLimits_js_1 = require("./numbersLimits.js");
// eslint-disable-next-line no-bitwise
const mask = BigInt(1) << BigInt(256);
function bigIntToUint8Array(value, byteLength = utils_js_1.WORD_SIZE) {
    let hexValue;
    if (value < 0) {
        hexValue = (mask + value).toString(16);
    }
    else {
        hexValue = value.toString(16);
    }
    hexValue = (0, web3_utils_1.padLeft)(hexValue, byteLength * 2);
    return web3_validator_1.utils.hexToUint8Array(hexValue);
}
function uint8ArrayToBigInt(value, max) {
    const hexValue = web3_validator_1.utils.uint8ArrayToHexString(value);
    const result = BigInt(hexValue);
    if (result <= max)
        return result;
    return result - mask;
}
function encodeNumber(param, input) {
    let value;
    try {
        value = (0, web3_utils_1.toBigInt)(input);
    }
    catch (e) {
        throw new web3_errors_1.AbiError('provided input is not number value', {
            type: param.type,
            value: input,
            name: param.name,
        });
    }
    const limit = numbersLimits_js_1.numberLimits.get(param.type);
    if (!limit) {
        throw new web3_errors_1.AbiError('provided abi contains invalid number datatype', { type: param.type });
    }
    if (value < limit.min) {
        throw new web3_errors_1.AbiError('provided input is less then minimum for given type', {
            type: param.type,
            value: input,
            name: param.name,
            minimum: limit.min.toString(),
        });
    }
    if (value > limit.max) {
        throw new web3_errors_1.AbiError('provided input is greater then maximum for given type', {
            type: param.type,
            value: input,
            name: param.name,
            maximum: limit.max.toString(),
        });
    }
    return {
        dynamic: false,
        encoded: bigIntToUint8Array(value),
    };
}
function decodeNumber(param, bytes) {
    if (bytes.length < utils_js_1.WORD_SIZE) {
        throw new web3_errors_1.AbiError('Not enough bytes left to decode', { param, bytesLeft: bytes.length });
    }
    const boolBytes = bytes.subarray(0, utils_js_1.WORD_SIZE);
    const limit = numbersLimits_js_1.numberLimits.get(param.type);
    if (!limit) {
        throw new web3_errors_1.AbiError('provided abi contains invalid number datatype', { type: param.type });
    }
    const numberResult = uint8ArrayToBigInt(boolBytes, limit.max);
    if (numberResult < limit.min) {
        throw new web3_errors_1.AbiError('decoded value is less then minimum for given type', {
            type: param.type,
            value: numberResult,
            name: param.name,
            minimum: limit.min.toString(),
        });
    }
    if (numberResult > limit.max) {
        throw new web3_errors_1.AbiError('decoded value is greater then maximum for given type', {
            type: param.type,
            value: numberResult,
            name: param.name,
            maximum: limit.max.toString(),
        });
    }
    return {
        result: numberResult,
        encoded: bytes.subarray(utils_js_1.WORD_SIZE),
        consumed: utils_js_1.WORD_SIZE,
    };
}
//# sourceMappingURL=number.js.map