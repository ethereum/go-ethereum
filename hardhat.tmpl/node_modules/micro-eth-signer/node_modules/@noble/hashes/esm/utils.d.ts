/**
 * Utilities for hex, bytes, CSPRNG.
 * @module
 */
/*! noble-hashes - MIT License (c) 2022 Paul Miller (paulmillr.com) */
export declare function isBytes(a: unknown): a is Uint8Array;
export type TypedArray = Int8Array | Uint8ClampedArray | Uint8Array | Uint16Array | Int16Array | Uint32Array | Int32Array;
export declare function u8(arr: TypedArray): Uint8Array;
export declare function u32(arr: TypedArray): Uint32Array;
export declare function createView(arr: TypedArray): DataView;
/** The rotate right (circular right shift) operation for uint32 */
export declare function rotr(word: number, shift: number): number;
/** The rotate left (circular left shift) operation for uint32 */
export declare function rotl(word: number, shift: number): number;
/** Is current platform little-endian? Most are. Big-Endian platform: IBM */
export declare const isLE: boolean;
export declare function byteSwap(word: number): number;
/** Conditionally byte swap if on a big-endian platform */
export declare const byteSwapIfBE: (n: number) => number;
/** In place byte swap for Uint32Array */
export declare function byteSwap32(arr: Uint32Array): void;
/**
 * Convert byte array to hex string. Uses built-in function, when available.
 * @example bytesToHex(Uint8Array.from([0xca, 0xfe, 0x01, 0x23])) // 'cafe0123'
 */
export declare function bytesToHex(bytes: Uint8Array): string;
/**
 * Convert hex string to byte array. Uses built-in function, when available.
 * @example hexToBytes('cafe0123') // Uint8Array.from([0xca, 0xfe, 0x01, 0x23])
 */
export declare function hexToBytes(hex: string): Uint8Array;
/**
 * There is no setImmediate in browser and setTimeout is slow.
 * Call of async fn will return Promise, which will be fullfiled only on
 * next scheduler queue processing step and this is exactly what we need.
 */
export declare const nextTick: () => Promise<void>;
/** Returns control to thread each 'tick' ms to avoid blocking. */
export declare function asyncLoop(iters: number, tick: number, cb: (i: number) => void): Promise<void>;
/**
 * Convert JS string to byte array.
 * @example utf8ToBytes('abc') // new Uint8Array([97, 98, 99])
 */
export declare function utf8ToBytes(str: string): Uint8Array;
/** Accepted input of hash functions. Strings are converted to byte arrays. */
export type Input = Uint8Array | string;
/**
 * Normalizes (non-hex) string or Uint8Array to Uint8Array.
 * Warning: when Uint8Array is passed, it would NOT get copied.
 * Keep in mind for future mutable operations.
 */
export declare function toBytes(data: Input): Uint8Array;
/**
 * Copies several Uint8Arrays into one.
 */
export declare function concatBytes(...arrays: Uint8Array[]): Uint8Array;
/** For runtime check if class implements interface */
export declare abstract class Hash<T extends Hash<T>> {
    abstract blockLen: number;
    abstract outputLen: number;
    abstract update(buf: Input): this;
    abstract digestInto(buf: Uint8Array): void;
    abstract digest(): Uint8Array;
    /**
     * Resets internal state. Makes Hash instance unusable.
     * Reset is impossible for keyed hashes if key is consumed into state. If digest is not consumed
     * by user, they will need to manually call `destroy()` when zeroing is necessary.
     */
    abstract destroy(): void;
    /**
     * Clones hash instance. Unsafe: doesn't check whether `to` is valid. Can be used as `clone()`
     * when no options are passed.
     * Reasons to use `_cloneInto` instead of clone: 1) performance 2) reuse instance => all internal
     * buffers are overwritten => causes buffer overwrite which is used for digest in some cases.
     * There are no guarantees for clean-up because it's impossible in JS.
     */
    abstract _cloneInto(to?: T): T;
    clone(): T;
}
/**
 * XOF: streaming API to read digest in chunks.
 * Same as 'squeeze' in keccak/k12 and 'seek' in blake3, but more generic name.
 * When hash used in XOF mode it is up to user to call '.destroy' afterwards, since we cannot
 * destroy state, next call can require more bytes.
 */
export type HashXOF<T extends Hash<T>> = Hash<T> & {
    xof(bytes: number): Uint8Array;
    xofInto(buf: Uint8Array): Uint8Array;
};
type EmptyObj = {};
export declare function checkOpts<T1 extends EmptyObj, T2 extends EmptyObj>(defaults: T1, opts?: T2): T1 & T2;
/** Hash function */
export type CHash = ReturnType<typeof wrapConstructor>;
/** Hash function with output */
export type CHashO = ReturnType<typeof wrapConstructorWithOpts>;
/** XOF with output */
export type CHashXO = ReturnType<typeof wrapXOFConstructorWithOpts>;
/** Wraps hash function, creating an interface on top of it */
export declare function wrapConstructor<T extends Hash<T>>(hashCons: () => Hash<T>): {
    (msg: Input): Uint8Array;
    outputLen: number;
    blockLen: number;
    create(): Hash<T>;
};
export declare function wrapConstructorWithOpts<H extends Hash<H>, T extends Object>(hashCons: (opts?: T) => Hash<H>): {
    (msg: Input, opts?: T): Uint8Array;
    outputLen: number;
    blockLen: number;
    create(opts: T): Hash<H>;
};
export declare function wrapXOFConstructorWithOpts<H extends HashXOF<H>, T extends Object>(hashCons: (opts?: T) => HashXOF<H>): {
    (msg: Input, opts?: T): Uint8Array;
    outputLen: number;
    blockLen: number;
    create(opts: T): HashXOF<H>;
};
/** Cryptographically secure PRNG. Uses internal OS-level `crypto.getRandomValues`. */
export declare function randomBytes(bytesLength?: number): Uint8Array;
export {};
//# sourceMappingURL=utils.d.ts.map