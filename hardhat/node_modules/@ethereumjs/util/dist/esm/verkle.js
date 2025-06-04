import { bigIntToBytes, bytesToHex, concatBytes, int32ToBytes, intToBytes, setLengthLeft, setLengthRight, toBytes, } from './bytes.js';
/**
 * @dev Returns the 31-bytes verkle tree stem for a given address and tree index.
 * @dev Assumes that the verkle node width = 256
 * @param ffi The verkle ffi object from verkle-crypotography-wasm.
 * @param address The address to generate the tree key for.
 * @param treeIndex The index of the tree to generate the key for. Defaults to 0.
 * @return The 31-bytes verkle tree stem as a Uint8Array.
 */
export function getVerkleStem(ffi, address, treeIndex = 0) {
    const address32 = setLengthLeft(address.toBytes(), 32);
    let treeIndexBytes;
    if (typeof treeIndex === 'number') {
        treeIndexBytes = setLengthRight(int32ToBytes(Number(treeIndex), true), 32);
    }
    else {
        treeIndexBytes = setLengthRight(bigIntToBytes(BigInt(treeIndex), true).slice(0, 32), 32);
    }
    const treeStem = ffi.getTreeKey(address32, treeIndexBytes, 0).slice(0, 31);
    return treeStem;
}
/**
 * Verifies that the executionWitness is valid for the given prestateRoot.
 * @param ffi The verkle ffi object from verkle-crypotography-wasm.
 * @param prestateRoot The prestateRoot matching the executionWitness.
 * @param executionWitness The verkle execution witness.
 * @returns {boolean} Whether or not the executionWitness belongs to the prestateRoot.
 */
export function verifyVerkleProof(ffi, prestateRoot, executionWitness) {
    return ffi.verifyExecutionWitnessPreState(bytesToHex(prestateRoot), JSON.stringify(executionWitness));
}
export var VerkleLeafType;
(function (VerkleLeafType) {
    VerkleLeafType[VerkleLeafType["Version"] = 0] = "Version";
    VerkleLeafType[VerkleLeafType["Balance"] = 1] = "Balance";
    VerkleLeafType[VerkleLeafType["Nonce"] = 2] = "Nonce";
    VerkleLeafType[VerkleLeafType["CodeHash"] = 3] = "CodeHash";
    VerkleLeafType[VerkleLeafType["CodeSize"] = 4] = "CodeSize";
})(VerkleLeafType || (VerkleLeafType = {}));
export const VERKLE_VERSION_LEAF_KEY = intToBytes(VerkleLeafType.Version);
export const VERKLE_BALANCE_LEAF_KEY = intToBytes(VerkleLeafType.Balance);
export const VERKLE_NONCE_LEAF_KEY = intToBytes(VerkleLeafType.Nonce);
export const VERKLE_CODE_HASH_LEAF_KEY = intToBytes(VerkleLeafType.CodeHash);
export const VERKLE_CODE_SIZE_LEAF_KEY = intToBytes(VerkleLeafType.CodeSize);
export const VERKLE_HEADER_STORAGE_OFFSET = 64;
export const VERKLE_CODE_OFFSET = 128;
export const VERKLE_NODE_WIDTH = 256;
export const VERKLE_MAIN_STORAGE_OFFSET = BigInt(256) ** BigInt(31);
/**
 * @dev Returns the tree key for a given verkle tree stem, and sub index.
 * @dev Assumes that the verkle node width = 256
 * @param stem The 31-bytes verkle tree stem as a Uint8Array.
 * @param subIndex The sub index of the tree to generate the key for as a Uint8Array.
 * @return The tree key as a Uint8Array.
 */
export const getVerkleKey = (stem, leaf) => {
    switch (leaf) {
        case VerkleLeafType.Version:
            return concatBytes(stem, VERKLE_VERSION_LEAF_KEY);
        case VerkleLeafType.Balance:
            return concatBytes(stem, VERKLE_BALANCE_LEAF_KEY);
        case VerkleLeafType.Nonce:
            return concatBytes(stem, VERKLE_NONCE_LEAF_KEY);
        case VerkleLeafType.CodeHash:
            return concatBytes(stem, VERKLE_CODE_HASH_LEAF_KEY);
        case VerkleLeafType.CodeSize:
            return concatBytes(stem, VERKLE_CODE_SIZE_LEAF_KEY);
        default:
            return concatBytes(stem, leaf);
    }
};
export function getVerkleTreeIndexesForStorageSlot(storageKey) {
    let position;
    if (storageKey < VERKLE_CODE_OFFSET - VERKLE_HEADER_STORAGE_OFFSET) {
        position = BigInt(VERKLE_HEADER_STORAGE_OFFSET) + storageKey;
    }
    else {
        position = VERKLE_MAIN_STORAGE_OFFSET + storageKey;
    }
    const treeIndex = position / BigInt(VERKLE_NODE_WIDTH);
    const subIndex = Number(position % BigInt(VERKLE_NODE_WIDTH));
    return { treeIndex, subIndex };
}
export function getVerkleTreeIndicesForCodeChunk(chunkId) {
    const treeIndex = Math.floor((VERKLE_CODE_OFFSET + chunkId) / VERKLE_NODE_WIDTH);
    const subIndex = (VERKLE_CODE_OFFSET + chunkId) % VERKLE_NODE_WIDTH;
    return { treeIndex, subIndex };
}
export const getVerkleTreeKeyForCodeChunk = async (address, chunkId, verkleCrypto) => {
    const { treeIndex, subIndex } = getVerkleTreeIndicesForCodeChunk(chunkId);
    return concatBytes(getVerkleStem(verkleCrypto, address, treeIndex), toBytes(subIndex));
};
export const chunkifyCode = (code) => {
    // Pad code to multiple of 31 bytes
    if (code.length % 31 !== 0) {
        const paddingLength = 31 - (code.length % 31);
        code = setLengthRight(code, code.length + paddingLength);
    }
    throw new Error('Not implemented');
};
export const getVerkleTreeKeyForStorageSlot = async (address, storageKey, verkleCrypto) => {
    const { treeIndex, subIndex } = getVerkleTreeIndexesForStorageSlot(storageKey);
    return concatBytes(getVerkleStem(verkleCrypto, address, treeIndex), toBytes(subIndex));
};
//# sourceMappingURL=verkle.js.map