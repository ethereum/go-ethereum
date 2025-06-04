import { type Input, Hash } from './utils.ts';
/**
 * Internal blake variable.
 * For BLAKE2b, the two extra permutations for rounds 10 and 11 are SIGMA[10..11] = SIGMA[0..1].
 */
export declare const SIGMA: Uint8Array;
/** Blake hash options. dkLen is output length. key is used in MAC mode. salt is used in KDF mode. */
export type BlakeOpts = {
    dkLen?: number;
    key?: Input;
    salt?: Input;
    personalization?: Input;
};
/** Class, from which others are subclassed. */
export declare abstract class BLAKE<T extends BLAKE<T>> extends Hash<T> {
    protected abstract compress(msg: Uint32Array, offset: number, isLast: boolean): void;
    protected abstract get(): number[];
    protected abstract set(...args: number[]): void;
    abstract destroy(): void;
    protected buffer: Uint8Array;
    protected buffer32: Uint32Array;
    protected length: number;
    protected pos: number;
    protected finished: boolean;
    protected destroyed: boolean;
    readonly blockLen: number;
    readonly outputLen: number;
    constructor(blockLen: number, outputLen: number, opts: BlakeOpts | undefined, keyLen: number, saltLen: number, persLen: number);
    update(data: Input): this;
    digestInto(out: Uint8Array): void;
    digest(): Uint8Array;
    _cloneInto(to?: T): T;
}
//# sourceMappingURL=_blake.d.ts.map