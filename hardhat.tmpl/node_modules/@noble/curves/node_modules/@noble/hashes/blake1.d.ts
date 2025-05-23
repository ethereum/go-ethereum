import { Hash, type Input } from './utils.ts';
/** Blake1 options. Basically just "salt" */
export type BlakeOpts = {
    salt?: Input;
};
declare abstract class Blake1<T extends Blake1<T>> extends Hash<T> {
    protected finished: boolean;
    protected length: number;
    protected pos: number;
    protected destroyed: boolean;
    protected buffer: Uint8Array;
    protected view: DataView;
    protected salt: Uint32Array;
    abstract compress(view: DataView, offset: number, withLength?: boolean): void;
    protected abstract get(): number[];
    protected abstract set(...args: number[]): void;
    readonly blockLen: number;
    readonly outputLen: number;
    private lengthFlag;
    private counterLen;
    protected constants: Uint32Array;
    constructor(blockLen: number, outputLen: number, lengthFlag: number, counterLen: number, saltLen: number, constants: Uint32Array, opts?: BlakeOpts);
    update(data: Input): this;
    destroy(): void;
    _cloneInto(to?: T): T;
    digestInto(out: Uint8Array): void;
    digest(): Uint8Array;
}
declare class Blake1_32 extends Blake1<Blake1_32> {
    private v0;
    private v1;
    private v2;
    private v3;
    private v4;
    private v5;
    private v6;
    private v7;
    constructor(outputLen: number, IV: Uint32Array, lengthFlag: number, opts?: BlakeOpts);
    protected get(): [number, number, number, number, number, number, number, number];
    protected set(v0: number, v1: number, v2: number, v3: number, v4: number, v5: number, v6: number, v7: number): void;
    destroy(): void;
    compress(view: DataView, offset: number, withLength?: boolean): void;
}
declare class Blake1_64 extends Blake1<Blake1_64> {
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
    constructor(outputLen: number, IV: Uint32Array, lengthFlag: number, opts?: BlakeOpts);
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
    destroy(): void;
    compress(view: DataView, offset: number, withLength?: boolean): void;
}
export declare class Blake224 extends Blake1_32 {
    constructor(opts?: BlakeOpts);
}
export declare class Blake256 extends Blake1_32 {
    constructor(opts?: BlakeOpts);
}
export declare class Blake512 extends Blake1_64 {
    constructor(opts?: BlakeOpts);
}
export declare class Blake384 extends Blake1_64 {
    constructor(opts?: BlakeOpts);
}
/** blake1-224 hash function */
export declare const blake224: {
    (msg: Input, opts?: BlakeOpts | undefined): Uint8Array;
    outputLen: number;
    blockLen: number;
    create(opts: BlakeOpts): Hash<Blake224>;
};
/** blake1-256 hash function */
export declare const blake256: {
    (msg: Input, opts?: BlakeOpts | undefined): Uint8Array;
    outputLen: number;
    blockLen: number;
    create(opts: BlakeOpts): Hash<Blake256>;
};
/** blake1-384 hash function */
export declare const blake384: {
    (msg: Input, opts?: BlakeOpts | undefined): Uint8Array;
    outputLen: number;
    blockLen: number;
    create(opts: BlakeOpts): Hash<Blake512>;
};
/** blake1-512 hash function */
export declare const blake512: {
    (msg: Input, opts?: BlakeOpts | undefined): Uint8Array;
    outputLen: number;
    blockLen: number;
    create(opts: BlakeOpts): Hash<Blake512>;
};
export {};
//# sourceMappingURL=blake1.d.ts.map