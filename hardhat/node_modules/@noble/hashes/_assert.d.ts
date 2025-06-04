export declare function number(n: number): void;
export declare function bool(b: boolean): void;
export declare function bytes(b: Uint8Array | undefined, ...lengths: number[]): void;
declare type Hash = {
    (data: Uint8Array): Uint8Array;
    blockLen: number;
    outputLen: number;
    create: any;
};
export declare function hash(hash: Hash): void;
export declare function exists(instance: any, checkFinished?: boolean): void;
export declare function output(out: any, instance: any): void;
declare const assert: {
    number: typeof number;
    bool: typeof bool;
    bytes: typeof bytes;
    hash: typeof hash;
    exists: typeof exists;
    output: typeof output;
};
export default assert;
