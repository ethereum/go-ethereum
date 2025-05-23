export type EncoderResult = {
    dynamic: boolean;
    encoded: Uint8Array;
};
export type DecoderResult<T = unknown> = {
    result: T;
    encoded: Uint8Array;
    consumed: number;
};
export type NumberType = {
    signed: boolean;
    byteLength: number;
};
export type BytesType = {
    size?: number;
};
