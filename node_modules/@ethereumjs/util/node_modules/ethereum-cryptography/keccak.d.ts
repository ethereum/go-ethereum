import { Keccak } from "@noble/hashes/sha3";
import { Hash } from "@noble/hashes/utils";
interface K256 {
    (data: Uint8Array): Uint8Array;
    create(): Hash<Keccak>;
}
export declare const keccak224: (msg: Uint8Array) => Uint8Array;
export declare const keccak256: K256;
export declare const keccak384: (msg: Uint8Array) => Uint8Array;
export declare const keccak512: (msg: Uint8Array) => Uint8Array;
export {};
