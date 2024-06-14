import { BytesLike } from "@ethersproject/bytes";
export declare enum UnicodeNormalizationForm {
    current = "",
    NFC = "NFC",
    NFD = "NFD",
    NFKC = "NFKC",
    NFKD = "NFKD"
}
export declare enum Utf8ErrorReason {
    UNEXPECTED_CONTINUE = "unexpected continuation byte",
    BAD_PREFIX = "bad codepoint prefix",
    OVERRUN = "string overrun",
    MISSING_CONTINUE = "missing continuation byte",
    OUT_OF_RANGE = "out of UTF-8 range",
    UTF16_SURROGATE = "UTF-16 surrogate",
    OVERLONG = "overlong representation"
}
export declare type Utf8ErrorFunc = (reason: Utf8ErrorReason, offset: number, bytes: ArrayLike<number>, output: Array<number>, badCodepoint?: number) => number;
export declare const Utf8ErrorFuncs: {
    [name: string]: Utf8ErrorFunc;
};
export declare function toUtf8Bytes(str: string, form?: UnicodeNormalizationForm): Uint8Array;
export declare function _toEscapedUtf8String(bytes: BytesLike, onError?: Utf8ErrorFunc): string;
export declare function _toUtf8String(codePoints: Array<number>): string;
export declare function toUtf8String(bytes: BytesLike, onError?: Utf8ErrorFunc): string;
export declare function toUtf8CodePoints(str: string, form?: UnicodeNormalizationForm): Array<number>;
//# sourceMappingURL=utf8.d.ts.map