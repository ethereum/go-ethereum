"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.commitmentsToVersionedHashes = exports.computeVersionedHash = exports.blobsToProofs = exports.blobsToCommitments = exports.getBlobs = void 0;
const sha256_js_1 = require("ethereum-cryptography/sha256.js");
const bytes_js_1 = require("./bytes.js");
const kzg_js_1 = require("./kzg.js");
/**
 * These utilities for constructing blobs are borrowed from https://github.com/Inphi/eip4844-interop.git
 */
const BYTES_PER_FIELD_ELEMENT = 32;
const FIELD_ELEMENTS_PER_BLOB = 4096;
const USEFUL_BYTES_PER_BLOB = 32 * FIELD_ELEMENTS_PER_BLOB;
const MAX_BLOBS_PER_TX = 2;
const MAX_USEFUL_BYTES_PER_TX = USEFUL_BYTES_PER_BLOB * MAX_BLOBS_PER_TX - 1;
const BLOB_SIZE = BYTES_PER_FIELD_ELEMENT * FIELD_ELEMENTS_PER_BLOB;
function get_padded(data, blobs_len) {
    const pdata = new Uint8Array(blobs_len * USEFUL_BYTES_PER_BLOB).fill(0);
    pdata.set(data);
    pdata[data.byteLength] = 0x80;
    return pdata;
}
function get_blob(data) {
    const blob = new Uint8Array(BLOB_SIZE);
    for (let i = 0; i < FIELD_ELEMENTS_PER_BLOB; i++) {
        const chunk = new Uint8Array(32);
        chunk.set(data.subarray(i * 31, (i + 1) * 31), 0);
        blob.set(chunk, i * 32);
    }
    return blob;
}
const getBlobs = (input) => {
    const data = (0, bytes_js_1.utf8ToBytes)(input);
    const len = data.byteLength;
    if (len === 0) {
        throw Error('invalid blob data');
    }
    if (len > MAX_USEFUL_BYTES_PER_TX) {
        throw Error('blob data is too large');
    }
    const blobs_len = Math.ceil(len / USEFUL_BYTES_PER_BLOB);
    const pdata = get_padded(data, blobs_len);
    const blobs = [];
    for (let i = 0; i < blobs_len; i++) {
        const chunk = pdata.subarray(i * USEFUL_BYTES_PER_BLOB, (i + 1) * USEFUL_BYTES_PER_BLOB);
        const blob = get_blob(chunk);
        blobs.push(blob);
    }
    return blobs;
};
exports.getBlobs = getBlobs;
const blobsToCommitments = (blobs) => {
    const commitments = [];
    for (const blob of blobs) {
        commitments.push(kzg_js_1.kzg.blobToKzgCommitment(blob));
    }
    return commitments;
};
exports.blobsToCommitments = blobsToCommitments;
const blobsToProofs = (blobs, commitments) => {
    const proofs = blobs.map((blob, ctx) => kzg_js_1.kzg.computeBlobKzgProof(blob, commitments[ctx]));
    return proofs;
};
exports.blobsToProofs = blobsToProofs;
/**
 * Converts a vector commitment for a given data blob to its versioned hash.  For 4844, this version
 * number will be 0x01 for KZG vector commitments but could be different if future vector commitment
 * types are introduced
 * @param commitment a vector commitment to a blob
 * @param blobCommitmentVersion the version number corresponding to the type of vector commitment
 * @returns a versioned hash corresponding to a given blob vector commitment
 */
const computeVersionedHash = (commitment, blobCommitmentVersion) => {
    const computedVersionedHash = new Uint8Array(32);
    computedVersionedHash.set([blobCommitmentVersion], 0);
    computedVersionedHash.set((0, sha256_js_1.sha256)(Buffer.from(commitment)).subarray(1), 1);
    return computedVersionedHash;
};
exports.computeVersionedHash = computeVersionedHash;
/**
 * Generate an array of versioned hashes from corresponding kzg commitments
 * @param commitments array of kzg commitments
 * @returns array of versioned hashes
 * Note: assumes KZG commitments (version 1 version hashes)
 */
const commitmentsToVersionedHashes = (commitments) => {
    const hashes = [];
    for (const commitment of commitments) {
        hashes.push((0, exports.computeVersionedHash)(commitment, 0x01));
    }
    return hashes;
};
exports.commitmentsToVersionedHashes = commitmentsToVersionedHashes;
//# sourceMappingURL=blobs.js.map