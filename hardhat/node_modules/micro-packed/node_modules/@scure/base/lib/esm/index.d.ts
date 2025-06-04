/*! scure-base - MIT License (c) 2022 Paul Miller (paulmillr.com) */
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
/**
 * Encodes integer radix representation to array of strings using alphabet and back.
 * Could also be array of strings.
 * @__NO_SIDE_EFFECTS__
 */
declare function alphabet(letters: string | string[]): Coder<number[], string[]>;
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
 */
declare function convertRadix(data: number[], from: number, to: number): number[];
/**
 * Implemented with numbers, because BigInt is 5x slower
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
/**
 * base16 encoding from RFC 4648.
 * @example
 * ```js
 * base16.encode(Uint8Array.from([0x12, 0xab]));
 * // => '12AB'
 * ```
 */
export declare const base16: BytesCoder;
/**
 * base32 encoding from RFC 4648. Has padding.
 * Use `base32nopad` for unpadded version.
 * Also check out `base32hex`, `base32hexnopad`, `base32crockford`.
 * @example
 * ```js
 * base32.encode(Uint8Array.from([0x12, 0xab]));
 * // => 'CKVQ===='
 * base32.decode('CKVQ====');
 * // => Uint8Array.from([0x12, 0xab])
 * ```
 */
export declare const base32: BytesCoder;
/**
 * base32 encoding from RFC 4648. No padding.
 * Use `base32` for padded version.
 * Also check out `base32hex`, `base32hexnopad`, `base32crockford`.
 * @example
 * ```js
 * base32nopad.encode(Uint8Array.from([0x12, 0xab]));
 * // => 'CKVQ'
 * base32nopad.decode('CKVQ');
 * // => Uint8Array.from([0x12, 0xab])
 * ```
 */
export declare const base32nopad: BytesCoder;
/**
 * base32 encoding from RFC 4648. Padded. Compared to ordinary `base32`, slightly different alphabet.
 * Use `base32hexnopad` for unpadded version.
 * @example
 * ```js
 * base32hex.encode(Uint8Array.from([0x12, 0xab]));
 * // => '2ALG===='
 * base32hex.decode('2ALG====');
 * // => Uint8Array.from([0x12, 0xab])
 * ```
 */
export declare const base32hex: BytesCoder;
/**
 * base32 encoding from RFC 4648. No padding. Compared to ordinary `base32`, slightly different alphabet.
 * Use `base32hex` for padded version.
 * @example
 * ```js
 * base32hexnopad.encode(Uint8Array.from([0x12, 0xab]));
 * // => '2ALG'
 * base32hexnopad.decode('2ALG');
 * // => Uint8Array.from([0x12, 0xab])
 * ```
 */
export declare const base32hexnopad: BytesCoder;
/**
 * base32 encoding from RFC 4648. Doug Crockford's version.
 * https://www.crockford.com/base32.html
 * @example
 * ```js
 * base32crockford.encode(Uint8Array.from([0x12, 0xab]));
 * // => '2ANG'
 * base32crockford.decode('2ANG');
 * // => Uint8Array.from([0x12, 0xab])
 * ```
 */
export declare const base32crockford: BytesCoder;
/**
 * base64 from RFC 4648. Padded.
 * Use `base64nopad` for unpadded version.
 * Also check out `base64url`, `base64urlnopad`.
 * Falls back to built-in function, when available.
 * @example
 * ```js
 * base64.encode(Uint8Array.from([0x12, 0xab]));
 * // => 'Eqs='
 * base64.decode('Eqs=');
 * // => Uint8Array.from([0x12, 0xab])
 * ```
 */
export declare const base64: BytesCoder;
/**
 * base64 from RFC 4648. No padding.
 * Use `base64` for padded version.
 * @example
 * ```js
 * base64nopad.encode(Uint8Array.from([0x12, 0xab]));
 * // => 'Eqs'
 * base64nopad.decode('Eqs');
 * // => Uint8Array.from([0x12, 0xab])
 * ```
 */
export declare const base64nopad: BytesCoder;
/**
 * base64 from RFC 4648, using URL-safe alphabet. Padded.
 * Use `base64urlnopad` for unpadded version.
 * Falls back to built-in function, when available.
 * @example
 * ```js
 * base64url.encode(Uint8Array.from([0x12, 0xab]));
 * // => 'Eqs='
 * base64url.decode('Eqs=');
 * // => Uint8Array.from([0x12, 0xab])
 * ```
 */
export declare const base64url: BytesCoder;
/**
 * base64 from RFC 4648, using URL-safe alphabet. No padding.
 * Use `base64url` for padded version.
 * @example
 * ```js
 * base64urlnopad.encode(Uint8Array.from([0x12, 0xab]));
 * // => 'Eqs'
 * base64urlnopad.decode('Eqs');
 * // => Uint8Array.from([0x12, 0xab])
 * ```
 */
export declare const base64urlnopad: BytesCoder;
/**
 * base58: base64 without ambigous characters +, /, 0, O, I, l.
 * Quadratic (O(n^2)) - so, can't be used on large inputs.
 * @example
 * ```js
 * base58.decode('01abcdef');
 * // => '3UhJW'
 * ```
 */
export declare const base58: BytesCoder;
/**
 * base58: flickr version. Check out `base58`.
 */
export declare const base58flickr: BytesCoder;
/**
 * base58: XRP version. Check out `base58`.
 */
export declare const base58xrp: BytesCoder;
/**
 * base58: XMR version. Check out `base58`.
 * Done in 8-byte blocks (which equals 11 chars in decoding). Last (non-full) block padded with '1' to size in XMR_BLOCK_LEN.
 * Block encoding significantly reduces quadratic complexity of base58.
 */
export declare const base58xmr: BytesCoder;
/**
 * Method, which creates base58check encoder.
 * Requires function, calculating sha256.
 */
export declare const createBase58check: (sha256: (data: Uint8Array) => Uint8Array) => BytesCoder;
/**
 * Use `createBase58check` instead.
 * @deprecated
 */
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
export interface Bech32 {
    encode<Prefix extends string>(prefix: Prefix, words: number[] | Uint8Array, limit?: number | false): `${Lowercase<Prefix>}1${string}`;
    decode<Prefix extends string>(str: `${Prefix}1${string}`, limit?: number | false): Bech32Decoded<Prefix>;
    encodeFromBytes(prefix: string, bytes: Uint8Array): string;
    decodeToBytes(str: string): Bech32DecodedWithArray;
    decodeUnsafe(str: string, limit?: number | false): void | Bech32Decoded<string>;
    fromWords(to: number[]): Uint8Array;
    fromWordsUnsafe(to: number[]): void | Uint8Array;
    toWords(from: Uint8Array): number[];
}
/**
 * bech32 from BIP 173. Operates on words.
 * For high-level, check out scure-btc-signer:
 * https://github.com/paulmillr/scure-btc-signer.
 */
export declare const bech32: Bech32;
/**
 * bech32m from BIP 350. Operates on words.
 * It was to mitigate `bech32` weaknesses.
 * For high-level, check out scure-btc-signer:
 * https://github.com/paulmillr/scure-btc-signer.
 */
export declare const bech32m: Bech32;
/**
 * UTF-8-to-byte decoder. Uses built-in TextDecoder / TextEncoder.
 * @example
 * ```js
 * const b = utf8.decode("hey"); // => new Uint8Array([ 104, 101, 121 ])
 * const str = utf8.encode(b); // "hey"
 * ```
 */
export declare const utf8: BytesCoder;
/**
 * hex string decoder. Uses built-in function, when available.
 * @example
 * ```js
 * const b = hex.decode("0102ff"); // => new Uint8Array([ 1, 2, 255 ])
 * const str = hex.encode(b); // "0102ff"
 * ```
 */
export declare const hex: BytesCoder;
export type SomeCoders = {
    utf8: BytesCoder;
    hex: BytesCoder;
    base16: BytesCoder;
    base32: BytesCoder;
    base64: BytesCoder;
    base64url: BytesCoder;
    base58: BytesCoder;
    base58xmr: BytesCoder;
};
type CoderType = keyof SomeCoders;
/** @deprecated */
export declare const bytesToString: (type: CoderType, bytes: Uint8Array) => string;
/** @deprecated */
export declare const str: (type: CoderType, bytes: Uint8Array) => string;
/** @deprecated */
export declare const stringToBytes: (type: CoderType, str: string) => Uint8Array;
/** @deprecated */
export declare const bytes: (type: CoderType, str: string) => Uint8Array;
export {};
//# sourceMappingURL=index.d.ts.map