import { type AffinePoint, type Group } from './abstract/curve.ts';
import { type CurveFn, type ExtPointType } from './abstract/edwards.ts';
import { type htfBasicOpts, type HTFMethod } from './abstract/hash-to-curve.ts';
import { type CurveFn as XCurveFn } from './abstract/montgomery.ts';
import { type Hex } from './abstract/utils.ts';
/** Weird / bogus points, useful for debugging. */
export declare const ED25519_TORSION_SUBGROUP: string[];
/**
 * ed25519 curve with EdDSA signatures.
 * @example
 * import { ed25519 } from '@noble/curves/ed25519';
 * const priv = ed25519.utils.randomPrivateKey();
 * const pub = ed25519.getPublicKey(priv);
 * const msg = new TextEncoder().encode('hello');
 * const sig = ed25519.sign(msg, priv);
 * ed25519.verify(sig, msg, pub); // Default mode: follows ZIP215
 * ed25519.verify(sig, msg, pub, { zip215: false }); // RFC8032 / FIPS 186-5
 */
export declare const ed25519: CurveFn;
export declare const ed25519ctx: CurveFn;
export declare const ed25519ph: CurveFn;
/**
 * ECDH using curve25519 aka x25519.
 * @example
 * import { x25519 } from '@noble/curves/ed25519';
 * const priv = 'a546e36bf0527c9d3b16154b82465edd62144c0ac1fc5a18506a2244ba449ac4';
 * const pub = 'e6db6867583030db3594c1a424b15f7c726624ec26b3353b10a903a6d0ab1c4c';
 * x25519.getSharedSecret(priv, pub) === x25519.scalarMult(priv, pub); // aliases
 * x25519.getPublicKey(priv) === x25519.scalarMultBase(priv);
 * x25519.getPublicKey(x25519.utils.randomPrivateKey());
 */
export declare const x25519: XCurveFn;
/**
 * Converts ed25519 public key to x25519 public key. Uses formula:
 * * `(u, v) = ((1+y)/(1-y), sqrt(-486664)*u/x)`
 * * `(x, y) = (sqrt(-486664)*u/v, (u-1)/(u+1))`
 * @example
 *   const someonesPub = ed25519.getPublicKey(ed25519.utils.randomPrivateKey());
 *   const aPriv = x25519.utils.randomPrivateKey();
 *   x25519.getSharedSecret(aPriv, edwardsToMontgomeryPub(someonesPub))
 */
export declare function edwardsToMontgomeryPub(edwardsPub: Hex): Uint8Array;
export declare const edwardsToMontgomery: typeof edwardsToMontgomeryPub;
/**
 * Converts ed25519 secret key to x25519 secret key.
 * @example
 *   const someonesPub = x25519.getPublicKey(x25519.utils.randomPrivateKey());
 *   const aPriv = ed25519.utils.randomPrivateKey();
 *   x25519.getSharedSecret(edwardsToMontgomeryPriv(aPriv), someonesPub)
 */
export declare function edwardsToMontgomeryPriv(edwardsPriv: Uint8Array): Uint8Array;
export declare const hashToCurve: HTFMethod<bigint>;
export declare const encodeToCurve: HTFMethod<bigint>;
type ExtendedPoint = ExtPointType;
/**
 * Each ed25519/ExtendedPoint has 8 different equivalent points. This can be
 * a source of bugs for protocols like ring signatures. Ristretto was created to solve this.
 * Ristretto point operates in X:Y:Z:T extended coordinates like ExtendedPoint,
 * but it should work in its own namespace: do not combine those two.
 * https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-ristretto255-decaf448
 */
declare class RistPoint implements Group<RistPoint> {
    static BASE: RistPoint;
    static ZERO: RistPoint;
    private readonly ep;
    constructor(ep: ExtendedPoint);
    static fromAffine(ap: AffinePoint<bigint>): RistPoint;
    /**
     * Takes uniform output of 64-byte hash function like sha512 and converts it to `RistrettoPoint`.
     * The hash-to-group operation applies Elligator twice and adds the results.
     * **Note:** this is one-way map, there is no conversion from point to hash.
     * https://ristretto.group/formulas/elligator.html
     * @param hex 64-byte output of a hash function
     */
    static hashToCurve(hex: Hex): RistPoint;
    /**
     * Converts ristretto-encoded string to ristretto point.
     * https://ristretto.group/formulas/decoding.html
     * @param hex Ristretto-encoded 32 bytes. Not every 32-byte string is valid ristretto encoding
     */
    static fromHex(hex: Hex): RistPoint;
    static msm(points: RistPoint[], scalars: bigint[]): RistPoint;
    /**
     * Encodes ristretto point to Uint8Array.
     * https://ristretto.group/formulas/encoding.html
     */
    toRawBytes(): Uint8Array;
    toHex(): string;
    toString(): string;
    equals(other: RistPoint): boolean;
    add(other: RistPoint): RistPoint;
    subtract(other: RistPoint): RistPoint;
    multiply(scalar: bigint): RistPoint;
    multiplyUnsafe(scalar: bigint): RistPoint;
    double(): RistPoint;
    negate(): RistPoint;
}
export declare const RistrettoPoint: typeof RistPoint;
export declare const hashToRistretto255: (msg: Uint8Array, options: htfBasicOpts) => RistPoint;
/** @deprecated */
export declare const hash_to_ristretto255: (msg: Uint8Array, options: htfBasicOpts) => RistPoint;
export {};
//# sourceMappingURL=ed25519.d.ts.map