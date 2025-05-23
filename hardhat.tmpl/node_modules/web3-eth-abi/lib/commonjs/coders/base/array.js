"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.encodeArray = encodeArray;
exports.decodeArray = decodeArray;
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
const web3_errors_1 = require("web3-errors");
const web3_utils_1 = require("web3-utils");
// eslint-disable-next-line import/no-cycle
const index_js_1 = require("./index.js");
const utils_js_1 = require("../utils.js");
const number_js_1 = require("./number.js");
const utils_js_2 = require("./utils.js");
function encodeArray(param, values) {
    if (!Array.isArray(values)) {
        throw new web3_errors_1.AbiError('Expected value to be array', { abi: param, values });
    }
    const { size, param: arrayItemParam } = (0, utils_js_1.extractArrayType)(param);
    const encodedParams = values.map(v => (0, index_js_1.encodeParamFromAbiParameter)(arrayItemParam, v));
    const dynamic = size === -1;
    const dynamicItems = encodedParams.length > 0 && encodedParams[0].dynamic;
    if (!dynamic && values.length !== size) {
        throw new web3_errors_1.AbiError("Given arguments count doesn't match array length", {
            arrayLength: size,
            argumentsLength: values.length,
        });
    }
    if (dynamic || dynamicItems) {
        const encodingResult = (0, utils_js_2.encodeDynamicParams)(encodedParams);
        if (dynamic) {
            const encodedLength = (0, index_js_1.encodeNumber)({ type: 'uint256', name: '' }, encodedParams.length).encoded;
            return {
                dynamic: true,
                encoded: encodedParams.length > 0
                    ? (0, web3_utils_1.uint8ArrayConcat)(encodedLength, encodingResult)
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
        encoded: (0, web3_utils_1.uint8ArrayConcat)(...encodedParams.map(p => p.encoded)),
    };
}
function decodeArray(param, bytes) {
    // eslint-disable-next-line prefer-const
    let { size, param: arrayItemParam } = (0, utils_js_1.extractArrayType)(param);
    const dynamic = size === -1;
    let consumed = 0;
    const result = [];
    let remaining = bytes;
    // dynamic array, we need to decode length
    if (dynamic) {
        const lengthResult = (0, number_js_1.decodeNumber)({ type: 'uint32', name: '' }, bytes);
        size = Number(lengthResult.result);
        consumed = lengthResult.consumed;
        remaining = lengthResult.encoded;
    }
    const hasDynamicChild = (0, utils_js_1.isDynamic)(arrayItemParam);
    if (hasDynamicChild) {
        // known length but dynamic child, each child is actually head element with encoded offset
        for (let i = 0; i < size; i += 1) {
            const offsetResult = (0, number_js_1.decodeNumber)({ type: 'uint32', name: '' }, remaining.subarray(i * utils_js_1.WORD_SIZE));
            consumed += offsetResult.consumed;
            const decodedChildResult = (0, index_js_1.decodeParamFromAbiParameter)(arrayItemParam, remaining.subarray(Number(offsetResult.result)));
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
        const decodedChildResult = (0, index_js_1.decodeParamFromAbiParameter)(arrayItemParam, bytes.subarray(consumed));
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