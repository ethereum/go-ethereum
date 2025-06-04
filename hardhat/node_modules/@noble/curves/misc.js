"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.vesta = exports.pallas = exports.pasta_q = exports.pasta_p = exports.babyjubjub = exports.jubjub = void 0;
exports.jubjub_groupHash = jubjub_groupHash;
exports.jubjub_findGroupHash = jubjub_findGroupHash;
/**
 * Miscellaneous, rarely used curves.
 * jubjub, babyjubjub, pallas, vesta.
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
const blake1_1 = require("@noble/hashes/blake1");
const blake2s_1 = require("@noble/hashes/blake2s");
const sha2_1 = require("@noble/hashes/sha2");
const utils_1 = require("@noble/hashes/utils");
const _shortw_utils_ts_1 = require("./_shortw_utils.js");
const edwards_ts_1 = require("./abstract/edwards.js");
const modular_ts_1 = require("./abstract/modular.js");
const weierstrass_ts_1 = require("./abstract/weierstrass.js");
// Jubjub curves have 𝔽p over scalar fields of other curves. They are friendly to ZK proofs.
// jubjub Fp = bls n. babyjubjub Fp = bn254 n.
// verify manually, check bls12-381.ts and bn254.ts.
// https://neuromancer.sk/std/other/JubJub
const bls12_381_Fr = (0, modular_ts_1.Field)(BigInt('0x73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001'));
const bn254_Fr = (0, modular_ts_1.Field)(BigInt('21888242871839275222246405745257275088548364400416034343698204186575808495617'));
/** Curve over scalar field of bls12-381. jubjub Fp = bls n */
exports.jubjub = (0, edwards_ts_1.twistedEdwards)({
    a: BigInt('0x73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000000'),
    d: BigInt('0x2a9318e74bfa2b48f5fd9207e6bd7fd4292d7f6d37579d2601065fd6d6343eb1'),
    Fp: bls12_381_Fr,
    n: BigInt('0xe7db4ea6533afa906673b0101343b00a6682093ccc81082d0970e5ed6f72cb7'),
    h: BigInt(8),
    Gx: BigInt('0x11dafe5d23e1218086a365b99fbf3d3be72f6afd7d1f72623e6b071492d1122b'),
    Gy: BigInt('0x1d523cf1ddab1a1793132e78c866c0c33e26ba5cc220fed7cc3f870e59d292aa'),
    hash: sha2_1.sha512,
    randomBytes: utils_1.randomBytes,
});
/** Curve over scalar field of bn254. babyjubjub Fp = bn254 n */
exports.babyjubjub = (0, edwards_ts_1.twistedEdwards)({
    a: BigInt(168700),
    d: BigInt(168696),
    Fp: bn254_Fr,
    n: BigInt('21888242871839275222246405745257275088614511777268538073601725287587578984328'),
    h: BigInt(8),
    Gx: BigInt('995203441582195749578291179787384436505546430278305826713579947235728471134'),
    Gy: BigInt('5472060717959818805561601436314318772137091100104008585924551046643952123905'),
    hash: blake1_1.blake256,
    randomBytes: utils_1.randomBytes,
});
const jubjub_gh_first_block = (0, utils_1.utf8ToBytes)('096b36a5804bfacef1691e173c366a47ff5ba84a44f26ddd7e8d9f79d5b42df0');
// Returns point at JubJub curve which is prime order and not zero
function jubjub_groupHash(tag, personalization) {
    const h = blake2s_1.blake2s.create({ personalization, dkLen: 32 });
    h.update(jubjub_gh_first_block);
    h.update(tag);
    // NOTE: returns ExtendedPoint, in case it will be multiplied later
    let p = exports.jubjub.ExtendedPoint.fromHex(h.digest());
    // NOTE: cannot replace with isSmallOrder, returns Point*8
    p = p.multiply(exports.jubjub.CURVE.h);
    if (p.equals(exports.jubjub.ExtendedPoint.ZERO))
        throw new Error('Point has small order');
    return p;
}
// No secret data is leaked here at all.
// It operates over public data:
// const G_SPEND = jubjub.findGroupHash(new Uint8Array(), utf8ToBytes('Item_G_'));
function jubjub_findGroupHash(m, personalization) {
    const tag = (0, utils_1.concatBytes)(m, new Uint8Array([0]));
    const hashes = [];
    for (let i = 0; i < 256; i++) {
        tag[tag.length - 1] = i;
        try {
            hashes.push(jubjub_groupHash(tag, personalization));
        }
        catch (e) { }
    }
    if (!hashes.length)
        throw new Error('findGroupHash tag overflow');
    return hashes[0];
}
// Pasta curves. See [Spec](https://o1-labs.github.io/proof-systems/specs/pasta.html).
exports.pasta_p = BigInt('0x40000000000000000000000000000000224698fc094cf91b992d30ed00000001');
exports.pasta_q = BigInt('0x40000000000000000000000000000000224698fc0994a8dd8c46eb2100000001');
/** https://neuromancer.sk/std/other/Pallas */
exports.pallas = (0, weierstrass_ts_1.weierstrass)({
    a: BigInt(0),
    b: BigInt(5),
    Fp: (0, modular_ts_1.Field)(exports.pasta_p),
    n: exports.pasta_q,
    Gx: (0, modular_ts_1.mod)(BigInt(-1), exports.pasta_p),
    Gy: BigInt(2),
    h: BigInt(1),
    ...(0, _shortw_utils_ts_1.getHash)(sha2_1.sha256),
});
/** https://neuromancer.sk/std/other/Vesta */
exports.vesta = (0, weierstrass_ts_1.weierstrass)({
    a: BigInt(0),
    b: BigInt(5),
    Fp: (0, modular_ts_1.Field)(exports.pasta_q),
    n: exports.pasta_p,
    Gx: (0, modular_ts_1.mod)(BigInt(-1), exports.pasta_q),
    Gy: BigInt(2),
    h: BigInt(1),
    ...(0, _shortw_utils_ts_1.getHash)(sha2_1.sha256),
});
//# sourceMappingURL=misc.js.map