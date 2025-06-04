# Installation
> `npm install --save @types/pbkdf2`

# Summary
This package contains type definitions for pbkdf2 (https://github.com/crypto-browserify/pbkdf2).

# Details
Files were exported from https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/pbkdf2.
## [index.d.ts](https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/pbkdf2/index.d.ts)
````ts
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

````

### Additional Details
 * Last updated: Tue, 07 Nov 2023 09:09:39 GMT
 * Dependencies: [@types/node](https://npmjs.com/package/@types/node)

# Credits
These definitions were written by [Timon Engelke](https://github.com/timonegk).
