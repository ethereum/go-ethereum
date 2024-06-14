import { Hash, Input } from './utils.js';
export declare const Chi: (a: number, b: number, c: number) => number;
export declare const Maj: (a: number, b: number, c: number) => number;
/**
 * Merkle-Damgard hash construction base class.
 * Could be used to create MD5, RIPEMD, SHA1, SHA2.
 */
export declare abstract class HashMD<T extends HashMD<T>> extends Hash<T> {
    readonly blockLen: number;
    outputLen: number;
    readonly padOffset: number;
    readonly isLE: boolean;
    protected abstract process(buf: DataView, offset: number): void;
    protected abstract get(): number[];
    protected abstract set(...args: number[]): void;
    abstract destroy(): void;
    protected abstract roundClean(): void;
    protected buffer: Uint8Array;
    protected view: DataView;
    protected finished: boolean;
    protected length: number;
    protected pos: number;
    protected destroyed: boolean;
    constructor(blockLen: number, outputLen: number, padOffset: number, isLE: boolean);
    update(data: Input): this;
    digestInto(out: Uint8Array): void;
    digest(): Uint8Array;
    _cloneInto(to?: T): T;
}
//# sourceMappingURL=_md.d.ts.map