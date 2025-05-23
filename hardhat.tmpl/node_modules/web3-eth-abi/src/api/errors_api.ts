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

/**
 *
 *  @module ABI
 */

import { sha3Raw } from 'web3-utils';
import { AbiError } from 'web3-errors';
import { AbiErrorFragment } from 'web3-types';
import { jsonInterfaceMethodToString, isAbiErrorFragment } from '../utils.js';

/**
 * Encodes the error name to its ABI signature, which are the sha3 hash of the error name including input types.
 */
export const encodeErrorSignature = (functionName: string | AbiErrorFragment): string => {
	if (typeof functionName !== 'string' && !isAbiErrorFragment(functionName)) {
		throw new AbiError('Invalid parameter value in encodeErrorSignature');
	}

	let name: string;

	if (functionName && (typeof functionName === 'function' || typeof functionName === 'object')) {
		name = jsonInterfaceMethodToString(functionName);
	} else {
		name = functionName;
	}

	return sha3Raw(name);
};
