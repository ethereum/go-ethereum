/**
 * Blake2b hash function. Focuses on 64-bit platforms, but in JS speed different from Blake2s is negligible.
 * @module
 */
import { BLAKE, type BlakeOpts } from './_blake.ts';
import { type CHashO } from './utils.ts';
export declare class BLAKE2b extends BLAKE<BLAKE2b> {
    private v0l;
    private v0h;
    private v1l;
    private v1h;
    private v2l;
    private v2h;
    private v3l;
    private v3h;
    private v4l;
    private v4h;
    private v5l;
    private v5h;
    private v6l;
    private v6h;
    private v7l;
    private v7h;
    constructor(opts?: BlakeOpts);
    protected get(): [
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number
    ];
    protected set(v0l: number, v0h: number, v1l: number, v1h: number, v2l: number, v2h: number, v3l: number, v3h: number, v4l: number, v4h: number, v5l: number, v5h: number, v6l: number, v6h: number, v7l: number, v7h: number): void;
    protected compress(msg: Uint32Array, offset: number, isLast: boolean): void;
    destroy(): void;
}
/**
 * Blake2b hash function. Focuses on 64-bit platforms, but in JS speed different from Blake2s is negligible.
 * @param msg - message that would be hashed
 * @param opts - dkLen output length, key for MAC mode, salt, personalization
 */
export declare const blake2b: CHashO;
//# sourceMappingURL=blake2b.d.ts.map