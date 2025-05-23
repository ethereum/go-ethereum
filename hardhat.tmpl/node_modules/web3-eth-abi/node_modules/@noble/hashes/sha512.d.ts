import { HashMD } from './_md.js';
export declare class SHA512 extends HashMD<SHA512> {
    Ah: number;
    Al: number;
    Bh: number;
    Bl: number;
    Ch: number;
    Cl: number;
    Dh: number;
    Dl: number;
    Eh: number;
    El: number;
    Fh: number;
    Fl: number;
    Gh: number;
    Gl: number;
    Hh: number;
    Hl: number;
    constructor();
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
    protected set(Ah: number, Al: number, Bh: number, Bl: number, Ch: number, Cl: number, Dh: number, Dl: number, Eh: number, El: number, Fh: number, Fl: number, Gh: number, Gl: number, Hh: number, Hl: number): void;
    protected process(view: DataView, offset: number): void;
    protected roundClean(): void;
    destroy(): void;
}
export declare const sha512: {
    (msg: import("./utils.js").Input): Uint8Array;
    outputLen: number;
    blockLen: number;
    create(): import("./utils.js").Hash<SHA512>;
};
export declare const sha512_224: {
    (msg: import("./utils.js").Input): Uint8Array;
    outputLen: number;
    blockLen: number;
    create(): import("./utils.js").Hash<SHA512>;
};
export declare const sha512_256: {
    (msg: import("./utils.js").Input): Uint8Array;
    outputLen: number;
    blockLen: number;
    create(): import("./utils.js").Hash<SHA512>;
};
export declare const sha384: {
    (msg: import("./utils.js").Input): Uint8Array;
    outputLen: number;
    blockLen: number;
    create(): import("./utils.js").Hash<SHA512>;
};
//# sourceMappingURL=sha512.d.ts.map