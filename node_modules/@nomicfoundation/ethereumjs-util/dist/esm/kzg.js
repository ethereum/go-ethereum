function kzgNotLoaded() {
    throw Error('kzg library not loaded');
}
// eslint-disable-next-line import/no-mutable-exports
export let kzg = {
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
export function initKZG(kzgLib, trustedSetupPath) {
    kzg = kzgLib;
    kzg.loadTrustedSetup(trustedSetupPath);
}
//# sourceMappingURL=kzg.js.map