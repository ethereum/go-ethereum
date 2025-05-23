"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.format = exports.convert = exports.convertScalarValue = exports.isDataFormat = void 0;
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
const web3_types_1 = require("web3-types");
const web3_validator_1 = require("web3-validator");
const converters_js_1 = require("./converters.js");
const objects_js_1 = require("./objects.js");
const string_manipulation_js_1 = require("./string_manipulation.js");
const uint8array_js_1 = require("./uint8array.js");
const { parseBaseType } = web3_validator_1.utils;
const isDataFormat = (dataFormat) => typeof dataFormat === 'object' &&
    !(0, web3_validator_1.isNullish)(dataFormat) &&
    'number' in dataFormat &&
    'bytes' in dataFormat;
exports.isDataFormat = isDataFormat;
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
        else if (result.items && (0, web3_validator_1.isObject)(result.items)) {
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
const convertScalarValue = (value, ethType, format) => {
    try {
        const { baseType, baseTypeSize } = parseBaseType(ethType);
        if (baseType === 'int' || baseType === 'uint') {
            switch (format.number) {
                case web3_types_1.FMT_NUMBER.NUMBER:
                    return Number((0, converters_js_1.toBigInt)(value));
                case web3_types_1.FMT_NUMBER.HEX:
                    return (0, converters_js_1.numberToHex)((0, converters_js_1.toBigInt)(value));
                case web3_types_1.FMT_NUMBER.STR:
                    return (0, converters_js_1.toBigInt)(value).toString();
                case web3_types_1.FMT_NUMBER.BIGINT:
                    return (0, converters_js_1.toBigInt)(value);
                default:
                    throw new web3_errors_1.FormatterError(`Invalid format: ${String(format.number)}`);
            }
        }
        if (baseType === 'bytes') {
            let paddedValue;
            if (baseTypeSize) {
                if (typeof value === 'string')
                    paddedValue = (0, string_manipulation_js_1.padLeft)(value, baseTypeSize * 2);
                else if ((0, uint8array_js_1.isUint8Array)(value)) {
                    paddedValue = (0, uint8array_js_1.uint8ArrayConcat)(new Uint8Array(baseTypeSize - value.length), value);
                }
            }
            else {
                paddedValue = value;
            }
            switch (format.bytes) {
                case web3_types_1.FMT_BYTES.HEX:
                    return (0, converters_js_1.bytesToHex)((0, converters_js_1.bytesToUint8Array)(paddedValue));
                case web3_types_1.FMT_BYTES.UINT8ARRAY:
                    return (0, converters_js_1.bytesToUint8Array)(paddedValue);
                default:
                    throw new web3_errors_1.FormatterError(`Invalid format: ${String(format.bytes)}`);
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
exports.convertScalarValue = convertScalarValue;
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
        if ((0, web3_validator_1.isNullish)(_schemaProp === null || _schemaProp === void 0 ? void 0 : _schemaProp.items)) {
            // Can not find schema for array item, delete that item
            // eslint-disable-next-line no-param-reassign
            delete object[key];
            dataPath.pop();
            return true;
        }
        // If schema for array items is a single type
        if ((0, web3_validator_1.isObject)(_schemaProp.items) && !(0, web3_validator_1.isNullish)(_schemaProp.items.format)) {
            for (let i = 0; i < value.length; i += 1) {
                // eslint-disable-next-line no-param-reassign
                object[key][i] = (0, exports.convertScalarValue)(value[i], 
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
                (0, exports.convert)(arrObject, schema, dataPath, format, oneOfPath);
            }
            dataPath.pop();
            return true;
        }
        // If schema for array is a tuple
        if (Array.isArray(_schemaProp === null || _schemaProp === void 0 ? void 0 : _schemaProp.items)) {
            for (let i = 0; i < value.length; i += 1) {
                // eslint-disable-next-line no-param-reassign
                object[key][i] = (0, exports.convertScalarValue)(value[i], _schemaProp.items[i].format, format);
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
const convert = (data, schema, dataPath, format, oneOfPath = []) => {
    var _a;
    // If it's a scalar value
    if (!(0, web3_validator_1.isObject)(data) && !Array.isArray(data)) {
        return (0, exports.convertScalarValue)(data, schema === null || schema === void 0 ? void 0 : schema.format, format);
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
            if ((0, web3_validator_1.isNullish)(schemaProp)) {
                delete object[key];
                dataPath.pop();
                continue;
            }
            // If value is an object, recurse into it
            if ((0, web3_validator_1.isObject)(value)) {
                (0, exports.convert)(value, schema, dataPath, format, oneOfPath);
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
            object[key] = (0, exports.convertScalarValue)(value, schemaProp.format, format);
            dataPath.pop();
        }
    }
    return object;
};
exports.convert = convert;
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
const format = (schema, data, returnFormat = web3_types_1.DEFAULT_RETURN_FORMAT) => {
    let dataToParse;
    if ((0, web3_validator_1.isObject)(data)) {
        dataToParse = (0, objects_js_1.mergeDeep)({}, data);
    }
    else if (Array.isArray(data)) {
        dataToParse = [...data];
    }
    else {
        dataToParse = data;
    }
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    const jsonSchema = (0, web3_validator_1.isObject)(schema) ? schema : web3_validator_1.utils.ethAbiToJsonSchema(schema);
    if (!jsonSchema.properties && !jsonSchema.items && !jsonSchema.format) {
        throw new web3_errors_1.FormatterError('Invalid json schema for formatting');
    }
    return (0, exports.convert)(dataToParse, jsonSchema, [], returnFormat);
};
exports.format = format;
//# sourceMappingURL=formatter.js.map