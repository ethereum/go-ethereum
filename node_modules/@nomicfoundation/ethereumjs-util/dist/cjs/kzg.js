"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.initKZG = exports.kzg = void 0;
function kzgNotLoaded() {
    throw Error('kzg library not loaded');
}
// eslint-disable-next-line import/no-mutable-exports
exports.kzg = {
    loadTrustedSetup: kzgNotLoaded,
    blobToKzgCommitment: kzgNotLoaded,
    computeBlobKzgProof: kzgNotLoaded,
    verifyKzgProof: kzgNotLoaded,
    verifyBlobKzgProofBatch: kzgNotLoaded,
};
/**
 * @param kzgLib a KZG implementation (defaults to c-kzg)
 * @param trustedSetupPath the full path (e.g. "/home/linux/devnet4.txt") to a kzg trusted setup text file
 */
function initKZG(kzgLib, trustedSetupPath) {
    exports.kzg = kzgLib;
    exports.kzg.loadTrustedSetup(trustedSetupPath);
}
exports.initKZG = initKZG;
//# sourceMappingURL=kzg.js.map