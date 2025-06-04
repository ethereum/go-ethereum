"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.encodeToCurve = exports.hashToCurve = exports.secp256r1 = exports.p256 = void 0;
/**
 * NIST secp256r1 aka p256.
 * https://www.secg.org/sec2-v2.pdf, https://neuromancer.sk/std/nist/P-256
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
const sha2_1 = require("@noble/hashes/sha2");
const _shortw_utils_ts_1 = require("./_shortw_utils.js");
const hash_to_curve_ts_1 = require("./abstract/hash-to-curve.js");
const modular_ts_1 = require("./abstract/modular.js");
const weierstrass_ts_1 = require("./abstract/weierstrass.js");
const Fp256 = (0, modular_ts_1.Field)(BigInt('0xffffffff00000001000000000000000000000000ffffffffffffffffffffffff'));
const CURVE_A = Fp256.create(BigInt('-3'));
const CURVE_B = BigInt('0x5ac635d8aa3a93e7b3ebbd55769886bc651d06b0cc53b0f63bce3c3e27d2604b');
/**
 * secp256r1 curve, ECDSA and ECDH methods.
 * Field: `2n**224n * (2n**32n-1n) + 2n**192n + 2n**96n-1n`
 */
// prettier-ignore
exports.p256 = (0, _shortw_utils_ts_1.createCurve)({
    a: CURVE_A,
    b: CURVE_B,
    Fp: Fp256,
    n: BigInt('0xffffffff00000000ffffffffffffffffbce6faada7179e84f3b9cac2fc632551'),
    Gx: BigInt('0x6b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296'),
    Gy: BigInt('0x4fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5'),
    h: BigInt(1),
    lowS: false,
}, sha2_1.sha256);
/** Alias to p256. */
exports.secp256r1 = exports.p256;
const mapSWU = /* @__PURE__ */ (() => (0, weierstrass_ts_1.mapToCurveSimpleSWU)(Fp256, {
    A: CURVE_A,
    B: CURVE_B,
    Z: Fp256.create(BigInt('-10')),
}))();
const htf = /* @__PURE__ */ (() => (0, hash_to_curve_ts_1.createHasher)(exports.secp256r1.ProjectivePoint, (scalars) => mapSWU(scalars[0]), {
    DST: 'P256_XMD:SHA-256_SSWU_RO_',
    encodeDST: 'P256_XMD:SHA-256_SSWU_NU_',
    p: Fp256.ORDER,
    m: 1,
    k: 128,
    expand: 'xmd',
    hash: sha2_1.sha256,
}))();
/** secp256r1 hash-to-curve from RFC 9380. */
exports.hashToCurve = (() => htf.hashToCurve)();
/** secp256r1 encode-to-curve from RFC 9380. */
exports.encodeToCurve = (() => htf.encodeToCurve)();
//# sourceMappingURL=p256.js.map