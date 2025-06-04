export namespace MT {
    const POS_INT: number;
    const NEG_INT: number;
    const BYTE_STRING: number;
    const UTF8_STRING: number;
    const ARRAY: number;
    const MAP: number;
    const TAG: number;
    const SIMPLE_FLOAT: number;
}
export type MT = number;
export namespace TAG {
    const DATE_STRING: number;
    const DATE_EPOCH: number;
    const POS_BIGINT: number;
    const NEG_BIGINT: number;
    const DECIMAL_FRAC: number;
    const BIGFLOAT: number;
    const BASE64URL_EXPECTED: number;
    const BASE64_EXPECTED: number;
    const BASE16_EXPECTED: number;
    const CBOR: number;
    const URI: number;
    const BASE64URL: number;
    const BASE64: number;
    const REGEXP: number;
    const MIME: number;
    const SET: number;
}
export type TAG = number;
export namespace NUMBYTES {
    const ZERO: number;
    const ONE: number;
    const TWO: number;
    const FOUR: number;
    const EIGHT: number;
    const INDEFINITE: number;
}
export type NUMBYTES = number;
export namespace SIMPLE {
    const FALSE: number;
    const TRUE: number;
    const NULL: number;
    const UNDEFINED: number;
}
export type SIMPLE = number;
export namespace SYMS {
    const NULL_1: symbol;
    export { NULL_1 as NULL };
    const UNDEFINED_1: symbol;
    export { UNDEFINED_1 as UNDEFINED };
    export const PARENT: symbol;
    export const BREAK: symbol;
    export const STREAM: symbol;
}
export const SHIFT32: 4294967296;
export namespace BI {
    const MINUS_ONE: bigint;
    const NEG_MAX: bigint;
    const MAXINT32: bigint;
    const MAXINT64: bigint;
    const SHIFT32: bigint;
}
