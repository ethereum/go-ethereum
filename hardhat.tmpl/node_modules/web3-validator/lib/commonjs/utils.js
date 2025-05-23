"use strict";
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
Object.defineProperty(exports, "__esModule", { value: true });
exports.ensureIfUint8Array = exports.hexToUint8Array = exports.uint8ArrayToHexString = exports.padLeft = exports.numberToHex = exports.hexToNumber = exports.codePointToInt = exports.transformJsonDataToAbiFormat = exports.fetchArrayElement = exports.ethAbiToJsonSchema = exports.abiSchemaToJsonSchema = exports.parseBaseType = void 0;
const web3_errors_1 = require("web3-errors");
const constants_js_1 = require("./constants.js");
const abi_js_1 = require("./validation/abi.js");
const string_js_1 = require("./validation/string.js");
const errors_js_1 = require("./errors.js");
const extraTypes = ['hex', 'number', 'blockNumber', 'blockNumberOrTag', 'filter', 'bloom'];
const parseBaseType = (type) => {
    // Remove all empty spaces to avoid any parsing issue.
    let strippedType = type.replace(/ /, '');
    let baseTypeSize;
    let isArray = false;
    let arraySizes = [];
    if (type.includes('[')) {
        // Extract the array type
        strippedType = strippedType.slice(0, strippedType.indexOf('['));
        // Extract array indexes
        arraySizes = [...type.matchAll(/(?:\[(\d*)\])/g)]
            .map(match => parseInt(match[1], 10))
            .map(size => (Number.isNaN(size) ? -1 : size));
        isArray = arraySizes.length > 0;
    }
    if (constants_js_1.VALID_ETH_BASE_TYPES.includes(strippedType)) {
        return { baseType: strippedType, isArray, baseTypeSize, arraySizes };
    }
    if (strippedType.startsWith('int')) {
        baseTypeSize = parseInt(strippedType.substring(3), 10);
        strippedType = 'int';
    }
    else if (strippedType.startsWith('uint')) {
        baseTypeSize = parseInt(type.substring(4), 10);
        strippedType = 'uint';
    }
    else if (strippedType.startsWith('bytes')) {
        baseTypeSize = parseInt(strippedType.substring(5), 10);
        strippedType = 'bytes';
    }
    else {
        return { baseType: undefined, isArray: false, baseTypeSize: undefined, arraySizes };
    }
    return { baseType: strippedType, isArray, baseTypeSize, arraySizes };
};
exports.parseBaseType = parseBaseType;
const convertEthType = (type, parentSchema = {}) => {
    const typePropertyPresent = Object.keys(parentSchema).includes('type');
    if (typePropertyPresent) {
        throw new errors_js_1.Web3ValidatorError([
            {
                keyword: 'eth',
                message: 'Either "eth" or "type" can be presented in schema',
                params: { eth: type },
                instancePath: '',
                schemaPath: '',
            },
        ]);
    }
    const { baseType, baseTypeSize } = (0, exports.parseBaseType)(type);
    if (!baseType && !extraTypes.includes(type)) {
        throw new errors_js_1.Web3ValidatorError([
            {
                keyword: 'eth',
                message: `Eth data type "${type}" is not valid`,
                params: { eth: type },
                instancePath: '',
                schemaPath: '',
            },
        ]);
    }
    if (baseType) {
        if (baseType === 'tuple') {
            throw new Error('"tuple" type is not implemented directly.');
        }
        return { format: `${baseType}${baseTypeSize !== null && baseTypeSize !== void 0 ? baseTypeSize : ''}`, required: true };
    }
    if (type) {
        return { format: type, required: true };
    }
    return {};
};
const abiSchemaToJsonSchema = (abis, level = '/0') => {
    const schema = {
        type: 'array',
        items: [],
        maxItems: abis.length,
        minItems: abis.length,
    };
    for (const [index, abi] of abis.entries()) {
        // eslint-disable-next-line no-nested-ternary
        let abiType;
        let abiName;
        let abiComponents = [];
        // If it's a complete Abi Parameter
        // e.g. {name: 'a', type: 'uint'}
        if ((0, abi_js_1.isAbiParameterSchema)(abi)) {
            abiType = abi.type;
            abiName = abi.name || `${level}/${index}`;
            abiComponents = abi.components;
            // If its short form string value e.g. ['uint']
        }
        else if (typeof abi === 'string') {
            abiType = abi;
            abiName = `${level}/${index}`;
            // If it's provided in short form of tuple e.g. [['uint', 'string']]
        }
        else if (Array.isArray(abi)) {
            // If its custom tuple e.g. ['tuple[2]', ['uint', 'string']]
            if (abi[0] &&
                typeof abi[0] === 'string' &&
                abi[0].startsWith('tuple') &&
                !Array.isArray(abi[0]) &&
                abi[1] &&
                Array.isArray(abi[1])) {
                // eslint-disable-next-line prefer-destructuring
                abiType = abi[0];
                abiName = `${level}/${index}`;
                abiComponents = abi[1];
            }
            else {
                abiType = 'tuple';
                abiName = `${level}/${index}`;
                abiComponents = abi;
            }
        }
        const { baseType, isArray, arraySizes } = (0, exports.parseBaseType)(abiType);
        let childSchema;
        let lastSchema = schema;
        for (let i = arraySizes.length - 1; i > 0; i -= 1) {
            childSchema = {
                type: 'array',
                $id: abiName,
                items: [],
                maxItems: arraySizes[i],
                minItems: arraySizes[i],
            };
            if (arraySizes[i] < 0) {
                delete childSchema.maxItems;
                delete childSchema.minItems;
            }
            // lastSchema.items is a Schema, concat with 'childSchema'
            if (!Array.isArray(lastSchema.items)) {
                lastSchema.items = [lastSchema.items, childSchema];
            } // lastSchema.items is an empty Scheme array, set it to 'childSchema'
            else if (lastSchema.items.length === 0) {
                lastSchema.items = [childSchema];
            } // lastSchema.items is a non-empty Scheme array, append 'childSchema'
            else {
                lastSchema.items.push(childSchema);
            }
            lastSchema = childSchema;
        }
        if (baseType === 'tuple' && !isArray) {
            const nestedTuple = (0, exports.abiSchemaToJsonSchema)(abiComponents, abiName);
            nestedTuple.$id = abiName;
            lastSchema.items.push(nestedTuple);
        }
        else if (baseType === 'tuple' && isArray) {
            const arraySize = arraySizes[0];
            const item = Object.assign({ type: 'array', $id: abiName, items: (0, exports.abiSchemaToJsonSchema)(abiComponents, abiName) }, (arraySize >= 0 && { minItems: arraySize, maxItems: arraySize }));
            lastSchema.items.push(item);
        }
        else if (isArray) {
            const arraySize = arraySizes[0];
            const item = Object.assign({ type: 'array', $id: abiName, items: convertEthType(abiType) }, (arraySize >= 0 && { minItems: arraySize, maxItems: arraySize }));
            lastSchema.items.push(item);
        }
        else if (Array.isArray(lastSchema.items)) {
            // Array of non-tuple items
            lastSchema.items.push(Object.assign({ $id: abiName }, convertEthType(abiType)));
        }
        else {
            // Nested object
            lastSchema.items.push(Object.assign({ $id: abiName }, convertEthType(abiType)));
        }
        lastSchema = schema;
    }
    return schema;
};
exports.abiSchemaToJsonSchema = abiSchemaToJsonSchema;
const ethAbiToJsonSchema = (abis) => (0, exports.abiSchemaToJsonSchema)(abis);
exports.ethAbiToJsonSchema = ethAbiToJsonSchema;
const fetchArrayElement = (data, level) => {
    if (level === 1) {
        return data;
    }
    return (0, exports.fetchArrayElement)(data[0], level - 1);
};
exports.fetchArrayElement = fetchArrayElement;
const transformJsonDataToAbiFormat = (abis, data, transformedData) => {
    const newData = [];
    for (const [index, abi] of abis.entries()) {
        // eslint-disable-next-line no-nested-ternary
        let abiType;
        let abiName;
        let abiComponents = [];
        // If it's a complete Abi Parameter
        // e.g. {name: 'a', type: 'uint'}
        if ((0, abi_js_1.isAbiParameterSchema)(abi)) {
            abiType = abi.type;
            abiName = abi.name;
            abiComponents = abi.components;
            // If its short form string value e.g. ['uint']
        }
        else if (typeof abi === 'string') {
            abiType = abi;
            // If it's provided in short form of tuple e.g. [['uint', 'string']]
        }
        else if (Array.isArray(abi)) {
            // If its custom tuple e.g. ['tuple[2]', ['uint', 'string']]
            if (abi[1] && Array.isArray(abi[1])) {
                abiType = abi[0];
                abiComponents = abi[1];
            }
            else {
                abiType = 'tuple';
                abiComponents = abi;
            }
        }
        const { baseType, isArray, arraySizes } = (0, exports.parseBaseType)(abiType);
        const dataItem = Array.isArray(data)
            ? data[index]
            : data[abiName];
        if (baseType === 'tuple' && !isArray) {
            newData.push((0, exports.transformJsonDataToAbiFormat)(abiComponents, dataItem, transformedData));
        }
        else if (baseType === 'tuple' && isArray) {
            const tupleData = [];
            for (const tupleItem of dataItem) {
                // Nested array
                if (arraySizes.length > 1) {
                    const nestedItems = (0, exports.fetchArrayElement)(tupleItem, arraySizes.length - 1);
                    const nestedData = [];
                    for (const nestedItem of nestedItems) {
                        nestedData.push((0, exports.transformJsonDataToAbiFormat)(abiComponents, nestedItem, transformedData));
                    }
                    tupleData.push(nestedData);
                }
                else {
                    tupleData.push((0, exports.transformJsonDataToAbiFormat)(abiComponents, tupleItem, transformedData));
                }
            }
            newData.push(tupleData);
        }
        else {
            newData.push(dataItem);
        }
    }
    // Have to reassign before pushing to transformedData
    // eslint-disable-next-line no-param-reassign
    transformedData = transformedData !== null && transformedData !== void 0 ? transformedData : [];
    transformedData.push(...newData);
    return transformedData;
};
exports.transformJsonDataToAbiFormat = transformJsonDataToAbiFormat;
/**
 * Code points to int
 */
const codePointToInt = (codePoint) => {
    if (codePoint >= 48 && codePoint <= 57) {
        /* ['0'..'9'] -> [0..9] */
        return codePoint - 48;
    }
    if (codePoint >= 65 && codePoint <= 70) {
        /* ['A'..'F'] -> [10..15] */
        return codePoint - 55;
    }
    if (codePoint >= 97 && codePoint <= 102) {
        /* ['a'..'f'] -> [10..15] */
        return codePoint - 87;
    }
    throw new Error(`Invalid code point: ${codePoint}`);
};
exports.codePointToInt = codePointToInt;
/**
 * Converts value to it's number representation
 */
const hexToNumber = (value) => {
    if (!(0, string_js_1.isHexStrict)(value)) {
        throw new Error('Invalid hex string');
    }
    const [negative, hexValue] = value.startsWith('-') ? [true, value.slice(1)] : [false, value];
    const num = BigInt(hexValue);
    if (num > Number.MAX_SAFE_INTEGER) {
        return negative ? -num : num;
    }
    if (num < Number.MIN_SAFE_INTEGER) {
        return num;
    }
    return negative ? -1 * Number(num) : Number(num);
};
exports.hexToNumber = hexToNumber;
/**
 * Converts value to it's hex representation
 */
const numberToHex = (value) => {
    if ((typeof value === 'number' || typeof value === 'bigint') && value < 0) {
        return `-0x${value.toString(16).slice(1)}`;
    }
    if ((typeof value === 'number' || typeof value === 'bigint') && value >= 0) {
        return `0x${value.toString(16)}`;
    }
    if (typeof value === 'string' && (0, string_js_1.isHexStrict)(value)) {
        const [negative, hex] = value.startsWith('-') ? [true, value.slice(1)] : [false, value];
        const hexValue = hex.split(/^(-)?0(x|X)/).slice(-1)[0];
        return `${negative ? '-' : ''}0x${hexValue.replace(/^0+/, '').toLowerCase()}`;
    }
    if (typeof value === 'string' && !(0, string_js_1.isHexStrict)(value)) {
        return (0, exports.numberToHex)(BigInt(value));
    }
    throw new web3_errors_1.InvalidNumberError(value);
};
exports.numberToHex = numberToHex;
/**
 * Adds a padding on the left of a string, if value is a integer or bigInt will be converted to a hex string.
 */
const padLeft = (value, characterAmount, sign = '0') => {
    if (typeof value === 'string' && !(0, string_js_1.isHexStrict)(value)) {
        return value.padStart(characterAmount, sign);
    }
    const hex = typeof value === 'string' && (0, string_js_1.isHexStrict)(value) ? value : (0, exports.numberToHex)(value);
    const [prefix, hexValue] = hex.startsWith('-') ? ['-0x', hex.slice(3)] : ['0x', hex.slice(2)];
    return `${prefix}${hexValue.padStart(characterAmount, sign)}`;
};
exports.padLeft = padLeft;
function uint8ArrayToHexString(uint8Array) {
    let hexString = '0x';
    for (const e of uint8Array) {
        const hex = e.toString(16);
        hexString += hex.length === 1 ? `0${hex}` : hex;
    }
    return hexString;
}
exports.uint8ArrayToHexString = uint8ArrayToHexString;
// for optimized technique for hex to bytes conversion
const charCodeMap = {
    zero: 48,
    nine: 57,
    A: 65,
    F: 70,
    a: 97,
    f: 102,
};
function charCodeToBase16(char) {
    if (char >= charCodeMap.zero && char <= charCodeMap.nine)
        return char - charCodeMap.zero;
    if (char >= charCodeMap.A && char <= charCodeMap.F)
        return char - (charCodeMap.A - 10);
    if (char >= charCodeMap.a && char <= charCodeMap.f)
        return char - (charCodeMap.a - 10);
    return undefined;
}
function hexToUint8Array(hex) {
    let offset = 0;
    if (hex.startsWith('0') && (hex[1] === 'x' || hex[1] === 'X')) {
        offset = 2;
    }
    if (hex.length % 2 !== 0) {
        throw new web3_errors_1.InvalidBytesError(`hex string has odd length: ${hex}`);
    }
    const length = (hex.length - offset) / 2;
    const bytes = new Uint8Array(length);
    for (let index = 0, j = offset; index < length; index += 1) {
        // eslint-disable-next-line no-plusplus
        const nibbleLeft = charCodeToBase16(hex.charCodeAt(j++));
        // eslint-disable-next-line no-plusplus
        const nibbleRight = charCodeToBase16(hex.charCodeAt(j++));
        if (nibbleLeft === undefined || nibbleRight === undefined) {
            throw new web3_errors_1.InvalidBytesError(`Invalid byte sequence ("${hex[j - 2]}${hex[j - 1]}" in "${hex}").`);
        }
        bytes[index] = nibbleLeft * 16 + nibbleRight;
    }
    return bytes;
}
exports.hexToUint8Array = hexToUint8Array;
// @TODO: Remove this function and its usages once all sub dependencies uses version 1.3.3 or above of @noble/hashes
function ensureIfUint8Array(data) {
    var _a;
    if (!(data instanceof Uint8Array) &&
        ((_a = data === null || data === void 0 ? void 0 : data.constructor) === null || _a === void 0 ? void 0 : _a.name) === 'Uint8Array') {
        return Uint8Array.from(data);
    }
    return data;
}
exports.ensureIfUint8Array = ensureIfUint8Array;
//# sourceMappingURL=utils.js.map