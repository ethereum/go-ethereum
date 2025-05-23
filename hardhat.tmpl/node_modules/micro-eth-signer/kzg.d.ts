type Scalar = string | bigint;
type Blob = string | string[] | bigint[];
export type SetupData = {
    g1_lagrange: string[];
    g2_monomial: string[];
};
/**
 * KZG from [EIP-4844](https://eips.ethereum.org/EIPS/eip-4844).
 * @example
 * const kzg = new KZG(trustedSetupData);
 */
export declare class KZG {
    private readonly POLY_NUM;
    private readonly G1LB;
    private readonly G2M;
    private readonly ROOTS_OF_UNITY;
    private readonly FIAT_SHAMIR_PROTOCOL_DOMAIN;
    private readonly RANDOM_CHALLENGE_KZG_BATCH_DOMAIN;
    private readonly POLY_NUM_BYTES;
    constructor(setup: SetupData & {
        encoding?: 'fast_v1';
    });
    private parseG1;
    private parseG1Unchecked;
    private parseG2;
    private parseG2Unchecked;
    private parseBlob;
    private invSafe;
    private G1msm;
    private computeChallenge;
    private evalPoly;
    computeProof(blob: Blob, z: bigint | string): [string, string];
    verifyProof(commitment: string, z: Scalar, y: Scalar, proof: string): boolean;
    private verifyProofBatch;
    blobToKzgCommitment(blob: Blob): string;
    computeBlobProof(blob: Blob, commitment: string): string;
    verifyBlobProof(blob: Blob, commitment: string, proof: string): boolean;
    verifyBlobProofBatch(blobs: string[], commitments: string[], proofs: string[]): boolean;
}
export {};
//# sourceMappingURL=kzg.d.ts.map