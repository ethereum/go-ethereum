import type { Address } from './address.js';
import type { PrefixedHexString } from './types.js';
/**
 * Verkle related constants and helper functions
 *
 * Experimental (do not use in production!)
 */
export interface VerkleCrypto {
    getTreeKey: (address: Uint8Array, treeIndex: Uint8Array, subIndex: number) => Uint8Array;
    getTreeKeyHash: (address: Uint8Array, treeIndexLE: Uint8Array) => Uint8Array;
    updateCommitment: (commitment: Uint8Array, commitmentIndex: number, oldScalarValue: Uint8Array, newScalarValue: Uint8Array) => Uint8Array;
    zeroCommitment: Uint8Array;
    verifyExecutionWitnessPreState: (prestateRoot: string, execution_witness_json: string) => boolean;
    hashCommitment: (commitment: Uint8Array) => Uint8Array;
    serializeCommitment: (commitment: Uint8Array) => Uint8Array;
}
/**
 * @dev Returns the 31-bytes verkle tree stem for a given address and tree index.
 * @dev Assumes that the verkle node width = 256
 * @param ffi The verkle ffi object from verkle-crypotography-wasm.
 * @param address The address to generate the tree key for.
 * @param treeIndex The index of the tree to generate the key for. Defaults to 0.
 * @return The 31-bytes verkle tree stem as a Uint8Array.
 */
export declare function getVerkleStem(ffi: VerkleCrypto, address: Address, treeIndex?: number | bigint): Uint8Array;
/**
 * Verifies that the executionWitness is valid for the given prestateRoot.
 * @param ffi The verkle ffi object from verkle-crypotography-wasm.
 * @param prestateRoot The prestateRoot matching the executionWitness.
 * @param executionWitness The verkle execution witness.
 * @returns {boolean} Whether or not the executionWitness belongs to the prestateRoot.
 */
export declare function verifyVerkleProof(ffi: VerkleCrypto, prestateRoot: Uint8Array, executionWitness: VerkleExecutionWitness): boolean;
export interface VerkleProof {
    commitmentsByPath: PrefixedHexString[];
    d: PrefixedHexString;
    depthExtensionPresent: PrefixedHexString;
    ipaProof: {
        cl: PrefixedHexString[];
        cr: PrefixedHexString[];
        finalEvaluation: PrefixedHexString;
    };
    otherStems: PrefixedHexString[];
}
export interface VerkleStateDiff {
    stem: PrefixedHexString;
    suffixDiffs: {
        currentValue: PrefixedHexString | null;
        newValue: PrefixedHexString | null;
        suffix: number | string;
    }[];
}
/**
 * Experimental, object format could eventual change.
 * An object that provides the state and proof necessary for verkle stateless execution
 * */
export interface VerkleExecutionWitness {
    /**
     * An array of state diffs.
     * Each item corresponding to state accesses or state modifications of the block.
     * In the current design, it also contains the resulting state of the block execution (post-state).
     */
    stateDiff: VerkleStateDiff[];
    /**
     * The verkle proof for the block.
     * Proves that the provided stateDiff belongs to the canonical verkle tree.
     */
    verkleProof: VerkleProof;
}
export declare enum VerkleLeafType {
    Version = 0,
    Balance = 1,
    Nonce = 2,
    CodeHash = 3,
    CodeSize = 4
}
export declare const VERKLE_VERSION_LEAF_KEY: Uint8Array;
export declare const VERKLE_BALANCE_LEAF_KEY: Uint8Array;
export declare const VERKLE_NONCE_LEAF_KEY: Uint8Array;
export declare const VERKLE_CODE_HASH_LEAF_KEY: Uint8Array;
export declare const VERKLE_CODE_SIZE_LEAF_KEY: Uint8Array;
export declare const VERKLE_HEADER_STORAGE_OFFSET = 64;
export declare const VERKLE_CODE_OFFSET = 128;
export declare const VERKLE_NODE_WIDTH = 256;
export declare const VERKLE_MAIN_STORAGE_OFFSET: bigint;
/**
 * @dev Returns the tree key for a given verkle tree stem, and sub index.
 * @dev Assumes that the verkle node width = 256
 * @param stem The 31-bytes verkle tree stem as a Uint8Array.
 * @param subIndex The sub index of the tree to generate the key for as a Uint8Array.
 * @return The tree key as a Uint8Array.
 */
export declare const getVerkleKey: (stem: Uint8Array, leaf: VerkleLeafType | Uint8Array) => Uint8Array;
export declare function getVerkleTreeIndexesForStorageSlot(storageKey: bigint): {
    treeIndex: bigint;
    subIndex: number;
};
export declare function getVerkleTreeIndicesForCodeChunk(chunkId: number): {
    treeIndex: number;
    subIndex: number;
};
export declare const getVerkleTreeKeyForCodeChunk: (address: Address, chunkId: number, verkleCrypto: VerkleCrypto) => Promise<Uint8Array>;
export declare const chunkifyCode: (code: Uint8Array) => never;
export declare const getVerkleTreeKeyForStorageSlot: (address: Address, storageKey: bigint, verkleCrypto: VerkleCrypto) => Promise<Uint8Array>;
//# sourceMappingURL=verkle.d.ts.map