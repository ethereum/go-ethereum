/// <reference types="node" />

// No need to export this
type TypedArray =
    | Int8Array
    | Uint8Array
    | Uint8ClampedArray
    | Int16Array
    | Uint16Array
    | Int32Array
    | Uint32Array
    | Float32Array
    | Float64Array;
export function pbkdf2(
    password: string | Buffer | TypedArray | DataView,
    salt: string | Buffer | TypedArray | DataView,
    iterations: number,
    keylen: number,
    callback: (err: Error, derivedKey: Buffer) => void,
): void;
export function pbkdf2(
    password: string | Buffer | TypedArray | DataView,
    salt: string | Buffer | TypedArray | DataView,
    iterations: number,
    keylen: number,
    digest: string,
    callback: (err: Error, derivedKey: Buffer) => void,
): void;
export function pbkdf2Sync(
    password: string | Buffer | TypedArray | DataView,
    salt: string | Buffer | TypedArray | DataView,
    iterations: number,
    keylen: number,
    digest?: string,
): Buffer;

export {};
