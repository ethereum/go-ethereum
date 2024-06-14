declare const assertBool: typeof import("@noble/hashes/_assert").bool;
declare const assertBytes: typeof import("@noble/hashes/_assert").bytes;
export { assertBool, assertBytes };
export { bytesToHex, bytesToHex as toHex, concatBytes, createView, utf8ToBytes } from "@noble/hashes/utils";
export declare function bytesToUtf8(data: Uint8Array): string;
export declare function hexToBytes(data: string): Uint8Array;
export declare function equalsBytes(a: Uint8Array, b: Uint8Array): boolean;
export declare function wrapHash(hash: (msg: Uint8Array) => Uint8Array): (msg: Uint8Array) => Uint8Array;
export declare const crypto: {
    node?: any;
    web?: Crypto;
};
