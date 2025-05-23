/**
 * BLS (Barreto-Lynn-Scott) family of pairing-friendly curves.
 * BLS != BLS.
 * The file implements BLS (Boneh-Lynn-Shacham) signatures.
 * Used in both BLS (Barreto-Lynn-Scott) and BN (Barreto-Naehrig)
 * families of pairing-friendly curves.
 * Consists of two curves: G1 and G2:
 * - G1 is a subgroup of (x, y) E(Fq) over y² = x³ + 4.
 * - G2 is a subgroup of ((x₁, x₂+i), (y₁, y₂+i)) E(Fq²) over y² = x³ + 4(1 + i) where i is √-1
 * - Gt, created by bilinear (ate) pairing e(G1, G2), consists of p-th roots of unity in
 *   Fq^k where k is embedding degree. Only degree 12 is currently supported, 24 is not.
 * Pairing is used to aggregate and verify signatures.
 * There are two main ways to use it:
 * 1. Fp for short private keys, Fp₂ for signatures
 * 2. Fp for short signatures, Fp₂ for private keys
 * @module
 **/
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
import { type htfBasicOpts, type Opts as HTFOpts, type MapToCurve, createHasher } from './hash-to-curve.ts';
import { type IField } from './modular.ts';
import type { Fp12, Fp12Bls, Fp2, Fp2Bls, Fp6 } from './tower.ts';
import { type CHash, type Hex, type PrivKey } from './utils.ts';
import { type CurvePointsRes, type CurvePointsType, type ProjPointType } from './weierstrass.ts';
type Fp = bigint;
export type TwistType = 'multiplicative' | 'divisive';
export type ShortSignatureCoder<Fp> = {
    fromHex(hex: Hex): ProjPointType<Fp>;
    toRawBytes(point: ProjPointType<Fp>): Uint8Array;
    toHex(point: ProjPointType<Fp>): string;
};
export type SignatureCoder<Fp> = {
    fromHex(hex: Hex): ProjPointType<Fp>;
    toRawBytes(point: ProjPointType<Fp>): Uint8Array;
    toHex(point: ProjPointType<Fp>): string;
};
export type PostPrecomputePointAddFn = (Rx: Fp2, Ry: Fp2, Rz: Fp2, Qx: Fp2, Qy: Fp2) => {
    Rx: Fp2;
    Ry: Fp2;
    Rz: Fp2;
};
export type PostPrecomputeFn = (Rx: Fp2, Ry: Fp2, Rz: Fp2, Qx: Fp2, Qy: Fp2, pointAdd: PostPrecomputePointAddFn) => void;
export type CurveType = {
    G1: Omit<CurvePointsType<Fp>, 'n'> & {
        ShortSignature: SignatureCoder<Fp>;
        mapToCurve: MapToCurve<Fp>;
        htfDefaults: HTFOpts;
    };
    G2: Omit<CurvePointsType<Fp2>, 'n'> & {
        Signature: SignatureCoder<Fp2>;
        mapToCurve: MapToCurve<Fp2>;
        htfDefaults: HTFOpts;
    };
    fields: {
        Fp: IField<Fp>;
        Fr: IField<bigint>;
        Fp2: Fp2Bls;
        Fp6: IField<Fp6>;
        Fp12: Fp12Bls;
    };
    params: {
        ateLoopSize: bigint;
        xNegative: boolean;
        r: bigint;
        twistType: TwistType;
    };
    htfDefaults: HTFOpts;
    hash: CHash;
    randomBytes: (bytesLength?: number) => Uint8Array;
    postPrecompute?: PostPrecomputeFn;
};
type PrecomputeSingle = [Fp2, Fp2, Fp2][];
type Precompute = PrecomputeSingle[];
export type CurveFn = {
    getPublicKey: (privateKey: PrivKey) => Uint8Array;
    getPublicKeyForShortSignatures: (privateKey: PrivKey) => Uint8Array;
    sign: {
        (message: Hex, privateKey: PrivKey, htfOpts?: htfBasicOpts): Uint8Array;
        (message: ProjPointType<Fp2>, privateKey: PrivKey, htfOpts?: htfBasicOpts): ProjPointType<Fp2>;
    };
    signShortSignature: {
        (message: Hex, privateKey: PrivKey, htfOpts?: htfBasicOpts): Uint8Array;
        (message: ProjPointType<Fp>, privateKey: PrivKey, htfOpts?: htfBasicOpts): ProjPointType<Fp>;
    };
    verify: (signature: Hex | ProjPointType<Fp2>, message: Hex | ProjPointType<Fp2>, publicKey: Hex | ProjPointType<Fp>, htfOpts?: htfBasicOpts) => boolean;
    verifyShortSignature: (signature: Hex | ProjPointType<Fp>, message: Hex | ProjPointType<Fp>, publicKey: Hex | ProjPointType<Fp2>, htfOpts?: htfBasicOpts) => boolean;
    verifyBatch: (signature: Hex | ProjPointType<Fp2>, messages: (Hex | ProjPointType<Fp2>)[], publicKeys: (Hex | ProjPointType<Fp>)[], htfOpts?: htfBasicOpts) => boolean;
    aggregatePublicKeys: {
        (publicKeys: Hex[]): Uint8Array;
        (publicKeys: ProjPointType<Fp>[]): ProjPointType<Fp>;
    };
    aggregateSignatures: {
        (signatures: Hex[]): Uint8Array;
        (signatures: ProjPointType<Fp2>[]): ProjPointType<Fp2>;
    };
    aggregateShortSignatures: {
        (signatures: Hex[]): Uint8Array;
        (signatures: ProjPointType<Fp>[]): ProjPointType<Fp>;
    };
    millerLoopBatch: (pairs: [Precompute, Fp, Fp][]) => Fp12;
    pairing: (P: ProjPointType<Fp>, Q: ProjPointType<Fp2>, withFinalExponent?: boolean) => Fp12;
    pairingBatch: (pairs: {
        g1: ProjPointType<Fp>;
        g2: ProjPointType<Fp2>;
    }[], withFinalExponent?: boolean) => Fp12;
    G1: CurvePointsRes<Fp> & ReturnType<typeof createHasher<Fp>>;
    G2: CurvePointsRes<Fp2> & ReturnType<typeof createHasher<Fp2>>;
    Signature: SignatureCoder<Fp2>;
    ShortSignature: ShortSignatureCoder<Fp>;
    params: {
        ateLoopSize: bigint;
        r: bigint;
        G1b: bigint;
        G2b: Fp2;
    };
    fields: {
        Fp: IField<Fp>;
        Fp2: Fp2Bls;
        Fp6: IField<Fp6>;
        Fp12: Fp12Bls;
        Fr: IField<bigint>;
    };
    utils: {
        randomPrivateKey: () => Uint8Array;
        calcPairingPrecomputes: (p: ProjPointType<Fp2>) => Precompute;
    };
};
export declare function bls(CURVE: CurveType): CurveFn;
export {};
//# sourceMappingURL=bls.d.ts.map