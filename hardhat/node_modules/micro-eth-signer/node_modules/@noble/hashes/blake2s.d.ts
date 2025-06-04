/**
 * Blake2s hash function. Focuses on 8-bit to 32-bit platforms. blake2b for 64-bit, but in JS it is slower.
 * @module
 */
import { BLAKE, type BlakeOpts } from './_blake.ts';
import { type CHashO } from './utils.ts';
/**
 * Initial state: same as SHA256. First 32 bits of the fractional parts of the square roots
 * of the first 8 primes 2..19.
 */
export declare const B2S_IV: Uint32Array;
type Num4 = {
    a: number;
    b: number;
    c: number;
    d: number;
};
type Num16 = {
    v0: number;
    v1: number;
    v2: number;
    v3: number;
    v4: number;
    v5: number;
    v6: number;
    v7: number;
    v8: number;
    v9: number;
    v10: number;
    v11: number;
    v12: number;
    v13: number;
    v14: number;
    v15: number;
};
export declare function G1s(a: number, b: number, c: number, d: number, x: number): Num4;
export declare function G2s(a: number, b: number, c: number, d: number, x: number): Num4;
export declare function compress(s: Uint8Array, offset: number, msg: Uint32Array, rounds: number, v0: number, v1: number, v2: number, v3: number, v4: number, v5: number, v6: number, v7: number, v8: number, v9: number, v10: number, v11: number, v12: number, v13: number, v14: number, v15: number): Num16;
export declare class BLAKE2s extends BLAKE<BLAKE2s> {
    private v0;
    private v1;
    private v2;
    private v3;
    private v4;
    private v5;
    private v6;
    private v7;
    constructor(opts?: BlakeOpts);
    protected get(): [number, number, number, number, number, number, number, number];
    protected set(v0: number, v1: number, v2: number, v3: number, v4: number, v5: number, v6: number, v7: number): void;
    protected compress(msg: Uint32Array, offset: number, isLast: boolean): void;
    destroy(): void;
}
/**
 * Blake2s hash function. Focuses on 8-bit to 32-bit platforms. blake2b for 64-bit, but in JS it is slower.
 * @param msg - message that would be hashed
 * @param opts - dkLen output length, key for MAC mode, salt, personalization
 */
export declare const blake2s: CHashO;
export {};
//# sourceMappingURL=blake2s.d.ts.map