import { HashMD } from './_md.js';
declare class SHA1 extends HashMD<SHA1> {
    private A;
    private B;
    private C;
    private D;
    private E;
    constructor();
    protected get(): [number, number, number, number, number];
    protected set(A: number, B: number, C: number, D: number, E: number): void;
    protected process(view: DataView, offset: number): void;
    protected roundClean(): void;
    destroy(): void;
}
export declare const sha1: {
    (msg: import("./utils.js").Input): Uint8Array;
    outputLen: number;
    blockLen: number;
    create(): import("./utils.js").Hash<SHA1>;
};
export {};
//# sourceMappingURL=sha1.d.ts.map