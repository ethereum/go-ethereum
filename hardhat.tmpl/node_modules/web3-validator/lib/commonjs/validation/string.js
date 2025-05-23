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
exports.validateNoLeadingZeroes = exports.isHexPrefixed = exports.isHexString32Bytes = exports.isHexString8Bytes = exports.isHex = exports.isHexString = exports.isHexStrict = exports.isString = void 0;
/**
 * checks input if typeof data is valid string input
 */
const isString = (value) => typeof value === 'string';
exports.isString = isString;
const isHexStrict = (hex) => typeof hex === 'string' && /^((-)?0x[0-9a-f]+|(0x))$/i.test(hex);
exports.isHexStrict = isHexStrict;
/**
 * Is the string a hex string.
 *
 * @param  value
 * @param  length
 * @returns  output the string is a hex string
 */
function isHexString(value, length) {
    if (typeof value !== 'string' || !value.match(/^0x[0-9A-Fa-f]*$/))
        return false;
    if (typeof length !== 'undefined' && length > 0 && value.length !== 2 + 2 * length)
        return false;
    return true;
}
exports.isHexString = isHexString;
const isHex = (hex) => typeof hex === 'number' ||
    typeof hex === 'bigint' ||
    (typeof hex === 'string' && /^((-0x|0x|-)?[0-9a-f]+|(0x))$/i.test(hex));
exports.isHex = isHex;
const isHexString8Bytes = (value, prefixed = true) => prefixed ? (0, exports.isHexStrict)(value) && value.length === 18 : (0, exports.isHex)(value) && value.length === 16;
exports.isHexString8Bytes = isHexString8Bytes;
const isHexString32Bytes = (value, prefixed = true) => prefixed ? (0, exports.isHexStrict)(value) && value.length === 66 : (0, exports.isHex)(value) && value.length === 64;
exports.isHexString32Bytes = isHexString32Bytes;
/**
 * Returns a `Boolean` on whether or not the a `String` starts with '0x'
 * @param str the string input value
 * @return a boolean if it is or is not hex prefixed
 * @throws if the str input is not a string
 */
function isHexPrefixed(str) {
    if (typeof str !== 'string') {
        throw new Error(`[isHexPrefixed] input must be type 'string', received type ${typeof str}`);
    }
    return str.startsWith('0x');
}
exports.isHexPrefixed = isHexPrefixed;
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
const validateNoLeadingZeroes = function (values) {
    for (const [k, v] of Object.entries(values)) {
        if (v !== undefined && v.length > 0 && v[0] === 0) {
            throw new Error(`${k} cannot have leading zeroes, received: ${v.toString()}`);
        }
    }
};
exports.validateNoLeadingZeroes = validateNoLeadingZeroes;
//# sourceMappingURL=string.js.map