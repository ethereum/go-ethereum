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
import { isNullish, keccak256 } from 'web3-utils';
import { AbiError } from 'web3-errors';
import { encodeParameters } from './coders/encode.js';
const TYPE_REGEX = /^\w+/;
const ARRAY_REGEX = /^(.*)\[([0-9]*?)]$/;
/**
 * Get the dependencies of a struct type. If a struct has the same dependency multiple times, it's only included once
 * in the resulting array.
 */
const getDependencies = (typedData, type, dependencies = []) => {
    const match = type.match(TYPE_REGEX);
    const actualType = match[0];
    if (dependencies.includes(actualType)) {
        return dependencies;
    }
    if (!typedData.types[actualType]) {
        return dependencies;
    }
    return [
        actualType,
        ...typedData.types[actualType].reduce((previous, _type) => [
            ...previous,
            ...getDependencies(typedData, _type.type, previous).filter(dependency => !previous.includes(dependency)),
        ], []),
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
const encodeType = (typedData, type) => {
    const [primary, ...dependencies] = getDependencies(typedData, type);
    // eslint-disable-next-line @typescript-eslint/require-array-sort-compare
    const types = [primary, ...dependencies.sort()];
    return types
        .map(dependency => 
    // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
    `${dependency}(${typedData.types[dependency].map(_type => `${_type.type} ${_type.name}`)})`)
        .join('');
};
/**
 * Get a type string as hash.
 */
const getTypeHash = (typedData, type) => keccak256(encodeType(typedData, type));
/**
 * Get encoded data as a hash. The data should be a key -> value object with all the required values. All dependant
 * types are automatically encoded.
 */
const getStructHash = (typedData, type, data) => keccak256(encodeData(typedData, type, data));
/**
 * Get the EIP-191 encoded message to sign, from the typedData object. If `hash` is enabled, the message will be hashed
 * with Keccak256.
 */
export const getMessage = (typedData, hash) => {
    const EIP_191_PREFIX = '1901';
    const message = `0x${EIP_191_PREFIX}${getStructHash(typedData, 'EIP712Domain', typedData.domain).substring(2)}${getStructHash(typedData, typedData.primaryType, typedData.message).substring(2)}`;
    if (hash) {
        return keccak256(message);
    }
    return message;
};
/**
 * Encodes a single value to an ABI serialisable string, number or Buffer. Returns the data as tuple, which consists of
 * an array of ABI compatible types, and an array of corresponding values.
 */
const encodeValue = (typedData, type, data) => {
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
            throw new AbiError(`Cannot encode data: expected length of ${length}, but got ${data.length}`, {
                data,
            });
        }
        const encodedData = data.map(item => encodeValue(typedData, arrayType, item));
        const types = encodedData.map(item => item[0]);
        const values = encodedData.map(item => item[1]);
        return ['bytes32', keccak256(encodeParameters(types, values))];
    }
    if (typedData.types[type]) {
        return ['bytes32', getStructHash(typedData, type, data)];
    }
    // Strings and arbitrary byte arrays are hashed to bytes32
    if (type === 'string') {
        return ['bytes32', keccak256(data)];
    }
    if (type === 'bytes') {
        return ['bytes32', keccak256(data)];
    }
    return [type, data];
};
/**
 * Encode the data to an ABI encoded Buffer. The data should be a key -> value object with all the required values. All
 * dependant types are automatically encoded.
 */
const encodeData = (typedData, type, data) => {
    const [types, values] = typedData.types[type].reduce(([_types, _values], field) => {
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
    }, [['bytes32'], [getTypeHash(typedData, type)]]);
    return encodeParameters(types, values);
};
//# sourceMappingURL=eip_712.js.map