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
exports.fromTwosComplement = exports.toTwosComplement = exports.leftPad = exports.rightPad = exports.padRight = exports.padLeft = void 0;
const web3_errors_1 = require("web3-errors");
const web3_validator_1 = require("web3-validator");
const converters_js_1 = require("./converters.js");
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
const padLeft = (value, characterAmount, sign = '0') => {
    // To avoid duplicate code and circular dependency we will
    // use `padLeft` implementation from `web3-validator`
    if (typeof value === 'string') {
        if (!(0, web3_validator_1.isHexStrict)(value)) {
            return value.padStart(characterAmount, sign);
        }
        return web3_validator_1.utils.padLeft(value, characterAmount, sign);
    }
    web3_validator_1.validator.validate(['int'], [value]);
    return web3_validator_1.utils.padLeft(value, characterAmount, sign);
};
exports.padLeft = padLeft;
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
const padRight = (value, characterAmount, sign = '0') => {
    if (typeof value === 'string' && !(0, web3_validator_1.isHexStrict)(value)) {
        return value.padEnd(characterAmount, sign);
    }
    const hexString = typeof value === 'string' && (0, web3_validator_1.isHexStrict)(value) ? value : (0, converters_js_1.numberToHex)(value);
    const prefixLength = hexString.startsWith('-') ? 3 : 2;
    web3_validator_1.validator.validate([hexString.startsWith('-') ? 'int' : 'uint'], [value]);
    return hexString.padEnd(characterAmount + prefixLength, sign);
};
exports.padRight = padRight;
/**
 * Adds a padding on the right of a string, if value is a integer or bigInt will be converted to a hex string. @alias `padRight`
 */
exports.rightPad = exports.padRight;
/**
 * Adds a padding on the left of a string, if value is a integer or bigInt will be converted to a hex string. @alias `padLeft`
 */
exports.leftPad = exports.padLeft;
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
const toTwosComplement = (value, nibbleWidth = 64) => {
    web3_validator_1.validator.validate(['int'], [value]);
    const val = (0, converters_js_1.toNumber)(value);
    if (val >= 0)
        return (0, exports.padLeft)((0, converters_js_1.toHex)(val), nibbleWidth);
    const largestBit = (0, web3_validator_1.bigintPower)(BigInt(2), BigInt(nibbleWidth * 4));
    if (-val >= largestBit) {
        throw new web3_errors_1.NibbleWidthError(`value: ${value}, nibbleWidth: ${nibbleWidth}`);
    }
    const updatedVal = BigInt(val);
    const complement = updatedVal + largestBit;
    return (0, exports.padLeft)((0, converters_js_1.numberToHex)(complement), nibbleWidth);
};
exports.toTwosComplement = toTwosComplement;
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
const fromTwosComplement = (value, nibbleWidth = 64) => {
    web3_validator_1.validator.validate(['int'], [value]);
    const val = (0, converters_js_1.toNumber)(value);
    if (val < 0)
        return val;
    const largestBit = Math.ceil(Math.log(Number(val)) / Math.log(2));
    if (largestBit > nibbleWidth * 4)
        throw new web3_errors_1.NibbleWidthError(`value: "${value}", nibbleWidth: "${nibbleWidth}"`);
    // check the largest bit to see if negative
    if (nibbleWidth * 4 !== largestBit)
        return val;
    const complement = (0, web3_validator_1.bigintPower)(BigInt(2), BigInt(nibbleWidth) * BigInt(4));
    return (0, converters_js_1.toNumber)(BigInt(val) - complement);
};
exports.fromTwosComplement = fromTwosComplement;
//# sourceMappingURL=string_manipulation.js.map