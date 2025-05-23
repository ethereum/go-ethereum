/**
 * Methods for elliptic curve multiplication by scalars.
 * Contains wNAF, pippenger
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
import { type IField } from './modular.ts';
export type AffinePoint<T> = {
    x: T;
    y: T;
} & {
    z?: never;
    t?: never;
};
export interface Group<T extends Group<T>> {
    double(): T;
    negate(): T;
    add(other: T): T;
    subtract(other: T): T;
    equals(other: T): boolean;
    multiply(scalar: bigint): T;
}
export type GroupConstructor<T> = {
    BASE: T;
    ZERO: T;
};
export type Mapper<T> = (i: T[]) => T[];
/** Internal wNAF opts for specific W and scalarBits */
export type WOpts = {
    windows: number;
    windowSize: number;
    mask: bigint;
    maxNumber: number;
    shiftBy: bigint;
};
export type IWNAF<T extends Group<T>> = {
    constTimeNegate: <T extends Group<T>>(condition: boolean, item: T) => T;
    hasPrecomputes(elm: T): boolean;
    unsafeLadder(elm: T, n: bigint, p?: T): T;
    precomputeWindow(elm: T, W: number): Group<T>[];
    wNAF(W: number, precomputes: T[], n: bigint): {
        p: T;
        f: T;
    };
    wNAFUnsafe(W: number, precomputes: T[], n: bigint, acc?: T): T;
    getPrecomputes(W: number, P: T, transform: Mapper<T>): T[];
    wNAFCached(P: T, n: bigint, transform: Mapper<T>): {
        p: T;
        f: T;
    };
    wNAFCachedUnsafe(P: T, n: bigint, transform: Mapper<T>, prev?: T): T;
    setWindowSize(P: T, W: number): void;
};
/**
 * Elliptic curve multiplication of Point by scalar. Fragile.
 * Scalars should always be less than curve order: this should be checked inside of a curve itself.
 * Creates precomputation tables for fast multiplication:
 * - private scalar is split by fixed size windows of W bits
 * - every window point is collected from window's table & added to accumulator
 * - since windows are different, same point inside tables won't be accessed more than once per calc
 * - each multiplication is 'Math.ceil(CURVE_ORDER / 𝑊) + 1' point additions (fixed for any scalar)
 * - +1 window is neccessary for wNAF
 * - wNAF reduces table size: 2x less memory + 2x faster generation, but 10% slower multiplication
 *
 * @todo Research returning 2d JS array of windows, instead of a single window.
 * This would allow windows to be in different memory locations
 */
export declare function wNAF<T extends Group<T>>(c: GroupConstructor<T>, bits: number): IWNAF<T>;
/**
 * Pippenger algorithm for multi-scalar multiplication (MSM, Pa + Qb + Rc + ...).
 * 30x faster vs naive addition on L=4096, 10x faster than precomputes.
 * For N=254bit, L=1, it does: 1024 ADD + 254 DBL. For L=5: 1536 ADD + 254 DBL.
 * Algorithmically constant-time (for same L), even when 1 point + scalar, or when scalar = 0.
 * @param c Curve Point constructor
 * @param fieldN field over CURVE.N - important that it's not over CURVE.P
 * @param points array of L curve points
 * @param scalars array of L scalars (aka private keys / bigints)
 */
export declare function pippenger<T extends Group<T>>(c: GroupConstructor<T>, fieldN: IField<bigint>, points: T[], scalars: bigint[]): T;
/**
 * Precomputed multi-scalar multiplication (MSM, Pa + Qb + Rc + ...).
 * @param c Curve Point constructor
 * @param fieldN field over CURVE.N - important that it's not over CURVE.P
 * @param points array of L curve points
 * @returns function which multiplies points with scaars
 */
export declare function precomputeMSMUnsafe<T extends Group<T>>(c: GroupConstructor<T>, fieldN: IField<bigint>, points: T[], windowSize: number): (scalars: bigint[]) => T;
/**
 * Generic BasicCurve interface: works even for polynomial fields (BLS): P, n, h would be ok.
 * Though generator can be different (Fp2 / Fp6 for BLS).
 */
export type BasicCurve<T> = {
    Fp: IField<T>;
    n: bigint;
    nBitLength?: number;
    nByteLength?: number;
    h: bigint;
    hEff?: bigint;
    Gx: T;
    Gy: T;
    allowInfinityPoint?: boolean;
};
export declare function validateBasic<FP, T>(curve: BasicCurve<FP> & T): Readonly<{
    readonly nBitLength: number;
    readonly nByteLength: number;
} & BasicCurve<FP> & T & {
    p: bigint;
}>;
//# sourceMappingURL=curve.d.ts.map