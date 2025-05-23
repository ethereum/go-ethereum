/**
 *
 * @param s byte sequence
 * @returns boolean indicating if input hex nibble sequence has terminator indicating leaf-node
 *          terminator is represented with 16 because a nibble ranges from 0 - 15(f)
 */
export declare const hasTerminator: (nibbles: Uint8Array) => boolean;
export declare const nibblesToBytes: (nibbles: Uint8Array, bytes: Uint8Array) => void;
export declare const nibblesToCompactBytes: (nibbles: Uint8Array) => Uint8Array;
export declare const bytesToNibbles: (str: Uint8Array) => Uint8Array;
export declare const compactBytesToNibbles: (compact: Uint8Array) => Uint8Array;
/**
 * A test helper to generates compact path for a subset of key bytes
 *
 * TODO: Commenting the code for now as this seems to be helper function
 * (from geth codebase )
 *
 */
//# sourceMappingURL=encoding.d.ts.map