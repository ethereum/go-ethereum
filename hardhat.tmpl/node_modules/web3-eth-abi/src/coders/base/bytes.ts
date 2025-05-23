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
import { AbiParameter, Bytes } from 'web3-types';
import { bytesToHex, bytesToUint8Array } from 'web3-utils';
import { isBytes, ValidInputTypes } from 'web3-validator';
import { DecoderResult, EncoderResult } from '../types.js';
import { alloc, WORD_SIZE } from '../utils.js';
import { decodeNumber, encodeNumber } from './number.js';

const MAX_STATIC_BYTES_COUNT = 32;

export function encodeBytes(param: AbiParameter, input: unknown): EncoderResult {
	// hack for odd length hex strings
	if (typeof input === 'string' && input.length % 2 !== 0) {
		// eslint-disable-next-line no-param-reassign
		input += '0';
	}
	if (!isBytes(input as ValidInputTypes)) {
		throw new AbiError('provided input is not valid bytes value', {
			type: param.type,
			value: input,
			name: param.name,
		});
	}
	const bytes = bytesToUint8Array(input as Bytes);
	const [, size] = param.type.split('bytes');
	// fixed size
	if (size) {
		if (Number(size) > MAX_STATIC_BYTES_COUNT || Number(size) < 1) {
			throw new AbiError(
				'invalid bytes type. Static byte type can have between 1 and 32 bytes',
				{
					type: param.type,
				},
			);
		}
		if (Number(size) < bytes.length) {
			throw new AbiError('provided input size is different than type size', {
				type: param.type,
				value: input,
				name: param.name,
			});
		}
		const encoded = alloc(WORD_SIZE);
		encoded.set(bytes);
		return {
			dynamic: false,
			encoded,
		};
	}

	const partsLength = Math.ceil(bytes.length / WORD_SIZE);
	// one word for length of data + WORD for each part of actual data
	const encoded = alloc(WORD_SIZE + partsLength * WORD_SIZE);

	encoded.set(encodeNumber({ type: 'uint32', name: '' }, bytes.length).encoded);
	encoded.set(bytes, WORD_SIZE);
	return {
		dynamic: true,
		encoded,
	};
}

export function decodeBytes(param: AbiParameter, bytes: Uint8Array): DecoderResult<string> {
	const [, sizeString] = param.type.split('bytes');
	let size = Number(sizeString);
	let remainingBytes = bytes;
	let partsCount = 1;
	let consumed = 0;
	if (!size) {
		// dynamic bytes
		const result = decodeNumber({ type: 'uint32', name: '' }, remainingBytes);
		size = Number(result.result);
		consumed += result.consumed;
		remainingBytes = result.encoded;
		partsCount = Math.ceil(size / WORD_SIZE);
	}
	if (size > bytes.length) {
		throw new AbiError('there is not enough data to decode', {
			type: param.type,
			encoded: bytes,
			size,
		});
	}

	return {
		result: bytesToHex(remainingBytes.subarray(0, size)),
		encoded: remainingBytes.subarray(partsCount * WORD_SIZE),
		consumed: consumed + partsCount * WORD_SIZE,
	};
}
