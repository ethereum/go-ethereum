"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getVerkleTreeKeyForStorageSlot = exports.chunkifyCode = exports.getVerkleTreeKeyForCodeChunk = exports.getVerkleTreeIndicesForCodeChunk = exports.getVerkleTreeIndexesForStorageSlot = exports.getVerkleKey = exports.VERKLE_MAIN_STORAGE_OFFSET = exports.VERKLE_NODE_WIDTH = exports.VERKLE_CODE_OFFSET = exports.VERKLE_HEADER_STORAGE_OFFSET = exports.VERKLE_CODE_SIZE_LEAF_KEY = exports.VERKLE_CODE_HASH_LEAF_KEY = exports.VERKLE_NONCE_LEAF_KEY = exports.VERKLE_BALANCE_LEAF_KEY = exports.VERKLE_VERSION_LEAF_KEY = exports.VerkleLeafType = exports.verifyVerkleProof = exports.getVerkleStem = void 0;
const bytes_js_1 = require("./bytes.js");
/**
 * @dev Returns the 31-bytes verkle tree stem for a given address and tree index.
 * @dev Assumes that the verkle node width = 256
 * @param ffi The verkle ffi object from verkle-crypotography-wasm.
 * @param address The address to generate the tree key for.
 * @param treeIndex The index of the tree to generate the key for. Defaults to 0.
 * @return The 31-bytes verkle tree stem as a Uint8Array.
 */
function getVerkleStem(ffi, address, treeIndex = 0) {
    const address32 = (0, bytes_js_1.setLengthLeft)(address.toBytes(), 32);
    let treeIndexBytes;
    if (typeof treeIndex === 'number') {
        treeIndexBytes = (0, bytes_js_1.setLengthRight)((0, bytes_js_1.int32ToBytes)(Number(treeIndex), true), 32);
    }
    else {
        treeIndexBytes = (0, bytes_js_1.setLengthRight)((0, bytes_js_1.bigIntToBytes)(BigInt(treeIndex), true).slice(0, 32), 32);
    }
    const treeStem = ffi.getTreeKey(address32, treeIndexBytes, 0).slice(0, 31);
    return treeStem;
}
exports.getVerkleStem = getVerkleStem;
/**
 * Verifies that the executionWitness is valid for the given prestateRoot.
 * @param ffi The verkle ffi object from verkle-crypotography-wasm.
 * @param prestateRoot The prestateRoot matching the executionWitness.
 * @param executionWitness The verkle execution witness.
 * @returns {boolean} Whether or not the executionWitness belongs to the prestateRoot.
 */
function verifyVerkleProof(ffi, prestateRoot, executionWitness) {
    return ffi.verifyExecutionWitnessPreState((0, bytes_js_1.bytesToHex)(prestateRoot), JSON.stringify(executionWitness));
}
exports.verifyVerkleProof = verifyVerkleProof;
var VerkleLeafType;
(function (VerkleLeafType) {
    VerkleLeafType[VerkleLeafType["Version"] = 0] = "Version";
    VerkleLeafType[VerkleLeafType["Balance"] = 1] = "Balance";
    VerkleLeafType[VerkleLeafType["Nonce"] = 2] = "Nonce";
    VerkleLeafType[VerkleLeafType["CodeHash"] = 3] = "CodeHash";
    VerkleLeafType[VerkleLeafType["CodeSize"] = 4] = "CodeSize";
})(VerkleLeafType = exports.VerkleLeafType || (exports.VerkleLeafType = {}));
exports.VERKLE_VERSION_LEAF_KEY = (0, bytes_js_1.intToBytes)(VerkleLeafType.Version);
exports.VERKLE_BALANCE_LEAF_KEY = (0, bytes_js_1.intToBytes)(VerkleLeafType.Balance);
exports.VERKLE_NONCE_LEAF_KEY = (0, bytes_js_1.intToBytes)(VerkleLeafType.Nonce);
exports.VERKLE_CODE_HASH_LEAF_KEY = (0, bytes_js_1.intToBytes)(VerkleLeafType.CodeHash);
exports.VERKLE_CODE_SIZE_LEAF_KEY = (0, bytes_js_1.intToBytes)(VerkleLeafType.CodeSize);
exports.VERKLE_HEADER_STORAGE_OFFSET = 64;
exports.VERKLE_CODE_OFFSET = 128;
exports.VERKLE_NODE_WIDTH = 256;
exports.VERKLE_MAIN_STORAGE_OFFSET = BigInt(256) ** BigInt(31);
/**
 * @dev Returns the tree key for a given verkle tree stem, and sub index.
 * @dev Assumes that the verkle node width = 256
 * @param stem The 31-bytes verkle tree stem as a Uint8Array.
 * @param subIndex The sub index of the tree to generate the key for as a Uint8Array.
 * @return The tree key as a Uint8Array.
 */
const getVerkleKey = (stem, leaf) => {
    switch (leaf) {
        case VerkleLeafType.Version:
            return (0, bytes_js_1.concatBytes)(stem, exports.VERKLE_VERSION_LEAF_KEY);
        case VerkleLeafType.Balance:
            return (0, bytes_js_1.concatBytes)(stem, exports.VERKLE_BALANCE_LEAF_KEY);
        case VerkleLeafType.Nonce:
            return (0, bytes_js_1.concatBytes)(stem, exports.VERKLE_NONCE_LEAF_KEY);
        case VerkleLeafType.CodeHash:
            return (0, bytes_js_1.concatBytes)(stem, exports.VERKLE_CODE_HASH_LEAF_KEY);
        case VerkleLeafType.CodeSize:
            return (0, bytes_js_1.concatBytes)(stem, exports.VERKLE_CODE_SIZE_LEAF_KEY);
        default:
            return (0, bytes_js_1.concatBytes)(stem, leaf);
    }
};
exports.getVerkleKey = getVerkleKey;
function getVerkleTreeIndexesForStorageSlot(storageKey) {
    let position;
    if (storageKey < exports.VERKLE_CODE_OFFSET - exports.VERKLE_HEADER_STORAGE_OFFSET) {
        position = BigInt(exports.VERKLE_HEADER_STORAGE_OFFSET) + storageKey;
    }
    else {
        position = exports.VERKLE_MAIN_STORAGE_OFFSET + storageKey;
    }
    const treeIndex = position / BigInt(exports.VERKLE_NODE_WIDTH);
    const subIndex = Number(position % BigInt(exports.VERKLE_NODE_WIDTH));
    return { treeIndex, subIndex };
}
exports.getVerkleTreeIndexesForStorageSlot = getVerkleTreeIndexesForStorageSlot;
function getVerkleTreeIndicesForCodeChunk(chunkId) {
    const treeIndex = Math.floor((exports.VERKLE_CODE_OFFSET + chunkId) / exports.VERKLE_NODE_WIDTH);
    const subIndex = (exports.VERKLE_CODE_OFFSET + chunkId) % exports.VERKLE_NODE_WIDTH;
    return { treeIndex, subIndex };
}
exports.getVerkleTreeIndicesForCodeChunk = getVerkleTreeIndicesForCodeChunk;
const getVerkleTreeKeyForCodeChunk = async (address, chunkId, verkleCrypto) => {
    const { treeIndex, subIndex } = getVerkleTreeIndicesForCodeChunk(chunkId);
    return (0, bytes_js_1.concatBytes)(getVerkleStem(verkleCrypto, address, treeIndex), (0, bytes_js_1.toBytes)(subIndex));
};
exports.getVerkleTreeKeyForCodeChunk = getVerkleTreeKeyForCodeChunk;
const chunkifyCode = (code) => {
    // Pad code to multiple of 31 bytes
    if (code.length % 31 !== 0) {
        const paddingLength = 31 - (code.length % 31);
        code = (0, bytes_js_1.setLengthRight)(code, code.length + paddingLength);
    }
    throw new Error('Not implemented');
};
exports.chunkifyCode = chunkifyCode;
const getVerkleTreeKeyForStorageSlot = async (address, storageKey, verkleCrypto) => {
    const { treeIndex, subIndex } = getVerkleTreeIndexesForStorageSlot(storageKey);
    return (0, bytes_js_1.concatBytes)(getVerkleStem(verkleCrypto, address, treeIndex), (0, bytes_js_1.toBytes)(subIndex));
};
exports.getVerkleTreeKeyForStorageSlot = getVerkleTreeKeyForStorageSlot;
//# sourceMappingURL=verkle.js.map