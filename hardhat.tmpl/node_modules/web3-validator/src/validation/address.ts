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

import { keccak256 } from 'ethereum-cryptography/keccak.js';
import { utf8ToBytes } from 'ethereum-cryptography/utils.js';
import { ValidInputTypes } from '../types.js';
import { ensureIfUint8Array, uint8ArrayToHexString } from '../utils.js';
import { isHexStrict } from './string.js';
import { isUint8Array } from './bytes.js';

/**
 * Checks the checksum of a given address. Will also return false on non-checksum addresses.
 */
export const checkAddressCheckSum = (data: string): boolean => {
	if (!/^(0x)?[0-9a-f]{40}$/i.test(data)) return false;
	const address = data.slice(2);
	const updatedData = utf8ToBytes(address.toLowerCase());

	const addressHash = uint8ArrayToHexString(keccak256(ensureIfUint8Array(updatedData))).slice(2);

	for (let i = 0; i < 40; i += 1) {
		// the nth letter should be uppercase if the nth digit of casemap is 1
		if (
			(parseInt(addressHash[i], 16) > 7 && address[i].toUpperCase() !== address[i]) ||
			(parseInt(addressHash[i], 16) <= 7 && address[i].toLowerCase() !== address[i])
		) {
			return false;
		}
	}
	return true;
};

/**
 * Checks if a given string is a valid Ethereum address. It will also check the checksum, if the address has upper and lowercase letters.
 */
export const isAddress = (value: ValidInputTypes, checkChecksum = true) => {
	if (typeof value !== 'string' && !isUint8Array(value)) {
		return false;
	}

	let valueToCheck: string;

	if (isUint8Array(value)) {
		valueToCheck = uint8ArrayToHexString(value);
	} else if (typeof value === 'string' && !isHexStrict(value)) {
		valueToCheck = value.toLowerCase().startsWith('0x') ? value : `0x${value}`;
	} else {
		valueToCheck = value;
	}

	// check if it has the basic requirements of an address
	if (!/^(0x)?[0-9a-f]{40}$/i.test(valueToCheck)) {
		return false;
	}
	// If it's ALL lowercase or ALL upppercase
	if (
		/^(0x|0X)?[0-9a-f]{40}$/.test(valueToCheck) ||
		/^(0x|0X)?[0-9A-F]{40}$/.test(valueToCheck)
	) {
		return true;
		// Otherwise check each case
	}
	return checkChecksum ? checkAddressCheckSum(valueToCheck) : true;
};
