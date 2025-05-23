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

import { AbiInput, HexString } from 'web3-types';
import { utils } from 'web3-validator';
import { decodeTuple } from './base/tuple.js';
import { toAbiParams } from './utils.js';

export function decodeParameters(
	abis: AbiInput[] | ReadonlyArray<AbiInput>,
	bytes: HexString,
	_loose: boolean,
): { [key: string]: unknown; __length__: number } {
	const abiParams = toAbiParams(abis);
	const bytesArray = utils.hexToUint8Array(bytes);

	return decodeTuple({ type: 'tuple', name: '', components: abiParams }, bytesArray).result;
}
