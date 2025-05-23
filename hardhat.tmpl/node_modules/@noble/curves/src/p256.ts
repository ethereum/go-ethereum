/**
 * NIST secp256r1 aka p256.
 * https://www.secg.org/sec2-v2.pdf, https://neuromancer.sk/std/nist/P-256
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
import { sha256 } from '@noble/hashes/sha2';
import { createCurve, type CurveFnWithCreate } from './_shortw_utils.ts';
import { createHasher, type HTFMethod } from './abstract/hash-to-curve.ts';
import { Field } from './abstract/modular.ts';
import { mapToCurveSimpleSWU } from './abstract/weierstrass.ts';

const Fp256 = Field(BigInt('0xffffffff00000001000000000000000000000000ffffffffffffffffffffffff'));
const CURVE_A = Fp256.create(BigInt('-3'));
const CURVE_B = BigInt('0x5ac635d8aa3a93e7b3ebbd55769886bc651d06b0cc53b0f63bce3c3e27d2604b');

/**
 * secp256r1 curve, ECDSA and ECDH methods.
 * Field: `2n**224n * (2n**32n-1n) + 2n**192n + 2n**96n-1n`
 */
// prettier-ignore
export const p256: CurveFnWithCreate = createCurve({
  a: CURVE_A,
  b: CURVE_B,
  Fp: Fp256,
  n: BigInt('0xffffffff00000000ffffffffffffffffbce6faada7179e84f3b9cac2fc632551'),
  Gx: BigInt('0x6b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296'),
  Gy: BigInt('0x4fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5'),
  h: BigInt(1),
  lowS: false,
} as const, sha256);
/** Alias to p256. */
export const secp256r1: CurveFnWithCreate = p256;

const mapSWU = /* @__PURE__ */ (() =>
  mapToCurveSimpleSWU(Fp256, {
    A: CURVE_A,
    B: CURVE_B,
    Z: Fp256.create(BigInt('-10')),
  }))();

const htf = /* @__PURE__ */ (() =>
  createHasher(secp256r1.ProjectivePoint, (scalars: bigint[]) => mapSWU(scalars[0]), {
    DST: 'P256_XMD:SHA-256_SSWU_RO_',
    encodeDST: 'P256_XMD:SHA-256_SSWU_NU_',
    p: Fp256.ORDER,
    m: 1,
    k: 128,
    expand: 'xmd',
    hash: sha256,
  }))();
/** secp256r1 hash-to-curve from RFC 9380. */
export const hashToCurve: HTFMethod<bigint> = /* @__PURE__ */ (() => htf.hashToCurve)();
/** secp256r1 encode-to-curve from RFC 9380. */
export const encodeToCurve: HTFMethod<bigint> = /* @__PURE__ */ (() => htf.encodeToCurve)();
