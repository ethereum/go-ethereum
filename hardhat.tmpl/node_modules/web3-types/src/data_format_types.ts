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

import { Bytes, HexString, Numbers } from './primitives_types.js';

export enum FMT_NUMBER {
	NUMBER = 'NUMBER_NUMBER',
	HEX = 'NUMBER_HEX',
	STR = 'NUMBER_STR',
	BIGINT = 'NUMBER_BIGINT',
}

export type NumberTypes = {
	[FMT_NUMBER.NUMBER]: number;
	[FMT_NUMBER.HEX]: HexString;
	[FMT_NUMBER.STR]: string;
	[FMT_NUMBER.BIGINT]: bigint;
};

export enum FMT_BYTES {
	HEX = 'BYTES_HEX',
	UINT8ARRAY = 'BYTES_UINT8ARRAY',
}

export type ByteTypes = {
	[FMT_BYTES.HEX]: HexString;
	[FMT_BYTES.UINT8ARRAY]: Uint8Array;
};

/**
 * Used to specify how data should be formatted. Bytes can be formatted as hexadecimal strings or
 * Uint8Arrays. Numbers can be formatted as BigInts, hexadecimal strings, primitive numbers, or
 * strings.
 */
export type DataFormat = {
	readonly number: FMT_NUMBER;
	readonly bytes: FMT_BYTES;
};

export const DEFAULT_RETURN_FORMAT = {
	number: FMT_NUMBER.BIGINT,
	bytes: FMT_BYTES.HEX,
} as const;
export const ETH_DATA_FORMAT = { number: FMT_NUMBER.HEX, bytes: FMT_BYTES.HEX } as const;

export type FormatType<T, F extends DataFormat> = number extends Extract<T, Numbers>
	? NumberTypes[F['number']] | Exclude<T, Numbers>
	: Uint8Array extends Extract<T, Bytes>
	? ByteTypes[F['bytes']] | Exclude<T, Bytes>
	: T extends object | undefined
	? {
			[P in keyof T]: FormatType<T[P], F>;
	  }
	: T;
