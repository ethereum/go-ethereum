/**
 * NIST secp384r1 aka p384.
 * https://www.secg.org/sec2-v2.pdf, https://neuromancer.sk/std/nist/P-384
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
import { sha384 } from '@noble/hashes/sha2';
import { createCurve, type CurveFnWithCreate } from './_shortw_utils.ts';
import { createHasher, type HTFMethod } from './abstract/hash-to-curve.ts';
import { Field } from './abstract/modular.ts';
import { mapToCurveSimpleSWU } from './abstract/weierstrass.ts';

// Field over which we'll do calculations.
const Fp384 = Field(
  BigInt(
    '0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffeffffffff0000000000000000ffffffff'
  )
);
const CURVE_A = Fp384.create(BigInt('-3'));
// prettier-ignore
const CURVE_B = BigInt('0xb3312fa7e23ee7e4988e056be3f82d19181d9c6efe8141120314088f5013875ac656398d8a2ed19d2a85c8edd3ec2aef');

/**
 * secp384r1 curve, ECDSA and ECDH methods.
 * Field: `2n**384n - 2n**128n - 2n**96n + 2n**32n - 1n`.
 * */
// prettier-ignore
export const p384: CurveFnWithCreate = createCurve({
  a: CURVE_A,
  b: CURVE_B,
  Fp: Fp384,
  n: BigInt('0xffffffffffffffffffffffffffffffffffffffffffffffffc7634d81f4372ddf581a0db248b0a77aecec196accc52973'),
  Gx: BigInt('0xaa87ca22be8b05378eb1c71ef320ad746e1d3b628ba79b9859f741e082542a385502f25dbf55296c3a545e3872760ab7'),
  Gy: BigInt('0x3617de4a96262c6f5d9e98bf9292dc29f8f41dbd289a147ce9da3113b5f0b8c00a60b1ce1d7e819d7a431d7c90ea0e5f'),
  h: BigInt(1),
  lowS: false,
} as const, sha384);
/** Alias to p384. */
export const secp384r1: CurveFnWithCreate = p384;

const mapSWU = /* @__PURE__ */ (() =>
  mapToCurveSimpleSWU(Fp384, {
    A: CURVE_A,
    B: CURVE_B,
    Z: Fp384.create(BigInt('-12')),
  }))();

const htf = /* @__PURE__ */ (() =>
  createHasher(secp384r1.ProjectivePoint, (scalars: bigint[]) => mapSWU(scalars[0]), {
    DST: 'P384_XMD:SHA-384_SSWU_RO_',
    encodeDST: 'P384_XMD:SHA-384_SSWU_NU_',
    p: Fp384.ORDER,
    m: 1,
    k: 192,
    expand: 'xmd',
    hash: sha384,
  }))();
/** secp384r1 hash-to-curve from RFC 9380. */
export const hashToCurve: HTFMethod<bigint> = /* @__PURE__ */ (() => htf.hashToCurve)();
/** secp384r1 encode-to-curve from RFC 9380. */
export const encodeToCurve: HTFMethod<bigint> = /* @__PURE__ */ (() => htf.encodeToCurve)();
