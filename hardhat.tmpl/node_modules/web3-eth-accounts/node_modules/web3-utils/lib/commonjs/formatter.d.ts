import { DataFormat, FormatType } from 'web3-types';
import { JsonSchema, ValidationSchemaInput } from 'web3-validator';
export declare const isDataFormat: (dataFormat: unknown) => dataFormat is DataFormat;
/**
 * Converts a value depending on the format
 * @param value - value to convert
 * @param ethType - The type of the value to be parsed
 * @param format - The format to be converted to
 * @returns - The value converted to the specified format
 */
export declare const convertScalarValue: (value: unknown, ethType: string, format: DataFormat) => unknown;
/**
 * Converts the data to the specified format
 * @param data - data to convert
 * @param schema - The JSON schema that describes the structure of the data
 * @param dataPath - A string array that specifies the path to the data within the JSON schema
 * @param format  - The format to be converted to
 * @param oneOfPath - An optional array of two-element tuples that specifies the "oneOf" option to choose, if the schema has oneOf and the data path can match multiple subschemas
 * @returns - The data converted to the specified format
 */
export declare const convert: (data: Record<string, unknown> | unknown[] | unknown, schema: JsonSchema, dataPath: string[], format: DataFormat, oneOfPath?: [string, number][]) => unknown;
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
export declare const format: <DataType extends Record<string, unknown> | unknown[] | unknown, ReturnType extends DataFormat>(schema: ValidationSchemaInput | JsonSchema, data: DataType, returnFormat?: ReturnType) => FormatType<DataType, ReturnType>;
