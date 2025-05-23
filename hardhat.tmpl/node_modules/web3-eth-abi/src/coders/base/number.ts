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
import type { AbiParameter } from 'web3-types';
import { padLeft, toBigInt } from 'web3-utils';
import { utils } from 'web3-validator';
import { DecoderResult, EncoderResult } from '../types.js';
import { WORD_SIZE } from '../utils.js';
import { numberLimits } from './numbersLimits.js';

// eslint-disable-next-line no-bitwise
const mask = BigInt(1) << BigInt(256);

function bigIntToUint8Array(value: bigint, byteLength = WORD_SIZE): Uint8Array {
	let hexValue;
	if (value < 0) {
		hexValue = (mask + value).toString(16);
	} else {
		hexValue = value.toString(16);
	}
	hexValue = padLeft(hexValue, byteLength * 2);
	return utils.hexToUint8Array(hexValue);
}

function uint8ArrayToBigInt(value: Uint8Array, max: bigint): bigint {
	const hexValue = utils.uint8ArrayToHexString(value);
	const result = BigInt(hexValue);
	if (result <= max) return result;
	return result - mask;
}

export function encodeNumber(param: AbiParameter, input: unknown): EncoderResult {
	let value;
	try {
		value = toBigInt(input);
	} catch (e) {
		throw new AbiError('provided input is not number value', {
			type: param.type,
			value: input,
			name: param.name,
		});
	}
	const limit = numberLimits.get(param.type);
	if (!limit) {
		throw new AbiError('provided abi contains invalid number datatype', { type: param.type });
	}
	if (value < limit.min) {
		throw new AbiError('provided input is less then minimum for given type', {
			type: param.type,
			value: input,
			name: param.name,
			minimum: limit.min.toString(),
		});
	}
	if (value > limit.max) {
		throw new AbiError('provided input is greater then maximum for given type', {
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

export function decodeNumber(param: AbiParameter, bytes: Uint8Array): DecoderResult<bigint> {
	if (bytes.length < WORD_SIZE) {
		throw new AbiError('Not enough bytes left to decode', { param, bytesLeft: bytes.length });
	}
	const boolBytes = bytes.subarray(0, WORD_SIZE);
	const limit = numberLimits.get(param.type);
	if (!limit) {
		throw new AbiError('provided abi contains invalid number datatype', { type: param.type });
	}
	const numberResult = uint8ArrayToBigInt(boolBytes, limit.max);

	if (numberResult < limit.min) {
		throw new AbiError('decoded value is less then minimum for given type', {
			type: param.type,
			value: numberResult,
			name: param.name,
			minimum: limit.min.toString(),
		});
	}
	if (numberResult > limit.max) {
		throw new AbiError('decoded value is greater then maximum for given type', {
			type: param.type,
			value: numberResult,
			name: param.name,
			maximum: limit.max.toString(),
		});
	}
	return {
		result: numberResult,
		encoded: bytes.subarray(WORD_SIZE),
		consumed: WORD_SIZE,
	};
}
