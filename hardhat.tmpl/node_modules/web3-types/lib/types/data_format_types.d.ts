import { Bytes, HexString, Numbers } from './primitives_types.js';
export declare enum FMT_NUMBER {
    NUMBER = "NUMBER_NUMBER",
    HEX = "NUMBER_HEX",
    STR = "NUMBER_STR",
    BIGINT = "NUMBER_BIGINT"
}
export type NumberTypes = {
    [FMT_NUMBER.NUMBER]: number;
    [FMT_NUMBER.HEX]: HexString;
    [FMT_NUMBER.STR]: string;
    [FMT_NUMBER.BIGINT]: bigint;
};
export declare enum FMT_BYTES {
    HEX = "BYTES_HEX",
    UINT8ARRAY = "BYTES_UINT8ARRAY"
}
export type ByteTypes = {
    [FMT_BYTES.HEX]: HexString;
    [FMT_BYTES.UINT8ARRAY]: Uint8Array;
};
/**
 * Used to specify how data should be formatted. Bytes can be formatted as hexadecimal strings or
 * Uint8Arrays. Numbers can be formatted as BigInts, hexadecimal strings, primitive numbers, or
 * strings.
 */
export type DataFormat = {
    readonly number: FMT_NUMBER;
    readonly bytes: FMT_BYTES;
};
export declare const DEFAULT_RETURN_FORMAT: {
    readonly number: FMT_NUMBER.BIGINT;
    readonly bytes: FMT_BYTES.HEX;
};
export declare const ETH_DATA_FORMAT: {
    readonly number: FMT_NUMBER.HEX;
    readonly bytes: FMT_BYTES.HEX;
};
export type FormatType<T, F extends DataFormat> = number extends Extract<T, Numbers> ? NumberTypes[F['number']] | Exclude<T, Numbers> : Uint8Array extends Extract<T, Bytes> ? ByteTypes[F['bytes']] | Exclude<T, Bytes> : T extends object | undefined ? {
    [P in keyof T]: FormatType<T[P], F>;
} : T;
//# sourceMappingURL=data_format_types.d.ts.map