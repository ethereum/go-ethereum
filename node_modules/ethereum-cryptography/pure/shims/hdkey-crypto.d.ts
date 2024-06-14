/// <reference types="node" />
export declare const createHmac: any;
export declare const randomBytes: any;
declare class Hash {
    private readonly hashFunction;
    private buffers;
    constructor(hashFunction: (msg: Buffer) => Buffer);
    update(buffer: Buffer): this;
    digest(param: any): Buffer;
}
export declare const createHash: (name: string) => Hash;
export {};
//# sourceMappingURL=hdkey-crypto.d.ts.map