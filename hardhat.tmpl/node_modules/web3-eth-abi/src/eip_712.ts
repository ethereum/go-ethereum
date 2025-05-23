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
 * The web3.eth.abi functions let you encode and decode parameters to ABI (Application Binary Interface) for function calls to the EVM (Ethereum Virtual Machine).
 *
 *  For using Web3 ABI functions, first install Web3 package using `npm i web3` or `yarn add web3`.
 * After that, Web3 ABI functions will be available.
 * ```ts
 * import { Web3 } from 'web3';
 *
 * const web3 = new Web3();
 * const encoded = web3.eth.abi.encodeFunctionSignature({
 *     name: 'myMethod',
 *     type: 'function',
 *     inputs: [{
 *         type: 'uint256',
 *         name: 'myNumber'
 *     },{
 *         type: 'string',
 *         name: 'myString'
 *     }]
 * });
 *
 * ```
 *
 * For using individual package install `web3-eth-abi` package using `npm i web3-eth-abi` or `yarn add web3-eth-abi` and only import required functions.
 * This is more efficient approach for building lightweight applications.
 * ```ts
 * import { encodeFunctionSignature } from 'web3-eth-abi';
 *
 * const encoded = encodeFunctionSignature({
 *     name: 'myMethod',
 *     type: 'function',
 *     inputs: [{
 *         type: 'uint256',
 *         name: 'myNumber'
 *     },{
 *         type: 'string',
 *         name: 'myString'
 *     }]
 * });
 *
 * ```
 *
 *  @module ABI
 */

// This code was taken from: https://github.com/Mrtenz/eip-712/tree/master

import { Eip712TypedData } from 'web3-types';
import { isNullish, keccak256 } from 'web3-utils';
import { AbiError } from 'web3-errors';
import { encodeParameters } from './coders/encode.js';

const TYPE_REGEX = /^\w+/;
const ARRAY_REGEX = /^(.*)\[([0-9]*?)]$/;

/**
 * Get the dependencies of a struct type. If a struct has the same dependency multiple times, it's only included once
 * in the resulting array.
 */
const getDependencies = (
	typedData: Eip712TypedData,
	type: string,
	dependencies: string[] = [],
): string[] => {
	const match = type.match(TYPE_REGEX)!;
	const actualType = match[0];
	if (dependencies.includes(actualType)) {
		return dependencies;
	}

	if (!typedData.types[actualType]) {
		return dependencies;
	}

	return [
		actualType,
		...typedData.types[actualType].reduce<string[]>(
			(previous, _type) => [
				...previous,
				...getDependencies(typedData, _type.type, previous).filter(
					dependency => !previous.includes(dependency),
				),
			],
			[],
		),
	];
};

/**
 * Encode a type to a string. All dependant types are alphabetically sorted.
 *
 * @param {TypedData} typedData
 * @param {string} type
 * @param {Options} [options]
 * @return {string}
 */
const encodeType = (typedData: Eip712TypedData, type: string): string => {
	const [primary, ...dependencies] = getDependencies(typedData, type);
	// eslint-disable-next-line @typescript-eslint/require-array-sort-compare
	const types = [primary, ...dependencies.sort()];

	return types
		.map(
			dependency =>
				// eslint-disable-next-line @typescript-eslint/restrict-template-expressions
				`${dependency}(${typedData.types[dependency].map(
					_type => `${_type.type} ${_type.name}`,
				)})`,
		)
		.join('');
};

/**
 * Get a type string as hash.
 */
const getTypeHash = (typedData: Eip712TypedData, type: string) =>
	keccak256(encodeType(typedData, type));

/**
 * Get encoded data as a hash. The data should be a key -> value object with all the required values. All dependant
 * types are automatically encoded.
 */
const getStructHash = (
	typedData: Eip712TypedData,
	type: string,
	data: Record<string, unknown>,
	// eslint-disable-next-line  no-use-before-define
): string => keccak256(encodeData(typedData, type, data));

/**
 * Get the EIP-191 encoded message to sign, from the typedData object. If `hash` is enabled, the message will be hashed
 * with Keccak256.
 */
export const getMessage = (typedData: Eip712TypedData, hash?: boolean): string => {
	const EIP_191_PREFIX = '1901';
	const message = `0x${EIP_191_PREFIX}${getStructHash(
		typedData,
		'EIP712Domain',
		typedData.domain as Record<string, unknown>,
	).substring(2)}${getStructHash(typedData, typedData.primaryType, typedData.message).substring(
		2,
	)}`;

	if (hash) {
		return keccak256(message);
	}

	return message;
};

/**
 * Encodes a single value to an ABI serialisable string, number or Buffer. Returns the data as tuple, which consists of
 * an array of ABI compatible types, and an array of corresponding values.
 */
const encodeValue = (
	typedData: Eip712TypedData,
	type: string,
	data: unknown,
): [string, string | Uint8Array | number] => {
	const match = type.match(ARRAY_REGEX);

	// Checks for array types
	if (match) {
		const arrayType = match[1];
		const length = Number(match[2]) || undefined;

		if (!Array.isArray(data)) {
			throw new AbiError('Cannot encode data: value is not of array type', {
				data,
			});
		}

		if (length && data.length !== length) {
			throw new AbiError(
				`Cannot encode data: expected length of ${length}, but got ${data.length}`,
				{
					data,
				},
			);
		}

		const encodedData = data.map(item => encodeValue(typedData, arrayType, item));
		const types = encodedData.map(item => item[0]);
		const values = encodedData.map(item => item[1]);

		return ['bytes32', keccak256(encodeParameters(types, values))];
	}

	if (typedData.types[type]) {
		return ['bytes32', getStructHash(typedData, type, data as Record<string, unknown>)];
	}

	// Strings and arbitrary byte arrays are hashed to bytes32
	if (type === 'string') {
		return ['bytes32', keccak256(data as string)];
	}

	if (type === 'bytes') {
		return ['bytes32', keccak256(data as string)];
	}

	return [type, data as string];
};

/**
 * Encode the data to an ABI encoded Buffer. The data should be a key -> value object with all the required values. All
 * dependant types are automatically encoded.
 */
const encodeData = (
	typedData: Eip712TypedData,
	type: string,
	data: Record<string, unknown>,
): string => {
	const [types, values] = typedData.types[type].reduce<[string[], unknown[]]>(
		([_types, _values], field) => {
			if (isNullish(data[field.name]) || isNullish(field.type)) {
				throw new AbiError(`Cannot encode data: missing data for '${field.name}'`, {
					data,
					field,
				});
			}

			const value = data[field.name];
			const [_type, encodedValue] = encodeValue(typedData, field.type, value);

			return [
				[..._types, _type],
				[..._values, encodedValue],
			];
		},
		[['bytes32'], [getTypeHash(typedData, type)]],
	);

	return encodeParameters(types, values);
};
