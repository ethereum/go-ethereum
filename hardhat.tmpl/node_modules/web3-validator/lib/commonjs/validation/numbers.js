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
exports.isNumber = exports.isInt = exports.isUInt = exports.bigintPower = exports.isBigInt = void 0;
const utils_js_1 = require("../utils.js");
const string_js_1 = require("./string.js");
/**
 * Checks if a given value is a valid big int
 */
const isBigInt = (value) => typeof value === 'bigint';
exports.isBigInt = isBigInt;
// Note: this could be simplified using ** operator, but babel does not handle it well
// 	you can find more at: https://github.com/babel/babel/issues/13109 and https://github.com/web3/web3.js/issues/6187
/** @internal */
const bigintPower = (base, expo) => {
    // edge case
    if (expo === BigInt(0)) {
        return BigInt(1);
    }
    let res = base;
    for (let index = 1; index < expo; index += 1) {
        res *= base;
    }
    return res;
};
exports.bigintPower = bigintPower;
const isUInt = (value, options = {
    abiType: 'uint',
}) => {
    if (!['number', 'string', 'bigint'].includes(typeof value) ||
        (typeof value === 'string' && value.length === 0)) {
        return false;
    }
    let size;
    if (options === null || options === void 0 ? void 0 : options.abiType) {
        const { baseTypeSize } = (0, utils_js_1.parseBaseType)(options.abiType);
        if (baseTypeSize) {
            size = baseTypeSize;
        }
    }
    else if (options.bitSize) {
        size = options.bitSize;
    }
    const maxSize = (0, exports.bigintPower)(BigInt(2), BigInt(size !== null && size !== void 0 ? size : 256)) - BigInt(1);
    try {
        const valueToCheck = typeof value === 'string' && (0, string_js_1.isHexStrict)(value)
            ? BigInt((0, utils_js_1.hexToNumber)(value))
            : BigInt(value);
        return valueToCheck >= 0 && valueToCheck <= maxSize;
    }
    catch (error) {
        // Some invalid number value given which can not be converted via BigInt
        return false;
    }
};
exports.isUInt = isUInt;
const isInt = (value, options = {
    abiType: 'int',
}) => {
    if (!['number', 'string', 'bigint'].includes(typeof value)) {
        return false;
    }
    if (typeof value === 'number' && value > Number.MAX_SAFE_INTEGER) {
        return false;
    }
    let size;
    if (options === null || options === void 0 ? void 0 : options.abiType) {
        const { baseTypeSize, baseType } = (0, utils_js_1.parseBaseType)(options.abiType);
        if (baseType !== 'int') {
            return false;
        }
        if (baseTypeSize) {
            size = baseTypeSize;
        }
    }
    else if (options.bitSize) {
        size = options.bitSize;
    }
    const maxSize = (0, exports.bigintPower)(BigInt(2), BigInt((size !== null && size !== void 0 ? size : 256) - 1));
    const minSize = BigInt(-1) * (0, exports.bigintPower)(BigInt(2), BigInt((size !== null && size !== void 0 ? size : 256) - 1));
    try {
        const valueToCheck = typeof value === 'string' && (0, string_js_1.isHexStrict)(value)
            ? BigInt((0, utils_js_1.hexToNumber)(value))
            : BigInt(value);
        return valueToCheck >= minSize && valueToCheck <= maxSize;
    }
    catch (error) {
        // Some invalid number value given which can not be converted via BigInt
        return false;
    }
};
exports.isInt = isInt;
const isNumber = (value) => {
    if ((0, exports.isInt)(value)) {
        return true;
    }
    // It would be a decimal number
    if (typeof value === 'string' &&
        /[0-9.]/.test(value) &&
        value.indexOf('.') === value.lastIndexOf('.')) {
        return true;
    }
    if (typeof value === 'number') {
        return true;
    }
    return false;
};
exports.isNumber = isNumber;
//# sourceMappingURL=numbers.js.map