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
import { hexToUtf8, utf8ToBytes } from 'web3-utils';
import { DecoderResult, EncoderResult } from '../types.js';
import { decodeBytes, encodeBytes } from './bytes.js';

export function encodeString(_param: AbiParameter, input: unknown): EncoderResult {
	if (typeof input !== 'string') {
		throw new AbiError('invalid input, should be string', { input });
	}
	const bytes = utf8ToBytes(input);
	return encodeBytes({ type: 'bytes', name: '' }, bytes);
}

export function decodeString(_param: AbiParameter, bytes: Uint8Array): DecoderResult<string> {
	const r = decodeBytes({ type: 'bytes', name: '' }, bytes);
	return {
		result: hexToUtf8(r.result),
		encoded: r.encoded,
		consumed: r.consumed,
	};
}
