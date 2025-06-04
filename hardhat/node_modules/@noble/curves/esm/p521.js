/**
 * NIST secp521r1 aka p521.
 * Note that it's 521, which differs from 512 of its hash function.
 * https://www.secg.org/sec2-v2.pdf, https://neuromancer.sk/std/nist/P-521
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
import { sha512 } from '@noble/hashes/sha2';
import { createCurve } from "./_shortw_utils.js";
import { createHasher } from "./abstract/hash-to-curve.js";
import { Field } from "./abstract/modular.js";
import { mapToCurveSimpleSWU } from "./abstract/weierstrass.js";
// Field over which we'll do calculations.
const Fp521 = Field(BigInt('0x1ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff'));
const CURVE_A = Fp521.create(BigInt('-3'));
const CURVE_B = BigInt('0x0051953eb9618e1c9a1f929a21a0b68540eea2da725b99b315f3b8b489918ef109e156193951ec7e937b1652c0bd3bb1bf073573df883d2c34f1ef451fd46b503f00');
/**
 * NIST secp521r1 aka p521 curve, ECDSA and ECDH methods.
 * Field: `2n**521n - 1n`.
 */
// prettier-ignore
export const p521 = createCurve({
    a: CURVE_A,
    b: CURVE_B,
    Fp: Fp521,
    n: BigInt('0x01fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa51868783bf2f966b7fcc0148f709a5d03bb5c9b8899c47aebb6fb71e91386409'),
    Gx: BigInt('0x00c6858e06b70404e9cd9e3ecb662395b4429c648139053fb521f828af606b4d3dbaa14b5e77efe75928fe1dc127a2ffa8de3348b3c1856a429bf97e7e31c2e5bd66'),
    Gy: BigInt('0x011839296a789a3bc0045c8a5fb42c7d1bd998f54449579b446817afbd17273e662c97ee72995ef42640c550b9013fad0761353c7086a272c24088be94769fd16650'),
    h: BigInt(1),
    lowS: false,
    allowedPrivateKeyLengths: [130, 131, 132] // P521 keys are variable-length. Normalize to 132b
}, sha512);
export const secp521r1 = p521;
const mapSWU = /* @__PURE__ */ (() => mapToCurveSimpleSWU(Fp521, {
    A: CURVE_A,
    B: CURVE_B,
    Z: Fp521.create(BigInt('-4')),
}))();
const htf = /* @__PURE__ */ (() => createHasher(secp521r1.ProjectivePoint, (scalars) => mapSWU(scalars[0]), {
    DST: 'P521_XMD:SHA-512_SSWU_RO_',
    encodeDST: 'P521_XMD:SHA-512_SSWU_NU_',
    p: Fp521.ORDER,
    m: 1,
    k: 256,
    expand: 'xmd',
    hash: sha512,
}))();
/** secp521r1 hash-to-curve from RFC 9380. */
export const hashToCurve = /* @__PURE__ */ (() => htf.hashToCurve)();
/** secp521r1 encode-to-curve from RFC 9380. */
export const encodeToCurve = /* @__PURE__ */ (() => htf.encodeToCurve)();
//# sourceMappingURL=p521.js.map