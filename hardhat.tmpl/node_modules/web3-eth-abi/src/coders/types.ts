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

export type EncoderResult = {
	dynamic: boolean;

	encoded: Uint8Array;
};
export type DecoderResult<T = unknown> = {
	result: T;

	// remaining bytes not yet decoded
	encoded: Uint8Array;

	// number of bytes consumed
	consumed: number;
};

export type NumberType = {
	signed: boolean;
	byteLength: number;
};
export type BytesType = {
	size?: number;
};
