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

import { ValidInputTypes } from '../types.js';

/**
 * checks input if typeof data is valid string input
 */
export const isString = (value: ValidInputTypes) => typeof value === 'string';

export const isHexStrict = (hex: ValidInputTypes) =>
	typeof hex === 'string' && /^((-)?0x[0-9a-f]+|(0x))$/i.test(hex);

/**
 * Is the string a hex string.
 *
 * @param  value
 * @param  length
 * @returns  output the string is a hex string
 */
export function isHexString(value: string, length?: number): boolean {
	if (typeof value !== 'string' || !value.match(/^0x[0-9A-Fa-f]*$/)) return false;

	if (typeof length !== 'undefined' && length > 0 && value.length !== 2 + 2 * length)
		return false;

	return true;
}

export const isHex = (hex: ValidInputTypes): boolean =>
	typeof hex === 'number' ||
	typeof hex === 'bigint' ||
	(typeof hex === 'string' && /^((-0x|0x|-)?[0-9a-f]+|(0x))$/i.test(hex));

export const isHexString8Bytes = (value: string, prefixed = true) =>
	prefixed ? isHexStrict(value) && value.length === 18 : isHex(value) && value.length === 16;

export const isHexString32Bytes = (value: string, prefixed = true) =>
	prefixed ? isHexStrict(value) && value.length === 66 : isHex(value) && value.length === 64;

/**
 * Returns a `Boolean` on whether or not the a `String` starts with '0x'
 * @param str the string input value
 * @return a boolean if it is or is not hex prefixed
 * @throws if the str input is not a string
 */
export function isHexPrefixed(str: string): boolean {
	if (typeof str !== 'string') {
		throw new Error(`[isHexPrefixed] input must be type 'string', received type ${typeof str}`);
	}

	return str.startsWith('0x');
}

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
export const validateNoLeadingZeroes = function (values: {
	[key: string]: Uint8Array | undefined;
}) {
	for (const [k, v] of Object.entries(values)) {
		if (v !== undefined && v.length > 0 && v[0] === 0) {
			throw new Error(`${k} cannot have leading zeroes, received: ${v.toString()}`);
		}
	}
};
