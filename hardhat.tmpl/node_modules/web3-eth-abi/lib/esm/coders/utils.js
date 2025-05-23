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
import { parseAbiParameter } from 'abitype';
import { AbiError } from 'web3-errors';
import { isNullish } from 'web3-utils';
import { isSimplifiedStructFormat, mapStructNameAndType, mapStructToCoderFormat, } from '../utils.js';
export const WORD_SIZE = 32;
export function alloc(size = 0) {
    var _a;
    if (((_a = globalThis.Buffer) === null || _a === void 0 ? void 0 : _a.alloc) !== undefined) {
        const buf = globalThis.Buffer.alloc(size);
        return new Uint8Array(buf.buffer, buf.byteOffset, buf.byteLength);
    }
    return new Uint8Array(size);
}
/**
 * Where possible returns a Uint8Array of the requested size that references
 * uninitialized memory. Only use if you are certain you will immediately
 * overwrite every value in the returned `Uint8Array`.
 */
export function allocUnsafe(size = 0) {
    var _a;
    if (((_a = globalThis.Buffer) === null || _a === void 0 ? void 0 : _a.allocUnsafe) !== undefined) {
        const buf = globalThis.Buffer.allocUnsafe(size);
        return new Uint8Array(buf.buffer, buf.byteOffset, buf.byteLength);
    }
    return new Uint8Array(size);
}
export function convertExternalAbiParameter(abiParam) {
    var _a, _b;
    return Object.assign(Object.assign({}, abiParam), { name: (_a = abiParam.name) !== null && _a !== void 0 ? _a : '', components: (_b = abiParam.components) === null || _b === void 0 ? void 0 : _b.map(c => convertExternalAbiParameter(c)) });
}
export function isAbiParameter(param) {
    return (!isNullish(param) &&
        typeof param === 'object' &&
        !isNullish(param.type) &&
        typeof param.type === 'string');
}
export function toAbiParams(abi) {
    return abi.map(input => {
        var _a;
        if (isAbiParameter(input)) {
            return input;
        }
        if (typeof input === 'string') {
            return convertExternalAbiParameter(parseAbiParameter(input.replace(/tuple/, '')));
        }
        if (isSimplifiedStructFormat(input)) {
            const structName = Object.keys(input)[0];
            const structInfo = mapStructNameAndType(structName);
            structInfo.name = (_a = structInfo.name) !== null && _a !== void 0 ? _a : '';
            return Object.assign(Object.assign({}, structInfo), { components: mapStructToCoderFormat(input[structName]) });
        }
        throw new AbiError('Invalid abi');
    });
}
export function extractArrayType(param) {
    const arrayParenthesisStart = param.type.lastIndexOf('[');
    const arrayParamType = param.type.substring(0, arrayParenthesisStart);
    const sizeString = param.type.substring(arrayParenthesisStart);
    let size = -1;
    if (sizeString !== '[]') {
        size = Number(sizeString.slice(1, -1));
        // eslint-disable-next-line no-restricted-globals
        if (isNaN(size)) {
            throw new AbiError('Invalid fixed array size', { size: sizeString });
        }
    }
    return {
        param: { type: arrayParamType, name: '', components: param.components },
        size,
    };
}
/**
 * Param is dynamic if it's dynamic base type or if some of his children (components, array items)
 * is of dynamic type
 * @param param
 */
export function isDynamic(param) {
    var _a, _b;
    if (param.type === 'string' || param.type === 'bytes' || param.type.endsWith('[]'))
        return true;
    if (param.type === 'tuple') {
        return (_b = (_a = param.components) === null || _a === void 0 ? void 0 : _a.some(isDynamic)) !== null && _b !== void 0 ? _b : false;
    }
    if (param.type.endsWith(']')) {
        return isDynamic(extractArrayType(param).param);
    }
    return false;
}
//# sourceMappingURL=utils.js.map