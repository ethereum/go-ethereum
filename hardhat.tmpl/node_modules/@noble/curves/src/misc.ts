/**
 * Miscellaneous, rarely used curves.
 * jubjub, babyjubjub, pallas, vesta.
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
import { blake256 } from '@noble/hashes/blake1';
import { blake2s } from '@noble/hashes/blake2s';
import { sha256, sha512 } from '@noble/hashes/sha2';
import { concatBytes, randomBytes, utf8ToBytes } from '@noble/hashes/utils';
import { getHash } from './_shortw_utils.ts';
import { type CurveFn, type ExtPointType, twistedEdwards } from './abstract/edwards.ts';
import { Field, mod } from './abstract/modular.ts';
import { type CurveFn as WCurveFn, weierstrass } from './abstract/weierstrass.ts';

// Jubjub curves have 𝔽p over scalar fields of other curves. They are friendly to ZK proofs.
// jubjub Fp = bls n. babyjubjub Fp = bn254 n.
// verify manually, check bls12-381.ts and bn254.ts.
// https://neuromancer.sk/std/other/JubJub

const bls12_381_Fr = Field(
  BigInt('0x73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001')
);
const bn254_Fr = Field(
  BigInt('21888242871839275222246405745257275088548364400416034343698204186575808495617')
);

/** Curve over scalar field of bls12-381. jubjub Fp = bls n */
export const jubjub: CurveFn = /* @__PURE__ */ twistedEdwards({
  a: BigInt('0x73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000000'),
  d: BigInt('0x2a9318e74bfa2b48f5fd9207e6bd7fd4292d7f6d37579d2601065fd6d6343eb1'),
  Fp: bls12_381_Fr,
  n: BigInt('0xe7db4ea6533afa906673b0101343b00a6682093ccc81082d0970e5ed6f72cb7'),
  h: BigInt(8),
  Gx: BigInt('0x11dafe5d23e1218086a365b99fbf3d3be72f6afd7d1f72623e6b071492d1122b'),
  Gy: BigInt('0x1d523cf1ddab1a1793132e78c866c0c33e26ba5cc220fed7cc3f870e59d292aa'),
  hash: sha512,
  randomBytes,
} as const);

/** Curve over scalar field of bn254. babyjubjub Fp = bn254 n */
export const babyjubjub: CurveFn = /* @__PURE__ */ twistedEdwards({
  a: BigInt(168700),
  d: BigInt(168696),
  Fp: bn254_Fr,
  n: BigInt('21888242871839275222246405745257275088614511777268538073601725287587578984328'),
  h: BigInt(8),
  Gx: BigInt('995203441582195749578291179787384436505546430278305826713579947235728471134'),
  Gy: BigInt('5472060717959818805561601436314318772137091100104008585924551046643952123905'),
  hash: blake256,
  randomBytes,
} as const);

const jubjub_gh_first_block = utf8ToBytes(
  '096b36a5804bfacef1691e173c366a47ff5ba84a44f26ddd7e8d9f79d5b42df0'
);

// Returns point at JubJub curve which is prime order and not zero
export function jubjub_groupHash(tag: Uint8Array, personalization: Uint8Array): ExtPointType {
  const h = blake2s.create({ personalization, dkLen: 32 });
  h.update(jubjub_gh_first_block);
  h.update(tag);
  // NOTE: returns ExtendedPoint, in case it will be multiplied later
  let p = jubjub.ExtendedPoint.fromHex(h.digest());
  // NOTE: cannot replace with isSmallOrder, returns Point*8
  p = p.multiply(jubjub.CURVE.h);
  if (p.equals(jubjub.ExtendedPoint.ZERO)) throw new Error('Point has small order');
  return p;
}

// No secret data is leaked here at all.
// It operates over public data:
// const G_SPEND = jubjub.findGroupHash(new Uint8Array(), utf8ToBytes('Item_G_'));
export function jubjub_findGroupHash(m: Uint8Array, personalization: Uint8Array): ExtPointType {
  const tag = concatBytes(m, new Uint8Array([0]));
  const hashes = [];
  for (let i = 0; i < 256; i++) {
    tag[tag.length - 1] = i;
    try {
      hashes.push(jubjub_groupHash(tag, personalization));
    } catch (e) {}
  }
  if (!hashes.length) throw new Error('findGroupHash tag overflow');
  return hashes[0];
}

// Pasta curves. See [Spec](https://o1-labs.github.io/proof-systems/specs/pasta.html).

export const pasta_p: bigint = BigInt(
  '0x40000000000000000000000000000000224698fc094cf91b992d30ed00000001'
);
export const pasta_q: bigint = BigInt(
  '0x40000000000000000000000000000000224698fc0994a8dd8c46eb2100000001'
);

/** https://neuromancer.sk/std/other/Pallas */
export const pallas: WCurveFn = weierstrass({
  a: BigInt(0),
  b: BigInt(5),
  Fp: Field(pasta_p),
  n: pasta_q,
  Gx: mod(BigInt(-1), pasta_p),
  Gy: BigInt(2),
  h: BigInt(1),
  ...getHash(sha256),
});
/** https://neuromancer.sk/std/other/Vesta */
export const vesta: WCurveFn = weierstrass({
  a: BigInt(0),
  b: BigInt(5),
  Fp: Field(pasta_q),
  n: pasta_p,
  Gx: mod(BigInt(-1), pasta_q),
  Gy: BigInt(2),
  h: BigInt(1),
  ...getHash(sha256),
});
