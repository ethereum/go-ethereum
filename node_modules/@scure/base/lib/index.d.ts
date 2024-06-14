/*! scure-base - MIT License (c) 2022 Paul Miller (paulmillr.com) */
/**
 * @__NO_SIDE_EFFECTS__
 */
export declare function assertNumber(n: number): void;
export interface Coder<F, T> {
    encode(from: F): T;
    decode(to: T): F;
}
export interface BytesCoder extends Coder<Uint8Array, string> {
    encode: (data: Uint8Array) => string;
    decode: (str: string) => Uint8Array;
}
type Chain = [Coder<any, any>, ...Coder<any, any>[]];
type Input<F> = F extends Coder<infer T, any> ? T : never;
type Output<F> = F extends Coder<any, infer T> ? T : never;
type First<T> = T extends [infer U, ...any[]] ? U : never;
type Last<T> = T extends [...any[], infer U] ? U : never;
type Tail<T> = T extends [any, ...infer U] ? U : never;
type AsChain<C extends Chain, Rest = Tail<C>> = {
    [K in keyof C]: Coder<Input<C[K]>, Input<K extends keyof Rest ? Rest[K] : any>>;
};
/**
 * @__NO_SIDE_EFFECTS__
 */
declare function chain<T extends Chain & AsChain<T>>(...args: T): Coder<Input<First<T>>, Output<Last<T>>>;
type Alphabet = string[] | string;
/**
 * Encodes integer radix representation to array of strings using alphabet and back
 * @__NO_SIDE_EFFECTS__
 */
declare function alphabet(alphabet: Alphabet): Coder<number[], string[]>;
/**
 * @__NO_SIDE_EFFECTS__
 */
declare function join(separator?: string): Coder<string[], string>;
/**
 * Pad strings array so it has integer number of bits
 * @__NO_SIDE_EFFECTS__
 */
declare function padding(bits: number, chr?: string): Coder<string[], string[]>;
/**
 * Slow: O(n^2) time complexity
 * @__NO_SIDE_EFFECTS__
 */
declare function convertRadix(data: number[], from: number, to: number): number[];
/**
 * Implemented with numbers, because BigInt is 5x slower
 * @__NO_SIDE_EFFECTS__
 */
declare function convertRadix2(data: number[], from: number, to: number, padding: boolean): number[];
/**
 * @__NO_SIDE_EFFECTS__
 */
declare function radix(num: number): Coder<Uint8Array, number[]>;
/**
 * If both bases are power of same number (like `2**8 <-> 2**64`),
 * there is a linear algorithm. For now we have implementation for power-of-two bases only.
 * @__NO_SIDE_EFFECTS__
 */
declare function radix2(bits: number, revPadding?: boolean): Coder<Uint8Array, number[]>;
/**
 * @__NO_SIDE_EFFECTS__
 */
declare function checksum(len: number, fn: (data: Uint8Array) => Uint8Array): Coder<Uint8Array, Uint8Array>;
export declare const utils: {
    alphabet: typeof alphabet;
    chain: typeof chain;
    checksum: typeof checksum;
    convertRadix: typeof convertRadix;
    convertRadix2: typeof convertRadix2;
    radix: typeof radix;
    radix2: typeof radix2;
    join: typeof join;
    padding: typeof padding;
};
export declare const base16: BytesCoder;
export declare const base32: BytesCoder;
export declare const base32nopad: BytesCoder;
export declare const base32hex: BytesCoder;
export declare const base32hexnopad: BytesCoder;
export declare const base32crockford: BytesCoder;
export declare const base64: BytesCoder;
export declare const base64nopad: BytesCoder;
export declare const base64url: BytesCoder;
export declare const base64urlnopad: BytesCoder;
export declare const base58: BytesCoder;
export declare const base58flickr: BytesCoder;
export declare const base58xrp: BytesCoder;
export declare const base58xmr: BytesCoder;
export declare const createBase58check: (sha256: (data: Uint8Array) => Uint8Array) => BytesCoder;
export declare const base58check: (sha256: (data: Uint8Array) => Uint8Array) => BytesCoder;
export interface Bech32Decoded<Prefix extends string = string> {
    prefix: Prefix;
    words: number[];
}
export interface Bech32DecodedWithArray<Prefix extends string = string> {
    prefix: Prefix;
    words: number[];
    bytes: Uint8Array;
}
export declare const bech32: {
    encode: <Prefix extends string>(prefix: Prefix, words: number[] | Uint8Array, limit?: number | false) => `${Lowercase<Prefix>}1${string}`;
    decode: {
        <Prefix_1 extends string>(str: `${Prefix_1}1${string}`, limit?: number | false): Bech32Decoded<Prefix_1>;
        (str: string, limit?: number | false): Bech32Decoded;
    };
    decodeToBytes: (str: string) => Bech32DecodedWithArray;
    decodeUnsafe: (str: string, limit?: number | false | undefined) => void | Bech32Decoded<string>;
    fromWords: (to: number[]) => Uint8Array;
    fromWordsUnsafe: (to: number[]) => void | Uint8Array;
    toWords: (from: Uint8Array) => number[];
};
export declare const bech32m: {
    encode: <Prefix extends string>(prefix: Prefix, words: number[] | Uint8Array, limit?: number | false) => `${Lowercase<Prefix>}1${string}`;
    decode: {
        <Prefix_1 extends string>(str: `${Prefix_1}1${string}`, limit?: number | false): Bech32Decoded<Prefix_1>;
        (str: string, limit?: number | false): Bech32Decoded;
    };
    decodeToBytes: (str: string) => Bech32DecodedWithArray;
    decodeUnsafe: (str: string, limit?: number | false | undefined) => void | Bech32Decoded<string>;
    fromWords: (to: number[]) => Uint8Array;
    fromWordsUnsafe: (to: number[]) => void | Uint8Array;
    toWords: (from: Uint8Array) => number[];
};
export declare const utf8: BytesCoder;
export declare const hex: BytesCoder;
declare const CODERS: {
    utf8: BytesCoder;
    hex: BytesCoder;
    base16: BytesCoder;
    base32: BytesCoder;
    base64: BytesCoder;
    base64url: BytesCoder;
    base58: BytesCoder;
    base58xmr: BytesCoder;
};
type CoderType = keyof typeof CODERS;
export declare const bytesToString: (type: CoderType, bytes: Uint8Array) => string;
export declare const str: (type: CoderType, bytes: Uint8Array) => string;
export declare const stringToBytes: (type: CoderType, str: string) => Uint8Array;
export declare const bytes: (type: CoderType, str: string) => Uint8Array;
export {};
//# sourceMappingURL=index.d.ts.map