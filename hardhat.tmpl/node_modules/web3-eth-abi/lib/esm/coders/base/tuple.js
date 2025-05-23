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
import { decodeParamFromAbiParameter, encodeParamFromAbiParameter } from './index.js';
import { encodeDynamicParams } from './utils.js';
import { isDynamic } from '../utils.js';
import { decodeNumber } from './number.js';
export function encodeTuple(param, input) {
    var _a, _b, _c;
    let dynamic = false;
    if (!Array.isArray(input) && typeof input !== 'object') {
        throw new AbiError('param must be either Array or Object', {
            param,
            input,
        });
    }
    const narrowedInput = input;
    const encoded = [];
    for (let i = 0; i < ((_b = (_a = param.components) === null || _a === void 0 ? void 0 : _a.length) !== null && _b !== void 0 ? _b : 0); i += 1) {
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        const paramComponent = param.components[i];
        let result;
        if (Array.isArray(narrowedInput)) {
            if (i >= narrowedInput.length) {
                throw new AbiError('input param length missmatch', {
                    param,
                    input,
                });
            }
            result = encodeParamFromAbiParameter(paramComponent, narrowedInput[i]);
        }
        else {
            const paramInput = narrowedInput[(_c = paramComponent.name) !== null && _c !== void 0 ? _c : ''];
            // eslint-disable-next-line no-null/no-null
            if (paramInput === undefined || paramInput === null) {
                throw new AbiError('missing input defined in abi', {
                    param,
                    input,
                    paramName: paramComponent.name,
                });
            }
            result = encodeParamFromAbiParameter(paramComponent, paramInput);
        }
        if (result.dynamic) {
            dynamic = true;
        }
        encoded.push(result);
    }
    if (dynamic) {
        return {
            dynamic: true,
            encoded: encodeDynamicParams(encoded),
        };
    }
    return {
        dynamic: false,
        encoded: uint8ArrayConcat(...encoded.map(e => e.encoded)),
    };
}
export function decodeTuple(param, bytes) {
    const result = {
        __length__: 0,
    };
    // tracks how much static params consumed bytes
    let consumed = 0;
    if (!param.components) {
        return {
            result,
            encoded: bytes,
            consumed,
        };
    }
    // track how much dynamic params consumed bytes
    let dynamicConsumed = 0;
    for (const [index, childParam] of param.components.entries()) {
        let decodedResult;
        if (isDynamic(childParam)) {
            // if dynamic, we will have offset encoded
            const offsetResult = decodeNumber({ type: 'uint32', name: '' }, bytes.subarray(consumed));
            // offset counts from start of original byte sequence
            decodedResult = decodeParamFromAbiParameter(childParam, bytes.subarray(Number(offsetResult.result)));
            consumed += offsetResult.consumed;
            dynamicConsumed += decodedResult.consumed;
        }
        else {
            // static param, just decode
            decodedResult = decodeParamFromAbiParameter(childParam, bytes.subarray(consumed));
            consumed += decodedResult.consumed;
        }
        result.__length__ += 1;
        result[index] = decodedResult.result;
        if (childParam.name && childParam.name !== '') {
            result[childParam.name] = decodedResult.result;
        }
    }
    return {
        encoded: bytes.subarray(consumed + dynamicConsumed),
        result,
        consumed: consumed + dynamicConsumed,
    };
}
//# sourceMappingURL=tuple.js.map