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
import { uint8ArrayConcat } from 'web3-utils';
// eslint-disable-next-line import/no-cycle
import { decodeParamFromAbiParameter, encodeNumber, encodeParamFromAbiParameter } from './index.js';
import { extractArrayType, isDynamic, WORD_SIZE } from '../utils.js';
import { decodeNumber } from './number.js';
import { encodeDynamicParams } from './utils.js';
export function encodeArray(param, values) {
    if (!Array.isArray(values)) {
        throw new AbiError('Expected value to be array', { abi: param, values });
    }
    const { size, param: arrayItemParam } = extractArrayType(param);
    const encodedParams = values.map(v => encodeParamFromAbiParameter(arrayItemParam, v));
    const dynamic = size === -1;
    const dynamicItems = encodedParams.length > 0 && encodedParams[0].dynamic;
    if (!dynamic && values.length !== size) {
        throw new AbiError("Given arguments count doesn't match array length", {
            arrayLength: size,
            argumentsLength: values.length,
        });
    }
    if (dynamic || dynamicItems) {
        const encodingResult = encodeDynamicParams(encodedParams);
        if (dynamic) {
            const encodedLength = encodeNumber({ type: 'uint256', name: '' }, encodedParams.length).encoded;
            return {
                dynamic: true,
                encoded: encodedParams.length > 0
                    ? uint8ArrayConcat(encodedLength, encodingResult)
                    : encodedLength,
            };
        }
        return {
            dynamic: true,
            encoded: encodingResult,
        };
    }
    return {
        dynamic: false,
        encoded: uint8ArrayConcat(...encodedParams.map(p => p.encoded)),
    };
}
export function decodeArray(param, bytes) {
    // eslint-disable-next-line prefer-const
    let { size, param: arrayItemParam } = extractArrayType(param);
    const dynamic = size === -1;
    let consumed = 0;
    const result = [];
    let remaining = bytes;
    // dynamic array, we need to decode length
    if (dynamic) {
        const lengthResult = decodeNumber({ type: 'uint32', name: '' }, bytes);
        size = Number(lengthResult.result);
        consumed = lengthResult.consumed;
        remaining = lengthResult.encoded;
    }
    const hasDynamicChild = isDynamic(arrayItemParam);
    if (hasDynamicChild) {
        // known length but dynamic child, each child is actually head element with encoded offset
        for (let i = 0; i < size; i += 1) {
            const offsetResult = decodeNumber({ type: 'uint32', name: '' }, remaining.subarray(i * WORD_SIZE));
            consumed += offsetResult.consumed;
            const decodedChildResult = decodeParamFromAbiParameter(arrayItemParam, remaining.subarray(Number(offsetResult.result)));
            consumed += decodedChildResult.consumed;
            result.push(decodedChildResult.result);
        }
        return {
            result,
            encoded: remaining.subarray(consumed),
            consumed,
        };
    }
    for (let i = 0; i < size; i += 1) {
        // decode static params
        const decodedChildResult = decodeParamFromAbiParameter(arrayItemParam, bytes.subarray(consumed));
        consumed += decodedChildResult.consumed;
        result.push(decodedChildResult.result);
    }
    return {
        result,
        encoded: bytes.subarray(consumed),
        consumed,
    };
}
//# sourceMappingURL=array.js.map