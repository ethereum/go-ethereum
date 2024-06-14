export declare const getBlobs: (input: string) => Uint8Array[];
export declare const blobsToCommitments: (blobs: Uint8Array[]) => Uint8Array[];
export declare const blobsToProofs: (blobs: Uint8Array[], commitments: Uint8Array[]) => Uint8Array[];
/**
 * Converts a vector commitment for a given data blob to its versioned hash.  For 4844, this version
 * number will be 0x01 for KZG vector commitments but could be different if future vector commitment
 * types are introduced
 * @param commitment a vector commitment to a blob
 * @param blobCommitmentVersion the version number corresponding to the type of vector commitment
 * @returns a versioned hash corresponding to a given blob vector commitment
 */
export declare const computeVersionedHash: (commitment: Uint8Array, blobCommitmentVersion: number) => Uint8Array;
/**
 * Generate an array of versioned hashes from corresponding kzg commitments
 * @param commitments array of kzg commitments
 * @returns array of versioned hashes
 * Note: assumes KZG commitments (version 1 version hashes)
 */
export declare const commitmentsToVersionedHashes: (commitments: Uint8Array[]) => Uint8Array[];
//# sourceMappingURL=blobs.d.ts.map