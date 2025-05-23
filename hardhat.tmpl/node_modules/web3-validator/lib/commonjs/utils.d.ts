import { FullValidationSchema, ShortValidationSchema, ValidationSchemaInput, ValidInputTypes } from './types.js';
export declare const parseBaseType: <T = string>(type: string) => {
    baseType?: T | undefined;
    baseTypeSize: number | undefined;
    arraySizes: number[];
    isArray: boolean;
};
export declare const abiSchemaToJsonSchema: (abis: ShortValidationSchema | FullValidationSchema, level?: string) => import("./types.js").Schema;
export declare const ethAbiToJsonSchema: (abis: ValidationSchemaInput) => import("./types.js").Schema;
export declare const fetchArrayElement: (data: Array<unknown>, level: number) => unknown;
export declare const transformJsonDataToAbiFormat: (abis: FullValidationSchema, data: ReadonlyArray<unknown> | Record<string, unknown>, transformedData?: Array<unknown>) => Array<unknown>;
/**
 * Code points to int
 */
export declare const codePointToInt: (codePoint: number) => number;
/**
 * Converts value to it's number representation
 */
export declare const hexToNumber: (value: string) => bigint | number;
/**
 * Converts value to it's hex representation
 */
export declare const numberToHex: (value: ValidInputTypes) => string;
/**
 * Adds a padding on the left of a string, if value is a integer or bigInt will be converted to a hex string.
 */
export declare const padLeft: (value: ValidInputTypes, characterAmount: number, sign?: string) => string;
export declare function uint8ArrayToHexString(uint8Array: Uint8Array): string;
export declare function hexToUint8Array(hex: string): Uint8Array;
export declare function ensureIfUint8Array<T = any>(data: T): Uint8Array | T;
