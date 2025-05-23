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

export interface Web3Error extends Error {
	readonly name: string;
	readonly code: number;
	readonly stack?: string;
}

export type Web3ValidationErrorObject<
	K extends string = string,
	P = Record<string, any>,
	S = unknown,
> = {
	keyword: K;
	instancePath: string;
	schemaPath: string;
	params: P;
	// Added to validation errors of "propertyNames" keyword schema
	propertyName?: string;
	// Excluded if option `messages` set to false.
	message?: string;
	schema?: S;
	data?: unknown;
};
