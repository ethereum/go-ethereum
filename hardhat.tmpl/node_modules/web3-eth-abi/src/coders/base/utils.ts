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
import { uint8ArrayConcat } from 'web3-utils';
import { EncoderResult } from '../types.js';
import { WORD_SIZE } from '../utils.js';
import { encodeNumber } from './number.js';

export function encodeDynamicParams(encodedParams: ReadonlyArray<EncoderResult>): Uint8Array {
	let staticSize = 0;
	let dynamicSize = 0;
	const staticParams: EncoderResult[] = [];
	const dynamicParams: EncoderResult[] = [];
	// figure out static size
	for (const encodedParam of encodedParams) {
		if (encodedParam.dynamic) {
			staticSize += WORD_SIZE;
		} else {
			staticSize += encodedParam.encoded.length;
		}
	}

	for (const encodedParam of encodedParams) {
		if (encodedParam.dynamic) {
			staticParams.push(
				encodeNumber({ type: 'uint256', name: '' }, staticSize + dynamicSize),
			);
			dynamicParams.push(encodedParam);
			dynamicSize += encodedParam.encoded.length;
		} else {
			staticParams.push(encodedParam);
		}
	}
	return uint8ArrayConcat(
		...staticParams.map(p => p.encoded),
		...dynamicParams.map(p => p.encoded),
	);
}
