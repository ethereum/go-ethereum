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
import { isNullish, isUint8Array, leftPad, rightPad, toHex } from 'web3-utils';
export const isAbiFragment = (item) => !isNullish(item) &&
    typeof item === 'object' &&
    !isNullish(item.type) &&
    ['function', 'event', 'constructor', 'error'].includes(item.type);
export const isAbiErrorFragment = (item) => !isNullish(item) &&
    typeof item === 'object' &&
    !isNullish(item.type) &&
    item.type === 'error';
export const isAbiEventFragment = (item) => !isNullish(item) &&
    typeof item === 'object' &&
    !isNullish(item.type) &&
    item.type === 'event';
export const isAbiFunctionFragment = (item) => !isNullish(item) &&
    typeof item === 'object' &&
    !isNullish(item.type) &&
    item.type === 'function';
export const isAbiConstructorFragment = (item) => !isNullish(item) &&
    typeof item === 'object' &&
    !isNullish(item.type) &&
    item.type === 'constructor';
/**
 * Check if type is simplified struct format
 */
export const isSimplifiedStructFormat = (type) => typeof type === 'object' &&
    typeof type.components === 'undefined' &&
    typeof type.name === 'undefined';
/**
 * Maps the correct tuple type and name when the simplified format in encode/decodeParameter is used
 */
export const mapStructNameAndType = (structName) => structName.includes('[]')
    ? { type: 'tuple[]', name: structName.slice(0, -2) }
    : { type: 'tuple', name: structName };
/**
 * Maps the simplified format in to the expected format of the ABICoder
 */
export const mapStructToCoderFormat = (struct) => {
    const components = [];
    for (const key of Object.keys(struct)) {
        const item = struct[key];
        if (typeof item === 'object') {
            components.push(Object.assign(Object.assign({}, mapStructNameAndType(key)), { components: mapStructToCoderFormat(item) }));
        }
        else {
            components.push({
                name: key,
                type: struct[key],
            });
        }
    }
    return components;
};
/**
 * Map types if simplified format is used
 */
export const mapTypes = (types) => {
    const mappedTypes = [];
    for (const type of types) {
        let modifiedType = type;
        // Clone object
        if (typeof type === 'object') {
            modifiedType = Object.assign({}, type);
        }
        // Remap `function` type params to bytes24 since Ethers does not
        // recognize former type. Solidity docs say `Function` is a bytes24
        // encoding the contract address followed by the function selector hash.
        if (typeof type === 'object' && type.type === 'function') {
            modifiedType = Object.assign(Object.assign({}, type), { type: 'bytes24' });
        }
        if (isSimplifiedStructFormat(modifiedType)) {
            const structName = Object.keys(modifiedType)[0];
            mappedTypes.push(Object.assign(Object.assign({}, mapStructNameAndType(structName)), { components: mapStructToCoderFormat(modifiedType[structName]) }));
        }
        else {
            mappedTypes.push(modifiedType);
        }
    }
    return mappedTypes;
};
/**
 * returns true if input is a hexstring and is odd-lengthed
 */
export const isOddHexstring = (param) => typeof param === 'string' && /^(-)?0x[0-9a-f]*$/i.test(param) && param.length % 2 === 1;
/**
 * format odd-length bytes to even-length
 */
export const formatOddHexstrings = (param) => isOddHexstring(param) ? `0x0${param.substring(2)}` : param;
const paramTypeBytes = /^bytes([0-9]*)$/;
const paramTypeBytesArray = /^bytes([0-9]*)\[\]$/;
const paramTypeNumber = /^(u?int)([0-9]*)$/;
const paramTypeNumberArray = /^(u?int)([0-9]*)\[\]$/;
/**
 * Handle some formatting of params for backwards compatibility with Ethers V4
 */
export const formatParam = (type, _param) => {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    // clone if _param is an object
    const param = typeof _param === 'object' && !Array.isArray(_param) ? Object.assign({}, _param) : _param;
    // Format BN to string
    if (param instanceof BigInt || typeof param === 'bigint') {
        return param.toString(10);
    }
    if (paramTypeBytesArray.exec(type) || paramTypeNumberArray.exec(type)) {
        // eslint-disable-next-line @typescript-eslint/no-unsafe-return
        const paramClone = [...param];
        return paramClone.map(p => formatParam(type.replace('[]', ''), p));
    }
    // Format correct width for u?int[0-9]*
    let match = paramTypeNumber.exec(type);
    if (match) {
        const size = parseInt(match[2] ? match[2] : '256', 10);
        if (size / 8 < param.length) {
            // pad to correct bit width
            return leftPad(param, size);
        }
    }
    // Format correct length for bytes[0-9]+
    match = paramTypeBytes.exec(type);
    if (match) {
        const hexParam = isUint8Array(param) ? toHex(param) : param;
        // format to correct length
        const size = parseInt(match[1], 10);
        if (size) {
            let maxSize = size * 2;
            if (param.startsWith('0x')) {
                maxSize += 2;
            }
            // pad to correct length
            const paddedParam = hexParam.length < maxSize
                ? rightPad(param, size * 2)
                : hexParam;
            return formatOddHexstrings(paddedParam);
        }
        return formatOddHexstrings(hexParam);
    }
    return param;
};
/**
 *  used to flatten json abi inputs/outputs into an array of type-representing-strings
 */
export const flattenTypes = (includeTuple, puts) => {
    const types = [];
    puts.forEach(param => {
        if (typeof param.components === 'object') {
            if (!param.type.startsWith('tuple')) {
                throw new AbiError(`Invalid value given "${param.type}". Error: components found but type is not tuple.`);
            }
            const arrayBracket = param.type.indexOf('[');
            const suffix = arrayBracket >= 0 ? param.type.substring(arrayBracket) : '';
            const result = flattenTypes(includeTuple, param.components);
            if (Array.isArray(result) && includeTuple) {
                types.push(`tuple(${result.join(',')})${suffix}`);
            }
            else if (!includeTuple) {
                types.push(`(${result.join(',')})${suffix}`);
            }
            else {
                types.push(`(${result.join()})`);
            }
        }
        else {
            types.push(param.type);
        }
    });
    return types;
};
/**
 * Should be used to create full function/event name from json abi
 * returns a string
 */
export const jsonInterfaceMethodToString = (json) => {
    var _a, _b, _c, _d;
    // eslint-disable-next-line @typescript-eslint/prefer-nullish-coalescing
    if (isAbiErrorFragment(json) || isAbiEventFragment(json) || isAbiFunctionFragment(json)) {
        if ((_a = json.name) === null || _a === void 0 ? void 0 : _a.includes('(')) {
            return json.name;
        }
        return `${(_b = json.name) !== null && _b !== void 0 ? _b : ''}(${flattenTypes(false, (_c = json.inputs) !== null && _c !== void 0 ? _c : []).join(',')})`;
    }
    // Constructor fragment
    return `(${flattenTypes(false, (_d = json.inputs) !== null && _d !== void 0 ? _d : []).join(',')})`;
};
//# sourceMappingURL=utils.js.map