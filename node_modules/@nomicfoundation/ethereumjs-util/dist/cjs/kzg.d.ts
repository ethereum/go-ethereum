/**
 * Interface for an externally provided kzg library used when creating blob transactions
 */
export interface Kzg {
    loadTrustedSetup(filePath: string): void;
    blobToKzgCommitment(blob: Uint8Array): Uint8Array;
    computeBlobKzgProof(blob: Uint8Array, commitment: Uint8Array): Uint8Array;
    verifyKzgProof(polynomialKzg: Uint8Array, z: Uint8Array, y: Uint8Array, kzgProof: Uint8Array): boolean;
    verifyBlobKzgProofBatch(blobs: Uint8Array[], expectedKzgCommitments: Uint8Array[], kzgProofs: Uint8Array[]): boolean;
}
export declare let kzg: Kzg;
/**
 * @param kzgLib a KZG implementation (defaults to c-kzg)
 * @param trustedSetupPath the full path (e.g. "/home/linux/devnet4.txt") to a kzg trusted setup text file
 */
export declare function initKZG(kzgLib: Kzg, trustedSetupPath: string): void;
//# sourceMappingURL=kzg.d.ts.map