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
import { NibbleWidthError } from 'web3-errors';
import { isHexStrict, validator, utils as validatorUtils, bigintPower } from 'web3-validator';
import { numberToHex, toHex, toNumber } from './converters.js';
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
export const padLeft = (value, characterAmount, sign = '0') => {
    // To avoid duplicate code and circular dependency we will
    // use `padLeft` implementation from `web3-validator`
    if (typeof value === 'string') {
        if (!isHexStrict(value)) {
            return value.padStart(characterAmount, sign);
        }
        return validatorUtils.padLeft(value, characterAmount, sign);
    }
    validator.validate(['int'], [value]);
    return validatorUtils.padLeft(value, characterAmount, sign);
};
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
export const padRight = (value, characterAmount, sign = '0') => {
    if (typeof value === 'string' && !isHexStrict(value)) {
        return value.padEnd(characterAmount, sign);
    }
    const hexString = typeof value === 'string' && isHexStrict(value) ? value : numberToHex(value);
    const prefixLength = hexString.startsWith('-') ? 3 : 2;
    validator.validate([hexString.startsWith('-') ? 'int' : 'uint'], [value]);
    return hexString.padEnd(characterAmount + prefixLength, sign);
};
/**
 * Adds a padding on the right of a string, if value is a integer or bigInt will be converted to a hex string. @alias `padRight`
 */
export const rightPad = padRight;
/**
 * Adds a padding on the left of a string, if value is a integer or bigInt will be converted to a hex string. @alias `padLeft`
 */
export const leftPad = padLeft;
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
export const toTwosComplement = (value, nibbleWidth = 64) => {
    validator.validate(['int'], [value]);
    const val = toNumber(value);
    if (val >= 0)
        return padLeft(toHex(val), nibbleWidth);
    const largestBit = bigintPower(BigInt(2), BigInt(nibbleWidth * 4));
    if (-val >= largestBit) {
        throw new NibbleWidthError(`value: ${value}, nibbleWidth: ${nibbleWidth}`);
    }
    const updatedVal = BigInt(val);
    const complement = updatedVal + largestBit;
    return padLeft(numberToHex(complement), nibbleWidth);
};
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
export const fromTwosComplement = (value, nibbleWidth = 64) => {
    validator.validate(['int'], [value]);
    const val = toNumber(value);
    if (val < 0)
        return val;
    const largestBit = Math.ceil(Math.log(Number(val)) / Math.log(2));
    if (largestBit > nibbleWidth * 4)
        throw new NibbleWidthError(`value: "${value}", nibbleWidth: "${nibbleWidth}"`);
    // check the largest bit to see if negative
    if (nibbleWidth * 4 !== largestBit)
        return val;
    const complement = bigintPower(BigInt(2), BigInt(nibbleWidth) * BigInt(4));
    return toNumber(BigInt(val) - complement);
};
//# sourceMappingURL=string_manipulation.js.map