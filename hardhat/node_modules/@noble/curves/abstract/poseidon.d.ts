/**
 * Implements [Poseidon](https://www.poseidon-hash.info) ZK-friendly hash.
 *
 * There are many poseidon variants with different constants.
 * We don't provide them: you should construct them manually.
 * Check out [micro-starknet](https://github.com/paulmillr/micro-starknet) package for a proper example.
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
import { type IField } from './modular.ts';
export type PoseidonOpts = {
    Fp: IField<bigint>;
    t: number;
    roundsFull: number;
    roundsPartial: number;
    sboxPower?: number;
    reversePartialPowIdx?: boolean;
    mds: bigint[][];
    roundConstants: bigint[][];
};
export declare function validateOpts(opts: PoseidonOpts): Readonly<{
    rounds: number;
    sboxFn: (n: bigint) => bigint;
    roundConstants: bigint[][];
    mds: bigint[][];
    Fp: IField<bigint>;
    t: number;
    roundsFull: number;
    roundsPartial: number;
    sboxPower?: number;
    reversePartialPowIdx?: boolean;
}>;
export declare function splitConstants(rc: bigint[], t: number): bigint[][];
/** Poseidon NTT-friendly hash. */
export declare function poseidon(opts: PoseidonOpts): {
    (values: bigint[]): bigint[];
    roundConstants: bigint[][];
};
//# sourceMappingURL=poseidon.d.ts.map