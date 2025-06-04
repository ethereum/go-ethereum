/**
 * Towered extension fields.
 * Rather than implementing a massive 12th-degree extension directly, it is more efficient
 * to build it up from smaller extensions: a tower of extensions.
 *
 * For BLS12-381, the Fp12 field is implemented as a quadratic (degree two) extension,
 * on top of a cubic (degree three) extension, on top of a quadratic extension of Fp.
 *
 * For more info: "Pairings for beginners" by Costello, section 7.3.
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
import * as mod from './modular.ts';
import type { ProjConstructor, ProjPointType } from './weierstrass.ts';
export type BigintTuple = [bigint, bigint];
export type Fp = bigint;
export type Fp2 = {
    c0: bigint;
    c1: bigint;
};
export type BigintSix = [bigint, bigint, bigint, bigint, bigint, bigint];
export type Fp6 = {
    c0: Fp2;
    c1: Fp2;
    c2: Fp2;
};
export type Fp12 = {
    c0: Fp6;
    c1: Fp6;
};
export type BigintTwelve = [
    bigint,
    bigint,
    bigint,
    bigint,
    bigint,
    bigint,
    bigint,
    bigint,
    bigint,
    bigint,
    bigint,
    bigint
];
export type Fp2Bls = mod.IField<Fp2> & {
    reim: (num: Fp2) => {
        re: Fp;
        im: Fp;
    };
    mulByB: (num: Fp2) => Fp2;
    frobeniusMap(num: Fp2, power: number): Fp2;
    fromBigTuple(num: [bigint, bigint]): Fp2;
};
export type Fp12Bls = mod.IField<Fp12> & {
    frobeniusMap(num: Fp12, power: number): Fp12;
    mul014(num: Fp12, o0: Fp2, o1: Fp2, o4: Fp2): Fp12;
    mul034(num: Fp12, o0: Fp2, o3: Fp2, o4: Fp2): Fp12;
    conjugate(num: Fp12): Fp12;
    finalExponentiate(num: Fp12): Fp12;
    fromBigTwelve(num: BigintTwelve): Fp12;
};
export declare function psiFrobenius(Fp: mod.IField<Fp>, Fp2: Fp2Bls, base: Fp2): {
    psi: (x: Fp2, y: Fp2) => [Fp2, Fp2];
    psi2: (x: Fp2, y: Fp2) => [Fp2, Fp2];
    G2psi: (c: ProjConstructor<Fp2>, P: ProjPointType<Fp2>) => ProjPointType<Fp2>;
    G2psi2: (c: ProjConstructor<Fp2>, P: ProjPointType<Fp2>) => ProjPointType<Fp2>;
    PSI_X: Fp2;
    PSI_Y: Fp2;
    PSI2_X: Fp2;
    PSI2_Y: Fp2;
};
export type Tower12Opts = {
    ORDER: bigint;
    NONRESIDUE?: Fp;
    FP2_NONRESIDUE: BigintTuple;
    Fp2sqrt?: (num: Fp2) => Fp2;
    Fp2mulByB: (num: Fp2) => Fp2;
    Fp12cyclotomicSquare: (num: Fp12) => Fp12;
    Fp12cyclotomicExp: (num: Fp12, n: bigint) => Fp12;
    Fp12finalExponentiate: (num: Fp12) => Fp12;
};
export declare function tower12(opts: Tower12Opts): {
    Fp: Readonly<mod.IField<bigint> & Required<Pick<mod.IField<bigint>, 'isOdd'>>>;
    Fp2: mod.IField<Fp2> & {
        NONRESIDUE: Fp2;
        fromBigTuple: (tuple: BigintTuple | bigint[]) => Fp2;
        reim: (num: Fp2) => {
            re: bigint;
            im: bigint;
        };
        mulByNonresidue: (num: Fp2) => Fp2;
        mulByB: (num: Fp2) => Fp2;
        frobeniusMap(num: Fp2, power: number): Fp2;
    };
    Fp6: mod.IField<Fp6> & {
        fromBigSix: (tuple: BigintSix) => Fp6;
        mulByNonresidue: (num: Fp6) => Fp6;
        frobeniusMap(num: Fp6, power: number): Fp6;
        mul1(num: Fp6, b1: Fp2): Fp6;
        mul01(num: Fp6, b0: Fp2, b1: Fp2): Fp6;
        mulByFp2(lhs: Fp6, rhs: Fp2): Fp6;
    };
    Fp4Square: (a: Fp2, b: Fp2) => {
        first: Fp2;
        second: Fp2;
    };
    Fp12: mod.IField<Fp12> & {
        fromBigTwelve: (t: BigintTwelve) => Fp12;
        frobeniusMap(num: Fp12, power: number): Fp12;
        mul014(num: Fp12, o0: Fp2, o1: Fp2, o4: Fp2): Fp12;
        mul034(num: Fp12, o0: Fp2, o3: Fp2, o4: Fp2): Fp12;
        mulByFp2(lhs: Fp12, rhs: Fp2): Fp12;
        conjugate(num: Fp12): Fp12;
        finalExponentiate(num: Fp12): Fp12;
        _cyclotomicSquare(num: Fp12): Fp12;
        _cyclotomicExp(num: Fp12, n: bigint): Fp12;
    };
};
//# sourceMappingURL=tower.d.ts.map