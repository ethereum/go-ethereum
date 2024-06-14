/*! noble-hashes - MIT License (c) 2022 Paul Miller (paulmillr.com) */
export declare type TypedArray = Int8Array | Uint8ClampedArray | Uint8Array | Uint16Array | Int16Array | Uint32Array | Int32Array;
export declare const u8: (arr: TypedArray) => Uint8Array;
export declare const u32: (arr: TypedArray) => Uint32Array;
export declare const createView: (arr: TypedArray) => DataView;
export declare const rotr: (word: number, shift: number) => number;
export declare const isLE: boolean;
/**
 * @example bytesToHex(Uint8Array.from([0xde, 0xad, 0xbe, 0xef]))
 */
export declare function bytesToHex(uint8a: Uint8Array): string;
/**
 * @example hexToBytes('deadbeef')
 */
export declare function hexToBytes(hex: string): Uint8Array;
export declare const nextTick: () => Promise<void>;
export declare function asyncLoop(iters: number, tick: number, cb: (i: number) => void): Promise<void>;
export declare function utf8ToBytes(str: string): Uint8Array;
export declare type Input = Uint8Array | string;
export declare function toBytes(data: Input): Uint8Array;
/**
 * Concats Uint8Array-s into one; like `Buffer.concat([buf1, buf2])`
 * @example concatBytes(buf1, buf2)
 */
export declare function concatBytes(...arrays: Uint8Array[]): Uint8Array;
export declare abstract class Hash<T extends Hash<T>> {
    abstract blockLen: number;
    abstract outputLen: number;
    abstract update(buf: Input): this;
    abstract digestInto(buf: Uint8Array): void;
    abstract digest(): Uint8Array;
    abstract destroy(): void;
    abstract _cloneInto(to?: T): T;
    clone(): T;
}
/**
 * XOF: streaming API to read digest in chunks.
 * Same as 'squeeze' in keccak/k12 and 'seek' in blake3, but more generic name.
 * When hash used in XOF mode it is up to user to call '.destroy' afterwards, since we cannot destroy state, next call can require more bytes.
 */
export declare type HashXOF<T extends Hash<T>> = Hash<T> & {
    xof(bytes: number): Uint8Array;
    xofInto(buf: Uint8Array): Uint8Array;
};
declare type EmptyObj = {};
export declare function checkOpts<T1 extends EmptyObj, T2 extends EmptyObj>(defaults: T1, opts?: T2): T1 & T2;
export declare type CHash = ReturnType<typeof wrapConstructor>;
export declare function wrapConstructor<T extends Hash<T>>(hashConstructor: () => Hash<T>): {
    (message: Input): Uint8Array;
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
/**
 * Secure PRNG
 */
export declare function randomBytes(bytesLength?: number): Uint8Array;
export {};
