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

import { Numbers } from './primitives_types.js';

// Make each attribute mutable by removing `readonly`
export type Mutable<T> = {
	-readonly [P in keyof T]: T[P];
};

export type ConnectionEvent = {
	code: number;
	reason: string;
	wasClean?: boolean; // if WS connection was closed properly
};

export type Optional<T, K extends keyof T> = Pick<Partial<T>, K> & Omit<T, K>;
export type EncodingTypes = Numbers | boolean | Numbers[] | boolean[];

export type TypedObject = {
	type: string;
	value: EncodingTypes;
};

export type TypedObjectAbbreviated = {
	t: string;
	v: EncodingTypes;
};

export type Sha3Input = TypedObject | TypedObjectAbbreviated | Numbers | boolean | object;

export type IndexKeysForArray<A extends readonly unknown[]> = Exclude<keyof A, keyof []>;

export type ArrayToIndexObject<T extends ReadonlyArray<unknown>> = {
	[K in IndexKeysForArray<T>]: T[K];
};

type _Grow<T, A extends Array<T>> = ((x: T, ...xs: A) => void) extends (...a: infer X) => void
	? X
	: never;

export type GrowToSize<T, A extends Array<T>, N extends number> = {
	0: A;
	1: GrowToSize<T, _Grow<T, A>, N>;
}[A['length'] extends N ? 0 : 1];

export type FixedSizeArray<T, N extends number> = GrowToSize<T, [], N>;
