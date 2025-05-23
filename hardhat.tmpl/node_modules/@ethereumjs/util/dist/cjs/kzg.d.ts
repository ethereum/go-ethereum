/**
 * Interface for an externally provided kzg library used when creating blob transactions
 */
export interface Kzg {
    loadTrustedSetup(trustedSetup?: {
        g1: string;
        g2: string;
        n1: number;
        n2: number;
    }): void;
    blobToKzgCommitment(blob: Uint8Array): Uint8Array;
    computeBlobKzgProof(blob: Uint8Array, commitment: Uint8Array): Uint8Array;
    verifyKzgProof(polynomialKzg: Uint8Array, z: Uint8Array, y: Uint8Array, kzgProof: Uint8Array): boolean;
    verifyBlobKzgProofBatch(blobs: Uint8Array[], expectedKzgCommitments: Uint8Array[], kzgProofs: Uint8Array[]): boolean;
}
/**
 * @deprecated This initialization method is deprecated since trusted setup loading is done directly in the reference KZG library
 * initialization or should othewise be assured independently before KZG libary usage.
 *
 * @param kzgLib a KZG implementation (defaults to c-kzg)
 * @param a dictionary of trusted setup options
 */
export declare function initKZG(kzg: Kzg, _trustedSetupPath?: string): void;
//# sourceMappingURL=kzg.d.ts.map