/**
 * Twisted Edwards curve. The formula is: ax² + y² = 1 + dx²y².
 * For design rationale of types / exports, see weierstrass module documentation.
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
import { type AffinePoint, type BasicCurve, type Group, type GroupConstructor } from './curve.ts';
import { type FHash, type Hex } from './utils.ts';
/** Edwards curves must declare params a & d. */
export type CurveType = BasicCurve<bigint> & {
    a: bigint;
    d: bigint;
    hash: FHash;
    randomBytes: (bytesLength?: number) => Uint8Array;
    adjustScalarBytes?: (bytes: Uint8Array) => Uint8Array;
    domain?: (data: Uint8Array, ctx: Uint8Array, phflag: boolean) => Uint8Array;
    uvRatio?: (u: bigint, v: bigint) => {
        isValid: boolean;
        value: bigint;
    };
    prehash?: FHash;
    mapToCurve?: (scalar: bigint[]) => AffinePoint<bigint>;
};
export type CurveTypeWithLength = Readonly<CurveType & {
    nByteLength: number;
    nBitLength: number;
}>;
declare function validateOpts(curve: CurveType): CurveTypeWithLength;
/** Instance of Extended Point with coordinates in X, Y, Z, T. */
export interface ExtPointType extends Group<ExtPointType> {
    readonly ex: bigint;
    readonly ey: bigint;
    readonly ez: bigint;
    readonly et: bigint;
    get x(): bigint;
    get y(): bigint;
    assertValidity(): void;
    multiply(scalar: bigint): ExtPointType;
    multiplyUnsafe(scalar: bigint): ExtPointType;
    isSmallOrder(): boolean;
    isTorsionFree(): boolean;
    clearCofactor(): ExtPointType;
    toAffine(iz?: bigint): AffinePoint<bigint>;
    toRawBytes(isCompressed?: boolean): Uint8Array;
    toHex(isCompressed?: boolean): string;
    _setWindowSize(windowSize: number): void;
}
/** Static methods of Extended Point with coordinates in X, Y, Z, T. */
export interface ExtPointConstructor extends GroupConstructor<ExtPointType> {
    new (x: bigint, y: bigint, z: bigint, t: bigint): ExtPointType;
    fromAffine(p: AffinePoint<bigint>): ExtPointType;
    fromHex(hex: Hex): ExtPointType;
    fromPrivateKey(privateKey: Hex): ExtPointType;
    msm(points: ExtPointType[], scalars: bigint[]): ExtPointType;
}
/**
 * Edwards Curve interface.
 * Main methods: `getPublicKey(priv)`, `sign(msg, priv)`, `verify(sig, msg, pub)`.
 */
export type CurveFn = {
    CURVE: ReturnType<typeof validateOpts>;
    getPublicKey: (privateKey: Hex) => Uint8Array;
    sign: (message: Hex, privateKey: Hex, options?: {
        context?: Hex;
    }) => Uint8Array;
    verify: (sig: Hex, message: Hex, publicKey: Hex, options?: {
        context?: Hex;
        zip215: boolean;
    }) => boolean;
    ExtendedPoint: ExtPointConstructor;
    utils: {
        randomPrivateKey: () => Uint8Array;
        getExtendedPublicKey: (key: Hex) => {
            head: Uint8Array;
            prefix: Uint8Array;
            scalar: bigint;
            point: ExtPointType;
            pointBytes: Uint8Array;
        };
        precompute: (windowSize?: number, point?: ExtPointType) => ExtPointType;
    };
};
/**
 * Creates Twisted Edwards curve with EdDSA signatures.
 * @example
 * import { Field } from '@noble/curves/abstract/modular';
 * // Before that, define BigInt-s: a, d, p, n, Gx, Gy, h
 * const curve = twistedEdwards({ a, d, Fp: Field(p), n, Gx, Gy, h })
 */
export declare function twistedEdwards(curveDef: CurveType): CurveFn;
export {};
//# sourceMappingURL=edwards.d.ts.map