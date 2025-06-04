/**
 * bn254, previously known as alt_bn_128, when it had 128-bit security.

Barbulescu-Duquesne 2017 shown it's weaker: just about 100 bits,
so the naming has been adjusted to its prime bit count:
https://hal.science/hal-01534101/file/main.pdf.
Compatible with EIP-196 and EIP-197.

There are huge compatibility issues in the ecosystem:

1. Different libraries call it in different ways: "bn254", "bn256", "alt_bn128", "bn128".
2. libff has bn128, but it's a different curve with different G2:
   https://github.com/scipr-lab/libff/blob/a44f482e18b8ac04d034c193bd9d7df7817ad73f/libff/algebra/curves/bn128/bn128_init.cpp#L166-L169
3. halo2curves bn256 is also incompatible and returns different outputs

The goal of our implementation is to support "Ethereum" variant of the curve,
because it at least has specs:

- EIP196 (https://eips.ethereum.org/EIPS/eip-196) describes bn254 ECADD and ECMUL opcodes for EVM
- EIP197 (https://eips.ethereum.org/EIPS/eip-197) describes bn254 pairings
- It's hard: EIPs don't have proper tests. EIP-197 returns boolean output instead of Fp12
- The existing implementations are bad. Some are deprecated:
    - https://github.com/paritytech/bn (old version)
    - https://github.com/ewasm/ethereum-bn128.rs (uses paritytech/bn)
    - https://github.com/zcash-hackworks/bn
    - https://github.com/arkworks-rs/curves/blob/master/bn254/src/lib.rs
- Python implementations use different towers and produce different Fp12 outputs:
    - https://github.com/ethereum/py_pairing
    - https://github.com/ethereum/execution-specs/blob/master/src/ethereum/crypto/alt_bn128.py
- Points are encoded differently in different implementations

### Params
Seed (X): 4965661367192848881
Fr: (36x⁴+36x³+18x²+6x+1)
Fp: (36x⁴+36x³+24x²+6x+1)
(E  / Fp ): Y² = X³+3
(Et / Fp²): Y² = X³+3/(u+9) (D-type twist)
Ate loop size: 6x+2

### Towers
- Fp²[u] = Fp/u²+1
- Fp⁶[v] = Fp²/v³-9-u
- Fp¹²[w] = Fp⁶/w²-v

 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
import { sha256 } from '@noble/hashes/sha2';
import { randomBytes } from '@noble/hashes/utils';
import { getHash } from './_shortw_utils.ts';
import {
  bls,
  type CurveFn as BLSCurveFn,
  type PostPrecomputeFn,
  type PostPrecomputePointAddFn,
} from './abstract/bls.ts';
import { Field } from './abstract/modular.ts';
import type { Fp, Fp12, Fp2, Fp6 } from './abstract/tower.ts';
import { psiFrobenius, tower12 } from './abstract/tower.ts';
import { bitGet, bitLen, notImplemented } from './abstract/utils.ts';
import { type CurveFn, weierstrass } from './abstract/weierstrass.ts';
// prettier-ignore
const _1n = BigInt(1), _2n = BigInt(2), _3n = BigInt(3);
const _6n = BigInt(6);

const BN_X = BigInt('4965661367192848881');
const BN_X_LEN = bitLen(BN_X);
const SIX_X_SQUARED = _6n * BN_X ** _2n;

// Finite field over r. It's for convenience and is not used in the code below.
const Fr = Field(
  BigInt('21888242871839275222246405745257275088548364400416034343698204186575808495617')
);
// Fp2.div(Fp2.mul(Fp2.ONE, _3n), Fp2.NONRESIDUE)
const Fp2B = {
  c0: BigInt('19485874751759354771024239261021720505790618469301721065564631296452457478373'),
  c1: BigInt('266929791119991161246907387137283842545076965332900288569378510910307636690'),
};

const { Fp, Fp2, Fp6, Fp4Square, Fp12 } = tower12({
  ORDER: BigInt('21888242871839275222246405745257275088696311157297823662689037894645226208583'),
  FP2_NONRESIDUE: [BigInt(9), _1n],
  Fp2mulByB: (num) => Fp2.mul(num, Fp2B),
  // The result of any pairing is in a cyclotomic subgroup
  // https://eprint.iacr.org/2009/565.pdf
  Fp12cyclotomicSquare: ({ c0, c1 }): Fp12 => {
    const { c0: c0c0, c1: c0c1, c2: c0c2 } = c0;
    const { c0: c1c0, c1: c1c1, c2: c1c2 } = c1;
    const { first: t3, second: t4 } = Fp4Square(c0c0, c1c1);
    const { first: t5, second: t6 } = Fp4Square(c1c0, c0c2);
    const { first: t7, second: t8 } = Fp4Square(c0c1, c1c2);
    let t9 = Fp2.mulByNonresidue(t8); // T8 * (u + 1)
    return {
      c0: Fp6.create({
        c0: Fp2.add(Fp2.mul(Fp2.sub(t3, c0c0), _2n), t3), // 2 * (T3 - c0c0)  + T3
        c1: Fp2.add(Fp2.mul(Fp2.sub(t5, c0c1), _2n), t5), // 2 * (T5 - c0c1)  + T5
        c2: Fp2.add(Fp2.mul(Fp2.sub(t7, c0c2), _2n), t7),
      }), // 2 * (T7 - c0c2)  + T7
      c1: Fp6.create({
        c0: Fp2.add(Fp2.mul(Fp2.add(t9, c1c0), _2n), t9), // 2 * (T9 + c1c0) + T9
        c1: Fp2.add(Fp2.mul(Fp2.add(t4, c1c1), _2n), t4), // 2 * (T4 + c1c1) + T4
        c2: Fp2.add(Fp2.mul(Fp2.add(t6, c1c2), _2n), t6),
      }),
    }; // 2 * (T6 + c1c2) + T6
  },
  Fp12cyclotomicExp(num, n) {
    let z = Fp12.ONE;
    for (let i = BN_X_LEN - 1; i >= 0; i--) {
      z = Fp12._cyclotomicSquare(z);
      if (bitGet(n, i)) z = Fp12.mul(z, num);
    }
    return z;
  },
  // https://eprint.iacr.org/2010/354.pdf
  // https://eprint.iacr.org/2009/565.pdf
  Fp12finalExponentiate: (num) => {
    const powMinusX = (num: Fp12) => Fp12.conjugate(Fp12._cyclotomicExp(num, BN_X));
    const r0 = Fp12.mul(Fp12.conjugate(num), Fp12.inv(num));
    const r = Fp12.mul(Fp12.frobeniusMap(r0, 2), r0);
    const y1 = Fp12._cyclotomicSquare(powMinusX(r));
    const y2 = Fp12.mul(Fp12._cyclotomicSquare(y1), y1);
    const y4 = powMinusX(y2);
    const y6 = powMinusX(Fp12._cyclotomicSquare(y4));
    const y8 = Fp12.mul(Fp12.mul(Fp12.conjugate(y6), y4), Fp12.conjugate(y2));
    const y9 = Fp12.mul(y8, y1);
    return Fp12.mul(
      Fp12.frobeniusMap(Fp12.mul(Fp12.conjugate(r), y9), 3),
      Fp12.mul(
        Fp12.frobeniusMap(y8, 2),
        Fp12.mul(Fp12.frobeniusMap(y9, 1), Fp12.mul(Fp12.mul(y8, y4), r))
      )
    );
  },
});

// END OF CURVE FIELDS
const { G2psi, psi } = psiFrobenius(Fp, Fp2, Fp2.NONRESIDUE);

/*
No hashToCurve for now (and signatures):

- RFC 9380 doesn't mention bn254 and doesn't provide test vectors
- Overall seems like nobody is using BLS signatures on top of bn254
- Seems like it can utilize SVDW, which is not implemented yet
*/
const htfDefaults = Object.freeze({
  // DST: a domain separation tag defined in section 2.2.5
  DST: 'BN254G2_XMD:SHA-256_SVDW_RO_',
  encodeDST: 'BN254G2_XMD:SHA-256_SVDW_RO_',
  p: Fp.ORDER,
  m: 2,
  k: 128,
  expand: 'xmd',
  hash: sha256,
} as const);

export const _postPrecompute: PostPrecomputeFn = (
  Rx: Fp2,
  Ry: Fp2,
  Rz: Fp2,
  Qx: Fp2,
  Qy: Fp2,
  pointAdd: PostPrecomputePointAddFn
) => {
  const q = psi(Qx, Qy);
  ({ Rx, Ry, Rz } = pointAdd(Rx, Ry, Rz, q[0], q[1]));
  const q2 = psi(q[0], q[1]);
  pointAdd(Rx, Ry, Rz, q2[0], Fp2.neg(q2[1]));
};

/**
 * bn254 (a.k.a. alt_bn128) pairing-friendly curve.
 * Contains G1 / G2 operations and pairings.
 */
export const bn254: BLSCurveFn = bls({
  // Fields
  fields: { Fp, Fp2, Fp6, Fp12, Fr },
  G1: {
    Fp,
    h: BigInt(1),
    Gx: BigInt(1),
    Gy: BigInt(2),
    a: Fp.ZERO,
    b: _3n,
    htfDefaults: { ...htfDefaults, m: 1, DST: 'BN254G2_XMD:SHA-256_SVDW_RO_' },
    wrapPrivateKey: true,
    allowInfinityPoint: true,
    mapToCurve: notImplemented,
    fromBytes: notImplemented,
    toBytes: notImplemented,
    ShortSignature: {
      fromHex: notImplemented,
      toRawBytes: notImplemented,
      toHex: notImplemented,
    },
  },
  G2: {
    Fp: Fp2,
    // cofactor: (36 * X^4) + (36 * X^3) + (30 * X^2) + 6*X + 1
    h: BigInt('21888242871839275222246405745257275088844257914179612981679871602714643921549'),
    Gx: Fp2.fromBigTuple([
      BigInt('10857046999023057135944570762232829481370756359578518086990519993285655852781'),
      BigInt('11559732032986387107991004021392285783925812861821192530917403151452391805634'),
    ]),
    Gy: Fp2.fromBigTuple([
      BigInt('8495653923123431417604973247489272438418190587263600148770280649306958101930'),
      BigInt('4082367875863433681332203403145435568316851327593401208105741076214120093531'),
    ]),
    a: Fp2.ZERO,
    b: Fp2B,
    hEff: BigInt('21888242871839275222246405745257275088844257914179612981679871602714643921549'),
    htfDefaults: { ...htfDefaults },
    wrapPrivateKey: true,
    allowInfinityPoint: true,
    isTorsionFree: (c, P) => P.multiplyUnsafe(SIX_X_SQUARED).equals(G2psi(c, P)), // [p]P = [6X^2]P
    mapToCurve: notImplemented,
    fromBytes: notImplemented,
    toBytes: notImplemented,
    Signature: {
      fromHex: notImplemented,
      toRawBytes: notImplemented,
      toHex: notImplemented,
    },
  },
  params: {
    ateLoopSize: BN_X * _6n + _2n,
    r: Fr.ORDER,
    xNegative: false,
    twistType: 'divisive',
  },
  htfDefaults,
  hash: sha256,
  randomBytes,

  postPrecompute: _postPrecompute,
});

/**
 * bn254 weierstrass curve with ECDSA.
 * This is very rare and probably not used anywhere.
 * Instead, you should use G1 / G2, defined above.
 */
export const bn254_weierstrass: CurveFn = weierstrass({
  a: BigInt(0),
  b: BigInt(3),
  Fp,
  n: BigInt('21888242871839275222246405745257275088548364400416034343698204186575808495617'),
  Gx: BigInt(1),
  Gy: BigInt(2),
  h: BigInt(1),
  ...getHash(sha256),
});
