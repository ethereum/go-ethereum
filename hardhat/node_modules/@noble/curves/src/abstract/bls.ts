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
// TODO: import { AffinePoint } from './curve.ts';
import {
  type H2CPointConstructor,
  type htfBasicOpts,
  type Opts as HTFOpts,
  type MapToCurve,
  createHasher,
} from './hash-to-curve.ts';
import { type IField, getMinHashLength, mapHashToField } from './modular.ts';
import type { Fp12, Fp12Bls, Fp2, Fp2Bls, Fp6 } from './tower.ts';
import { type CHash, type Hex, type PrivKey, ensureBytes, memoized } from './utils.ts';
import {
  type CurvePointsRes,
  type CurvePointsType,
  type ProjPointType,
  weierstrassPoints,
} from './weierstrass.ts';

type Fp = bigint; // Can be different field?

// prettier-ignore
const _0n = BigInt(0), _1n = BigInt(1), _2n = BigInt(2), _3n = BigInt(3);

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

export type PostPrecomputePointAddFn = (
  Rx: Fp2,
  Ry: Fp2,
  Rz: Fp2,
  Qx: Fp2,
  Qy: Fp2
) => { Rx: Fp2; Ry: Fp2; Rz: Fp2 };
export type PostPrecomputeFn = (
  Rx: Fp2,
  Ry: Fp2,
  Rz: Fp2,
  Qx: Fp2,
  Qy: Fp2,
  pointAdd: PostPrecomputePointAddFn
) => void;
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
    // NOTE: MSB is always ignored and used as marker for length,
    // otherwise leading zeros will be lost.
    // Can be different from 'X' (seed) param!
    ateLoopSize: bigint;
    xNegative: boolean;
    r: bigint;
    twistType: TwistType; // BLS12-381: Multiplicative, BN254: Divisive
  };
  htfDefaults: HTFOpts;
  hash: CHash; // Because we need outputLen for DRBG
  randomBytes: (bytesLength?: number) => Uint8Array;
  // This is super ugly hack for untwist point in BN254 after miller loop
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
  verify: (
    signature: Hex | ProjPointType<Fp2>,
    message: Hex | ProjPointType<Fp2>,
    publicKey: Hex | ProjPointType<Fp>,
    htfOpts?: htfBasicOpts
  ) => boolean;
  verifyShortSignature: (
    signature: Hex | ProjPointType<Fp>,
    message: Hex | ProjPointType<Fp>,
    publicKey: Hex | ProjPointType<Fp2>,
    htfOpts?: htfBasicOpts
  ) => boolean;
  verifyBatch: (
    signature: Hex | ProjPointType<Fp2>,
    messages: (Hex | ProjPointType<Fp2>)[],
    publicKeys: (Hex | ProjPointType<Fp>)[],
    htfOpts?: htfBasicOpts
  ) => boolean;
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
  pairingBatch: (
    pairs: { g1: ProjPointType<Fp>; g2: ProjPointType<Fp2> }[],
    withFinalExponent?: boolean
  ) => Fp12;
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

// Not used with BLS12-381 (no sequential `11` in X). Useful for other curves.
function NAfDecomposition(a: bigint) {
  const res = [];
  // a>1 because of marker bit
  for (; a > _1n; a >>= _1n) {
    if ((a & _1n) === _0n) res.unshift(0);
    else if ((a & _3n) === _3n) {
      res.unshift(-1);
      a += _1n;
    } else res.unshift(1);
  }
  return res;
}

export function bls(CURVE: CurveType): CurveFn {
  // Fields are specific for curve, so for now we'll need to pass them with opts
  const { Fp, Fr, Fp2, Fp6, Fp12 } = CURVE.fields;
  const BLS_X_IS_NEGATIVE = CURVE.params.xNegative;
  const TWIST: TwistType = CURVE.params.twistType;
  // Point on G1 curve: (x, y)
  const G1_ = weierstrassPoints({ n: Fr.ORDER, ...CURVE.G1 });
  const G1 = Object.assign(
    G1_,
    createHasher(G1_.ProjectivePoint, CURVE.G1.mapToCurve, {
      ...CURVE.htfDefaults,
      ...CURVE.G1.htfDefaults,
    })
  );
  // Point on G2 curve (complex numbers): (x₁, x₂+i), (y₁, y₂+i)
  const G2_ = weierstrassPoints({ n: Fr.ORDER, ...CURVE.G2 });
  const G2 = Object.assign(
    G2_,
    createHasher(G2_.ProjectivePoint as H2CPointConstructor<Fp2>, CURVE.G2.mapToCurve, {
      ...CURVE.htfDefaults,
      ...CURVE.G2.htfDefaults,
    })
  );
  type G1 = typeof G1.ProjectivePoint.BASE;
  type G2 = typeof G2.ProjectivePoint.BASE;

  // Applies sparse multiplication as line function
  let lineFunction: (c0: Fp2, c1: Fp2, c2: Fp2, f: Fp12, Px: Fp, Py: Fp) => Fp12;
  if (TWIST === 'multiplicative') {
    lineFunction = (c0: Fp2, c1: Fp2, c2: Fp2, f: Fp12, Px: Fp, Py: Fp) =>
      Fp12.mul014(f, c0, Fp2.mul(c1, Px), Fp2.mul(c2, Py));
  } else if (TWIST === 'divisive') {
    // NOTE: it should be [c0, c1, c2], but we use different order here to reduce complexity of
    // precompute calculations.
    lineFunction = (c0: Fp2, c1: Fp2, c2: Fp2, f: Fp12, Px: Fp, Py: Fp) =>
      Fp12.mul034(f, Fp2.mul(c2, Py), Fp2.mul(c1, Px), c0);
  } else throw new Error('bls: unknown twist type');

  const Fp2div2 = Fp2.div(Fp2.ONE, Fp2.mul(Fp2.ONE, _2n));
  function pointDouble(ell: PrecomputeSingle, Rx: Fp2, Ry: Fp2, Rz: Fp2) {
    const t0 = Fp2.sqr(Ry); // Ry²
    const t1 = Fp2.sqr(Rz); // Rz²
    const t2 = Fp2.mulByB(Fp2.mul(t1, _3n)); // 3 * T1 * B
    const t3 = Fp2.mul(t2, _3n); // 3 * T2
    const t4 = Fp2.sub(Fp2.sub(Fp2.sqr(Fp2.add(Ry, Rz)), t1), t0); // (Ry + Rz)² - T1 - T0
    const c0 = Fp2.sub(t2, t0); // T2 - T0 (i)
    const c1 = Fp2.mul(Fp2.sqr(Rx), _3n); // 3 * Rx²
    const c2 = Fp2.neg(t4); // -T4 (-h)

    ell.push([c0, c1, c2]);

    Rx = Fp2.mul(Fp2.mul(Fp2.mul(Fp2.sub(t0, t3), Rx), Ry), Fp2div2); // ((T0 - T3) * Rx * Ry) / 2
    Ry = Fp2.sub(Fp2.sqr(Fp2.mul(Fp2.add(t0, t3), Fp2div2)), Fp2.mul(Fp2.sqr(t2), _3n)); // ((T0 + T3) / 2)² - 3 * T2²
    Rz = Fp2.mul(t0, t4); // T0 * T4
    return { Rx, Ry, Rz };
  }
  function pointAdd(ell: PrecomputeSingle, Rx: Fp2, Ry: Fp2, Rz: Fp2, Qx: Fp2, Qy: Fp2) {
    // Addition
    const t0 = Fp2.sub(Ry, Fp2.mul(Qy, Rz)); // Ry - Qy * Rz
    const t1 = Fp2.sub(Rx, Fp2.mul(Qx, Rz)); // Rx - Qx * Rz
    const c0 = Fp2.sub(Fp2.mul(t0, Qx), Fp2.mul(t1, Qy)); // T0 * Qx - T1 * Qy == Ry * Qx  - Rx * Qy
    const c1 = Fp2.neg(t0); // -T0 == Qy * Rz - Ry
    const c2 = t1; // == Rx - Qx * Rz

    ell.push([c0, c1, c2]);

    const t2 = Fp2.sqr(t1); // T1²
    const t3 = Fp2.mul(t2, t1); // T2 * T1
    const t4 = Fp2.mul(t2, Rx); // T2 * Rx
    const t5 = Fp2.add(Fp2.sub(t3, Fp2.mul(t4, _2n)), Fp2.mul(Fp2.sqr(t0), Rz)); // T3 - 2 * T4 + T0² * Rz
    Rx = Fp2.mul(t1, t5); // T1 * T5
    Ry = Fp2.sub(Fp2.mul(Fp2.sub(t4, t5), t0), Fp2.mul(t3, Ry)); // (T4 - T5) * T0 - T3 * Ry
    Rz = Fp2.mul(Rz, t3); // Rz * T3
    return { Rx, Ry, Rz };
  }

  // Pre-compute coefficients for sparse multiplication
  // Point addition and point double calculations is reused for coefficients
  // pointAdd happens only if bit set, so wNAF is reasonable. Unfortunately we cannot combine
  // add + double in windowed precomputes here, otherwise it would be single op (since X is static)
  const ATE_NAF = NAfDecomposition(CURVE.params.ateLoopSize);

  const calcPairingPrecomputes = memoized((point: G2) => {
    const p = point;
    const { x, y } = p.toAffine();
    // prettier-ignore
    const Qx = x, Qy = y, negQy = Fp2.neg(y);
    // prettier-ignore
    let Rx = Qx, Ry = Qy, Rz = Fp2.ONE;
    const ell: Precompute = [];
    for (const bit of ATE_NAF) {
      const cur: PrecomputeSingle = [];
      ({ Rx, Ry, Rz } = pointDouble(cur, Rx, Ry, Rz));
      if (bit) ({ Rx, Ry, Rz } = pointAdd(cur, Rx, Ry, Rz, Qx, bit === -1 ? negQy : Qy));
      ell.push(cur);
    }
    if (CURVE.postPrecompute) {
      const last = ell[ell.length - 1];
      CURVE.postPrecompute(Rx, Ry, Rz, Qx, Qy, pointAdd.bind(null, last));
    }
    return ell;
  });

  // Main pairing logic is here. Computes product of miller loops + final exponentiate
  // Applies calculated precomputes
  type MillerInput = [Precompute, Fp, Fp][];
  function millerLoopBatch(pairs: MillerInput, withFinalExponent: boolean = false) {
    let f12 = Fp12.ONE;
    if (pairs.length) {
      const ellLen = pairs[0][0].length;
      for (let i = 0; i < ellLen; i++) {
        f12 = Fp12.sqr(f12); // This allows us to do sqr only one time for all pairings
        // NOTE: we apply multiple pairings in parallel here
        for (const [ell, Px, Py] of pairs) {
          for (const [c0, c1, c2] of ell[i]) f12 = lineFunction(c0, c1, c2, f12, Px, Py);
        }
      }
    }
    if (BLS_X_IS_NEGATIVE) f12 = Fp12.conjugate(f12);
    return withFinalExponent ? Fp12.finalExponentiate(f12) : f12;
  }
  type PairingInput = { g1: G1; g2: G2 };
  // Calculates product of multiple pairings
  // This up to x2 faster than just `map(({g1, g2})=>pairing({g1,g2}))`
  function pairingBatch(pairs: PairingInput[], withFinalExponent: boolean = true) {
    const res: MillerInput = [];
    // This cache precomputed toAffine for all points
    G1.ProjectivePoint.normalizeZ(pairs.map(({ g1 }) => g1));
    G2.ProjectivePoint.normalizeZ(pairs.map(({ g2 }) => g2));
    for (const { g1, g2 } of pairs) {
      if (g1.equals(G1.ProjectivePoint.ZERO) || g2.equals(G2.ProjectivePoint.ZERO))
        throw new Error('pairing is not available for ZERO point');
      // This uses toAffine inside
      g1.assertValidity();
      g2.assertValidity();
      const Qa = g1.toAffine();
      res.push([calcPairingPrecomputes(g2), Qa.x, Qa.y]);
    }
    return millerLoopBatch(res, withFinalExponent);
  }
  // Calculates bilinear pairing
  function pairing(Q: G1, P: G2, withFinalExponent: boolean = true): Fp12 {
    return pairingBatch([{ g1: Q, g2: P }], withFinalExponent);
  }

  const utils = {
    randomPrivateKey: (): Uint8Array => {
      const length = getMinHashLength(Fr.ORDER);
      return mapHashToField(CURVE.randomBytes(length), Fr.ORDER);
    },
    calcPairingPrecomputes,
  };

  const { ShortSignature } = CURVE.G1;
  const { Signature } = CURVE.G2;

  type G1Hex = Hex | G1;
  type G2Hex = Hex | G2;
  function normP1(point: G1Hex): G1 {
    return point instanceof G1.ProjectivePoint ? (point as G1) : G1.ProjectivePoint.fromHex(point);
  }
  function normP1Hash(point: G1Hex, htfOpts?: htfBasicOpts): G1 {
    return point instanceof G1.ProjectivePoint
      ? point
      : (G1.hashToCurve(ensureBytes('point', point), htfOpts) as G1);
  }
  function normP2(point: G2Hex): G2 {
    return point instanceof G2.ProjectivePoint ? point : Signature.fromHex(point);
  }
  function normP2Hash(point: G2Hex, htfOpts?: htfBasicOpts): G2 {
    return point instanceof G2.ProjectivePoint
      ? point
      : (G2.hashToCurve(ensureBytes('point', point), htfOpts) as G2);
  }

  // Multiplies generator (G1) by private key.
  // P = pk x G
  function getPublicKey(privateKey: PrivKey): Uint8Array {
    return G1.ProjectivePoint.fromPrivateKey(privateKey).toRawBytes(true);
  }

  // Multiplies generator (G2) by private key.
  // P = pk x G
  function getPublicKeyForShortSignatures(privateKey: PrivKey): Uint8Array {
    return G2.ProjectivePoint.fromPrivateKey(privateKey).toRawBytes(true);
  }

  // Executes `hashToCurve` on the message and then multiplies the result by private key.
  // S = pk x H(m)
  function sign(message: Hex, privateKey: PrivKey, htfOpts?: htfBasicOpts): Uint8Array;
  function sign(message: G2, privateKey: PrivKey, htfOpts?: htfBasicOpts): G2;
  function sign(message: G2Hex, privateKey: PrivKey, htfOpts?: htfBasicOpts): Uint8Array | G2 {
    const msgPoint = normP2Hash(message, htfOpts);
    msgPoint.assertValidity();
    const sigPoint = msgPoint.multiply(G1.normPrivateKeyToScalar(privateKey));
    if (message instanceof G2.ProjectivePoint) return sigPoint;
    return Signature.toRawBytes(sigPoint);
  }

  function signShortSignature(
    message: Hex,
    privateKey: PrivKey,
    htfOpts?: htfBasicOpts
  ): Uint8Array;
  function signShortSignature(message: G1, privateKey: PrivKey, htfOpts?: htfBasicOpts): G1;
  function signShortSignature(
    message: G1Hex,
    privateKey: PrivKey,
    htfOpts?: htfBasicOpts
  ): Uint8Array | G1 {
    const msgPoint = normP1Hash(message, htfOpts);
    msgPoint.assertValidity();
    const sigPoint = msgPoint.multiply(G1.normPrivateKeyToScalar(privateKey));
    if (message instanceof G1.ProjectivePoint) return sigPoint;
    return ShortSignature.toRawBytes(sigPoint);
  }

  // Checks if pairing of public key & hash is equal to pairing of generator & signature.
  // e(P, H(m)) == e(G, S)
  function verify(
    signature: G2Hex,
    message: G2Hex,
    publicKey: G1Hex,
    htfOpts?: htfBasicOpts
  ): boolean {
    const P = normP1(publicKey);
    const Hm = normP2Hash(message, htfOpts);
    const G = G1.ProjectivePoint.BASE;
    const S = normP2(signature);
    const exp = pairingBatch([
      { g1: P.negate(), g2: Hm }, // ePHM = pairing(P.negate(), Hm, false);
      { g1: G, g2: S }, // eGS = pairing(G, S, false);
    ]);
    return Fp12.eql(exp, Fp12.ONE);
  }

  // Checks if pairing of public key & hash is equal to pairing of generator & signature.
  // e(S, G) == e(H(m), P)
  function verifyShortSignature(
    signature: G1Hex,
    message: G1Hex,
    publicKey: G2Hex,
    htfOpts?: htfBasicOpts
  ): boolean {
    const P = normP2(publicKey);
    const Hm = normP1Hash(message, htfOpts);
    const G = G2.ProjectivePoint.BASE;
    const S = normP1(signature);
    const exp = pairingBatch([
      { g1: Hm, g2: P }, // eHmP = pairing(Hm, P, false);
      { g1: S, g2: G.negate() }, // eSG = pairing(S, G.negate(), false);
    ]);
    return Fp12.eql(exp, Fp12.ONE);
  }

  function aNonEmpty(arr: any[]) {
    if (!Array.isArray(arr) || arr.length === 0) throw new Error('expected non-empty array');
  }

  // Adds a bunch of public key points together.
  // pk1 + pk2 + pk3 = pkA
  function aggregatePublicKeys(publicKeys: Hex[]): Uint8Array;
  function aggregatePublicKeys(publicKeys: G1[]): G1;
  function aggregatePublicKeys(publicKeys: G1Hex[]): Uint8Array | G1 {
    aNonEmpty(publicKeys);
    const agg = publicKeys.map(normP1).reduce((sum, p) => sum.add(p), G1.ProjectivePoint.ZERO);
    const aggAffine = agg; //.toAffine();
    if (publicKeys[0] instanceof G1.ProjectivePoint) {
      aggAffine.assertValidity();
      return aggAffine;
    }
    // toRawBytes ensures point validity
    return aggAffine.toRawBytes(true);
  }

  // Adds a bunch of signature points together.
  function aggregateSignatures(signatures: Hex[]): Uint8Array;
  function aggregateSignatures(signatures: G2[]): G2;
  function aggregateSignatures(signatures: G2Hex[]): Uint8Array | G2 {
    aNonEmpty(signatures);
    const agg = signatures.map(normP2).reduce((sum, s) => sum.add(s), G2.ProjectivePoint.ZERO);
    const aggAffine = agg; //.toAffine();
    if (signatures[0] instanceof G2.ProjectivePoint) {
      aggAffine.assertValidity();
      return aggAffine;
    }
    return Signature.toRawBytes(aggAffine);
  }

  // Adds a bunch of signature points together.
  function aggregateShortSignatures(signatures: Hex[]): Uint8Array;
  function aggregateShortSignatures(signatures: G1[]): G1;
  function aggregateShortSignatures(signatures: G1Hex[]): Uint8Array | G1 {
    aNonEmpty(signatures);
    const agg = signatures.map(normP1).reduce((sum, s) => sum.add(s), G1.ProjectivePoint.ZERO);
    const aggAffine = agg; //.toAffine();
    if (signatures[0] instanceof G1.ProjectivePoint) {
      aggAffine.assertValidity();
      return aggAffine;
    }
    return ShortSignature.toRawBytes(aggAffine);
  }

  // https://ethresear.ch/t/fast-verification-of-multiple-bls-signatures/5407
  // e(G, S) = e(G, SUM(n)(Si)) = MUL(n)(e(G, Si))
  function verifyBatch(
    signature: G2Hex,
    // TODO: maybe `{message: G2Hex, publicKey: G1Hex}[]` instead?
    messages: G2Hex[],
    publicKeys: G1Hex[],
    htfOpts?: htfBasicOpts
  ): boolean {
    aNonEmpty(messages);
    if (publicKeys.length !== messages.length)
      throw new Error('amount of public keys and messages should be equal');
    const sig = normP2(signature);
    const nMessages = messages.map((i) => normP2Hash(i, htfOpts));
    const nPublicKeys = publicKeys.map(normP1);
    // NOTE: this works only for exact same object
    const messagePubKeyMap = new Map<G2, G1[]>();
    for (let i = 0; i < nPublicKeys.length; i++) {
      const pub = nPublicKeys[i];
      const msg = nMessages[i];
      let keys = messagePubKeyMap.get(msg);
      if (keys === undefined) {
        keys = [];
        messagePubKeyMap.set(msg, keys);
      }
      keys.push(pub);
    }
    const paired = [];
    try {
      for (const [msg, keys] of messagePubKeyMap) {
        const groupPublicKey = keys.reduce((acc, msg) => acc.add(msg));
        paired.push({ g1: groupPublicKey, g2: msg });
      }
      paired.push({ g1: G1.ProjectivePoint.BASE.negate(), g2: sig });
      return Fp12.eql(pairingBatch(paired), Fp12.ONE);
    } catch {
      return false;
    }
  }

  G1.ProjectivePoint.BASE._setWindowSize(4);

  return {
    getPublicKey,
    getPublicKeyForShortSignatures,
    sign,
    signShortSignature,
    verify,
    verifyBatch,
    verifyShortSignature,
    aggregatePublicKeys,
    aggregateSignatures,
    aggregateShortSignatures,
    millerLoopBatch,
    pairing,
    pairingBatch,
    G1,
    G2,
    Signature,
    ShortSignature,
    fields: {
      Fr,
      Fp,
      Fp2,
      Fp6,
      Fp12,
    },
    params: {
      ateLoopSize: CURVE.params.ateLoopSize,
      r: CURVE.params.r,
      G1b: CURVE.G1.b,
      G2b: CURVE.G2.b,
    },
    utils,
  };
}
