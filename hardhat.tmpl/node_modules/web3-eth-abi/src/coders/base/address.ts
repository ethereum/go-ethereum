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
import { AbiError } from 'web3-errors';
import { AbiParameter } from 'web3-types';
import { toChecksumAddress } from 'web3-utils';
import { isAddress, utils } from 'web3-validator';
import { DecoderResult, EncoderResult } from '../types.js';
import { alloc, WORD_SIZE } from '../utils.js';

const ADDRESS_BYTES_COUNT = 20;
const ADDRESS_OFFSET = WORD_SIZE - ADDRESS_BYTES_COUNT;

export function encodeAddress(param: AbiParameter, input: unknown): EncoderResult {
	if (typeof input !== 'string') {
		throw new AbiError('address type expects string as input type', {
			value: input,
			name: param.name,
			type: param.type,
		});
	}
	let address = input.toLowerCase();
	if (!address.startsWith('0x')) {
		address = `0x${address}`;
	}
	if (!isAddress(address)) {
		throw new AbiError('provided input is not valid address', {
			value: input,
			name: param.name,
			type: param.type,
		});
	}
	// for better performance, we could convert hex to destination bytes directly (encoded var)
	const addressBytes = utils.hexToUint8Array(address);
	// expand address to WORD_SIZE
	const encoded = alloc(WORD_SIZE);
	encoded.set(addressBytes, ADDRESS_OFFSET);
	return {
		dynamic: false,
		encoded,
	};
}

export function decodeAddress(_param: AbiParameter, bytes: Uint8Array): DecoderResult<string> {
	const addressBytes = bytes.subarray(ADDRESS_OFFSET, WORD_SIZE);
	if (addressBytes.length !== ADDRESS_BYTES_COUNT) {
		throw new AbiError('Invalid decoding input, not enough bytes to decode address', { bytes });
	}
	const result = utils.uint8ArrayToHexString(addressBytes);

	// should we check is decoded value is valid address?
	// if(!isAddress(result)) {
	//     throw new AbiError("encoded data is not valid address", {
	//         address: result,
	//     });
	// }
	return {
		result: toChecksumAddress(result),
		encoded: bytes.subarray(WORD_SIZE),
		consumed: WORD_SIZE,
	};
}
