export declare type Input = string | number | bigint | Uint8Array | Array<Input> | null | undefined;
export declare type NestedUint8Array = Array<Uint8Array | NestedUint8Array>;
export interface Decoded {
    data: Uint8Array | NestedUint8Array;
    remainder: Uint8Array;
}
/**
 * RLP Encoding based on https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/
 * This function takes in data, converts it to Uint8Array if not,
 * and adds a length for recursion.
 * @param input Will be converted to Uint8Array
 * @returns Uint8Array of encoded data
 **/
export declare function encode(input: Input): Uint8Array;
/**
 * RLP Decoding based on https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/
 * @param input Will be converted to Uint8Array
 * @param stream Is the input a stream (false by default)
 * @returns decoded Array of Uint8Arrays containing the original message
 **/
export declare function decode(input: Input, stream?: false): Uint8Array | NestedUint8Array;
export declare function decode(input: Input, stream?: true): Decoded;
declare function bytesToHex(uint8a: Uint8Array): string;
declare function hexToBytes(hex: string): Uint8Array;
/** Concatenates two Uint8Arrays into one. */
declare function concatBytes(...arrays: Uint8Array[]): Uint8Array;
declare function utf8ToBytes(utf: string): Uint8Array;
export declare const utils: {
    bytesToHex: typeof bytesToHex;
    concatBytes: typeof concatBytes;
    hexToBytes: typeof hexToBytes;
    utf8ToBytes: typeof utf8ToBytes;
};
export declare const RLP: {
    encode: typeof encode;
    decode: typeof decode;
};
export {};
//# sourceMappingURL=index.d.ts.map