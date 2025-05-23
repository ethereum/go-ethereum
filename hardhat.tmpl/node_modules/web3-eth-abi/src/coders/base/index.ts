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

import { AbiParameter } from 'web3-types';
import { AbiError } from 'web3-errors';
import { EncoderResult, DecoderResult } from '../types.js';
import { decodeAddress, encodeAddress } from './address.js';
import { decodeBool, encodeBoolean } from './bool.js';
import { decodeBytes, encodeBytes } from './bytes.js';
import { decodeNumber, encodeNumber } from './number.js';
import { decodeString, encodeString } from './string.js';
// eslint-disable-next-line import/no-cycle
import { decodeTuple, encodeTuple } from './tuple.js';
// eslint-disable-next-line import/no-cycle
import { decodeArray, encodeArray } from './array.js';

export { encodeAddress, decodeAddress } from './address.js';
export { encodeBoolean, decodeBool } from './bool.js';
export { encodeBytes, decodeBytes } from './bytes.js';
export { encodeNumber, decodeNumber } from './number.js';
export { encodeString, decodeString } from './string.js';
// eslint-disable-next-line import/no-cycle
export { encodeTuple, decodeTuple } from './tuple.js';
// eslint-disable-next-line import/no-cycle
export { encodeArray, decodeArray } from './array.js';

export function encodeParamFromAbiParameter(param: AbiParameter, value: unknown): EncoderResult {
	if (param.type === 'string') {
		return encodeString(param, value);
	}
	if (param.type === 'bool') {
		return encodeBoolean(param, value);
	}
	if (param.type === 'address') {
		return encodeAddress(param, value);
	}
	if (param.type === 'tuple') {
		return encodeTuple(param, value);
	}
	if (param.type.endsWith(']')) {
		return encodeArray(param, value);
	}
	if (param.type.startsWith('bytes')) {
		return encodeBytes(param, value);
	}
	if (param.type.startsWith('uint') || param.type.startsWith('int')) {
		return encodeNumber(param, value);
	}
	throw new AbiError('Unsupported', {
		param,
		value,
	});
}

export function decodeParamFromAbiParameter(param: AbiParameter, bytes: Uint8Array): DecoderResult {
	if (param.type === 'string') {
		return decodeString(param, bytes);
	}
	if (param.type === 'bool') {
		return decodeBool(param, bytes);
	}
	if (param.type === 'address') {
		return decodeAddress(param, bytes);
	}
	if (param.type === 'tuple') {
		return decodeTuple(param, bytes);
	}
	if (param.type.endsWith(']')) {
		return decodeArray(param, bytes);
	}
	if (param.type.startsWith('bytes')) {
		return decodeBytes(param, bytes);
	}
	if (param.type.startsWith('uint') || param.type.startsWith('int')) {
		return decodeNumber(param, bytes);
	}
	throw new AbiError('Unsupported', {
		param,
		bytes,
	});
}
