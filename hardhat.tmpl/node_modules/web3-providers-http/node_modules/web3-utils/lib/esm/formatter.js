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
import { FormatterError } from 'web3-errors';
import { FMT_BYTES, FMT_NUMBER, DEFAULT_RETURN_FORMAT, } from 'web3-types';
import { isNullish, isObject, utils } from 'web3-validator';
import { bytesToUint8Array, bytesToHex, numberToHex, toBigInt } from './converters.js';
import { mergeDeep } from './objects.js';
import { padLeft } from './string_manipulation.js';
import { isUint8Array, uint8ArrayConcat } from './uint8array.js';
const { parseBaseType } = utils;
export const isDataFormat = (dataFormat) => typeof dataFormat === 'object' &&
    !isNullish(dataFormat) &&
    'number' in dataFormat &&
    'bytes' in dataFormat;
/**
 * Finds the schema that corresponds to a specific data path within a larger JSON schema.
 * It works by iterating over the dataPath array and traversing the JSON schema one step at a time until it reaches the end of the path.
 *
 * @param schema - represents a JSON schema, which is an object that describes the structure of JSON data
 * @param dataPath - represents an array of strings that specifies the path to the data within the JSON schema
 * @param oneOfPath - represents an optional array of two-element tuples that specifies the "oneOf" option to choose, if the schema has oneOf and the data path can match multiple subschemas
 * @returns the JSON schema that matches the data path
 *
 */
const findSchemaByDataPath = (schema, dataPath, oneOfPath = []) => {
    let result = Object.assign({}, schema);
    let previousDataPath;
    for (const dataPart of dataPath) {
        if (result.oneOf && previousDataPath) {
            const currentDataPath = previousDataPath;
            const path = oneOfPath.find(([key]) => key === currentDataPath);
            if (path && path[0] === previousDataPath) {
                // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment, @typescript-eslint/no-unsafe-member-access
                result = result.oneOf[path[1]];
            }
        }
        if (!result.properties && !result.items) {
            return undefined;
        }
        if (result.properties) {
            result = result.properties[dataPart];
        }
        else if (result.items && result.items.properties) {
            const node = result.items.properties;
            result = node[dataPart];
        }
        else if (result.items && isObject(result.items)) {
            result = result.items;
        }
        else if (result.items && Array.isArray(result.items)) {
            result = result.items[parseInt(dataPart, 10)];
        }
        if (result && dataPart)
            previousDataPath = dataPart;
    }
    return result;
};
/**
 * Converts a value depending on the format
 * @param value - value to convert
 * @param ethType - The type of the value to be parsed
 * @param format - The format to be converted to
 * @returns - The value converted to the specified format
 */
export const convertScalarValue = (value, ethType, format) => {
    try {
        const { baseType, baseTypeSize } = parseBaseType(ethType);
        if (baseType === 'int' || baseType === 'uint') {
            switch (format.number) {
                case FMT_NUMBER.NUMBER:
                    return Number(toBigInt(value));
                case FMT_NUMBER.HEX:
                    return numberToHex(toBigInt(value));
                case FMT_NUMBER.STR:
                    return toBigInt(value).toString();
                case FMT_NUMBER.BIGINT:
                    return toBigInt(value);
                default:
                    throw new FormatterError(`Invalid format: ${String(format.number)}`);
            }
        }
        if (baseType === 'bytes') {
            let paddedValue;
            if (baseTypeSize) {
                if (typeof value === 'string')
                    paddedValue = padLeft(value, baseTypeSize * 2);
                else if (isUint8Array(value)) {
                    paddedValue = uint8ArrayConcat(new Uint8Array(baseTypeSize - value.length), value);
                }
            }
            else {
                paddedValue = value;
            }
            switch (format.bytes) {
                case FMT_BYTES.HEX:
                    return bytesToHex(bytesToUint8Array(paddedValue));
                case FMT_BYTES.UINT8ARRAY:
                    return bytesToUint8Array(paddedValue);
                default:
                    throw new FormatterError(`Invalid format: ${String(format.bytes)}`);
            }
        }
        if (baseType === 'string') {
            return String(value);
        }
    }
    catch (error) {
        // If someone didn't use `eth` keyword we can return original value
        // as the scope of this code is formatting not validation
        return value;
    }
    return value;
};
const convertArray = ({ value, schemaProp, schema, object, key, dataPath, format, oneOfPath = [], }) => {
    var _a, _b;
    // If value is an array
    if (Array.isArray(value)) {
        let _schemaProp = schemaProp;
        // TODO This is a naive approach to solving the issue of
        // a schema using oneOf. This chunk of code was intended to handle
        // BlockSchema.transactions
        // TODO BlockSchema.transactions are not being formatted
        if ((schemaProp === null || schemaProp === void 0 ? void 0 : schemaProp.oneOf) !== undefined) {
            // The following code is basically saying:
            // if the schema specifies oneOf, then we are to loop
            // over each possible schema and check if they type of the schema
            // matches the type of value[0], and if so we use the oneOfSchemaProp
            // as the schema for formatting
            // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call
            schemaProp.oneOf.forEach((oneOfSchemaProp, index) => {
                var _a, _b;
                if (!Array.isArray(schemaProp === null || schemaProp === void 0 ? void 0 : schemaProp.items) &&
                    ((typeof value[0] === 'object' &&
                        ((_a = oneOfSchemaProp === null || oneOfSchemaProp === void 0 ? void 0 : oneOfSchemaProp.items) === null || _a === void 0 ? void 0 : _a.type) === 'object') ||
                        (typeof value[0] === 'string' &&
                            ((_b = oneOfSchemaProp === null || oneOfSchemaProp === void 0 ? void 0 : oneOfSchemaProp.items) === null || _b === void 0 ? void 0 : _b.type) !== 'object'))) {
                    _schemaProp = oneOfSchemaProp;
                    oneOfPath.push([key, index]);
                }
            });
        }
        if (isNullish(_schemaProp === null || _schemaProp === void 0 ? void 0 : _schemaProp.items)) {
            // Can not find schema for array item, delete that item
            // eslint-disable-next-line no-param-reassign
            delete object[key];
            dataPath.pop();
            return true;
        }
        // If schema for array items is a single type
        if (isObject(_schemaProp.items) && !isNullish(_schemaProp.items.format)) {
            for (let i = 0; i < value.length; i += 1) {
                // eslint-disable-next-line no-param-reassign
                object[key][i] = convertScalarValue(value[i], 
                // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
                (_a = _schemaProp === null || _schemaProp === void 0 ? void 0 : _schemaProp.items) === null || _a === void 0 ? void 0 : _a.format, format);
            }
            dataPath.pop();
            return true;
        }
        // If schema for array items is an object
        if (!Array.isArray(_schemaProp === null || _schemaProp === void 0 ? void 0 : _schemaProp.items) && ((_b = _schemaProp === null || _schemaProp === void 0 ? void 0 : _schemaProp.items) === null || _b === void 0 ? void 0 : _b.type) === 'object') {
            for (const arrObject of value) {
                // eslint-disable-next-line no-use-before-define
                convert(arrObject, schema, dataPath, format, oneOfPath);
            }
            dataPath.pop();
            return true;
        }
        // If schema for array is a tuple
        if (Array.isArray(_schemaProp === null || _schemaProp === void 0 ? void 0 : _schemaProp.items)) {
            for (let i = 0; i < value.length; i += 1) {
                // eslint-disable-next-line no-param-reassign
                object[key][i] = convertScalarValue(value[i], _schemaProp.items[i].format, format);
            }
            dataPath.pop();
            return true;
        }
    }
    return false;
};
/**
 * Converts the data to the specified format
 * @param data - data to convert
 * @param schema - The JSON schema that describes the structure of the data
 * @param dataPath - A string array that specifies the path to the data within the JSON schema
 * @param format  - The format to be converted to
 * @param oneOfPath - An optional array of two-element tuples that specifies the "oneOf" option to choose, if the schema has oneOf and the data path can match multiple subschemas
 * @returns - The data converted to the specified format
 */
export const convert = (data, schema, dataPath, format, oneOfPath = []) => {
    var _a;
    // If it's a scalar value
    if (!isObject(data) && !Array.isArray(data)) {
        return convertScalarValue(data, schema === null || schema === void 0 ? void 0 : schema.format, format);
    }
    const object = data;
    // case when schema is array and `items` is object
    if (Array.isArray(object) &&
        (schema === null || schema === void 0 ? void 0 : schema.type) === 'array' &&
        ((_a = schema === null || schema === void 0 ? void 0 : schema.items) === null || _a === void 0 ? void 0 : _a.type) === 'object') {
        convertArray({
            value: object,
            schemaProp: schema,
            schema,
            object,
            key: '',
            dataPath,
            format,
            oneOfPath,
        });
    }
    else {
        for (const [key, value] of Object.entries(object)) {
            dataPath.push(key);
            let schemaProp = findSchemaByDataPath(schema, dataPath, oneOfPath);
            // If value is a scaler value
            if (isNullish(schemaProp)) {
                delete object[key];
                dataPath.pop();
                continue;
            }
            // If value is an object, recurse into it
            if (isObject(value)) {
                convert(value, schema, dataPath, format, oneOfPath);
                dataPath.pop();
                continue;
            }
            // If value is an array
            if (convertArray({
                value,
                schemaProp,
                schema,
                object,
                key,
                dataPath,
                format,
                oneOfPath,
            })) {
                continue;
            }
            // The following code is basically saying:
            // if the schema specifies oneOf, then we are to loop
            // over each possible schema and check if they type of the schema specifies format
            // and if so we use the oneOfSchemaProp as the schema for formatting
            // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call
            if ((schemaProp === null || schemaProp === void 0 ? void 0 : schemaProp.format) === undefined && (schemaProp === null || schemaProp === void 0 ? void 0 : schemaProp.oneOf) !== undefined) {
                for (const [_index, oneOfSchemaProp] of schemaProp.oneOf.entries()) {
                    if ((oneOfSchemaProp === null || oneOfSchemaProp === void 0 ? void 0 : oneOfSchemaProp.format) !== undefined) {
                        schemaProp = oneOfSchemaProp;
                        break;
                    }
                }
            }
            object[key] = convertScalarValue(value, schemaProp.format, format);
            dataPath.pop();
        }
    }
    return object;
};
/**
 * Given data that can be interpreted according to the provided schema, returns equivalent data that has been formatted
 * according to the provided return format.
 *
 * @param schema - how to interpret the data
 * @param data - data to be formatted
 * @param returnFormat - how to format the data
 * @returns - formatted data
 *
 * @example
 *
 * ```js
 * import { FMT_NUMBER, utils } from "web3";
 *
 * console.log(
 *   utils.format({ format: "uint" }, "221", { number: FMT_NUMBER.HEX }),
 * );
 * // 0xdd
 * ```
 *
 */
export const format = (schema, data, returnFormat = DEFAULT_RETURN_FORMAT) => {
    let dataToParse;
    if (isObject(data)) {
        dataToParse = mergeDeep({}, data);
    }
    else if (Array.isArray(data)) {
        dataToParse = [...data];
    }
    else {
        dataToParse = data;
    }
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    const jsonSchema = isObject(schema) ? schema : utils.ethAbiToJsonSchema(schema);
    if (!jsonSchema.properties && !jsonSchema.items && !jsonSchema.format) {
        throw new FormatterError('Invalid json schema for formatting');
    }
    return convert(dataToParse, jsonSchema, [], returnFormat);
};
//# sourceMappingURL=formatter.js.map