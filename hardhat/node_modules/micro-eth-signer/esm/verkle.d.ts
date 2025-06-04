export type Scalar = Uint8Array;
export type Commitment = Uint8Array;
export type ProverInput = {
    serializedCommitment: Uint8Array;
    vector: Uint8Array[];
    indices: number[];
};
export type VerifierInput = {
    serializedCommitment: Uint8Array;
    indexValuePairs: {
        index: number;
        value: Uint8Array;
    }[];
};
export declare const hashCommitment: (commitment: Uint8Array) => Uint8Array;
export declare const commitToScalars: (vector: Uint8Array[]) => Uint8Array;
export declare const hashCommitments: (commitments: Uint8Array[]) => Uint8Array[];
export declare const getTreeKeyHash: (address: Uint8Array, treeIndexLE: Uint8Array) => Uint8Array;
export declare const getTreeKey: (address: Uint8Array, treeIndex: Uint8Array, subIndex: number) => Uint8Array;
export declare const updateCommitment: (commitment: Uint8Array, commitmentIndex: number, oldScalarValue: Uint8Array, newScalarValue: Uint8Array) => Commitment;
export declare const zeroCommitment: Uint8Array;
export declare const serializeCommitment: (commitment: Uint8Array) => Uint8Array;
export declare const createProof: (proverInputs: ProverInput[]) => Uint8Array;
export declare const verifyProof: (proofBytes: Uint8Array, verifierInputs: VerifierInput[]) => boolean;
export declare function verifyExecutionWitnessPreState(rootHex: string, executionWitnessJson: string): boolean;
export declare const __tests: any;
//# sourceMappingURL=verkle.d.ts.map