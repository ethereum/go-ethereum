/**
 * Interface for an externally provided kzg library used when creating blob transactions
 */
export interface Kzg {
  loadTrustedSetup(filePath: string): void
  blobToKzgCommitment(blob: Uint8Array): Uint8Array
  computeBlobKzgProof(blob: Uint8Array, commitment: Uint8Array): Uint8Array
  verifyKzgProof(
    polynomialKzg: Uint8Array,
    z: Uint8Array,
    y: Uint8Array,
    kzgProof: Uint8Array
  ): boolean
  verifyBlobKzgProofBatch(
    blobs: Uint8Array[],
    expectedKzgCommitments: Uint8Array[],
    kzgProofs: Uint8Array[]
  ): boolean
}

function kzgNotLoaded(): never {
  throw Error('kzg library not loaded')
}

// eslint-disable-next-line import/no-mutable-exports
export let kzg: Kzg = {
  loadTrustedSetup: kzgNotLoaded,
  blobToKzgCommitment: kzgNotLoaded,
  computeBlobKzgProof: kzgNotLoaded,
  verifyKzgProof: kzgNotLoaded,
  verifyBlobKzgProofBatch: kzgNotLoaded,
}

/**
 * @param kzgLib a KZG implementation (defaults to c-kzg)
 * @param trustedSetupPath the full path (e.g. "/home/linux/devnet4.txt") to a kzg trusted setup text file
 */
export function initKZG(kzgLib: Kzg, trustedSetupPath: string) {
  kzg = kzgLib
  kzg.loadTrustedSetup(trustedSetupPath)
}
