/**
 * Short Weierstrass curve methods. The formula is: y² = x³ + ax + b.
 *
 * ### Parameters
 *
 * To initialize a weierstrass curve, one needs to pass following params:
 *
 * * a: formula param
 * * b: formula param
 * * Fp: finite Field over which we'll do calculations. Can be complex (Fp2, Fp12)
 * * n: Curve prime subgroup order, total count of valid points in the field
 * * Gx: Base point (x, y) aka generator point x coordinate
 * * Gy: ...y coordinate
 * * h: cofactor, usually 1. h*n = curve group order (n is only subgroup order)
 * * lowS: whether to enable (default) or disable "low-s" non-malleable signatures
 *
 * ### Design rationale for types
 *
 * * Interaction between classes from different curves should fail:
 *   `k256.Point.BASE.add(p256.Point.BASE)`
 * * For this purpose we want to use `instanceof` operator, which is fast and works during runtime
 * * Different calls of `curve()` would return different classes -
 *   `curve(params) !== curve(params)`: if somebody decided to monkey-patch their curve,
 *   it won't affect others
 *
 * TypeScript can't infer types for classes created inside a function. Classes is one instance
 * of nominative types in TypeScript and interfaces only check for shape, so it's hard to create
 * unique type for every function call.
 *
 * We can use generic types via some param, like curve opts, but that would:
 *     1. Enable interaction between `curve(params)` and `curve(params)` (curves of same params)
 *     which is hard to debug.
 *     2. Params can be generic and we can't enforce them to be constant value:
 *     if somebody creates curve from non-constant params,
 *     it would be allowed to interact with other curves with non-constant params
 *
 * @todo https://www.typescriptlang.org/docs/handbook/release-notes/typescript-2-7.html#unique-symbol
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
// prettier-ignore
import {
  type AffinePoint, type BasicCurve, type Group, type GroupConstructor,
  pippenger, validateBasic, wNAF,
} from './curve.ts';
// prettier-ignore
import {
  Field, type IField, getMinHashLength, invert, mapHashToField, mod, validateField,
} from './modular.ts';
// prettier-ignore
import {
  type CHash, type Hex, type PrivKey,
  aInRange, abool,
  bitMask,
  bytesToHex, bytesToNumberBE, concatBytes, createHmacDrbg, ensureBytes, hexToBytes,
  inRange, isBytes, memoized, numberToBytesBE, numberToHexUnpadded, validateObject
} from './utils.ts';

export type { AffinePoint };
type HmacFnSync = (key: Uint8Array, ...messages: Uint8Array[]) => Uint8Array;
type EndomorphismOpts = {
  beta: bigint;
  splitScalar: (k: bigint) => { k1neg: boolean; k1: bigint; k2neg: boolean; k2: bigint };
};
export type BasicWCurve<T> = BasicCurve<T> & {
  // Params: a, b
  a: T;
  b: T;

  // Optional params
  allowedPrivateKeyLengths?: readonly number[]; // for P521
  wrapPrivateKey?: boolean; // bls12-381 requires mod(n) instead of rejecting keys >= n
  endo?: EndomorphismOpts; // Endomorphism options for Koblitz curves
  // When a cofactor != 1, there can be an effective methods to:
  // 1. Determine whether a point is torsion-free
  isTorsionFree?: (c: ProjConstructor<T>, point: ProjPointType<T>) => boolean;
  // 2. Clear torsion component
  clearCofactor?: (c: ProjConstructor<T>, point: ProjPointType<T>) => ProjPointType<T>;
};

export type Entropy = Hex | boolean;
export type SignOpts = { lowS?: boolean; extraEntropy?: Entropy; prehash?: boolean };
export type VerOpts = { lowS?: boolean; prehash?: boolean; format?: 'compact' | 'der' | undefined };

function validateSigVerOpts(opts: SignOpts | VerOpts) {
  if (opts.lowS !== undefined) abool('lowS', opts.lowS);
  if (opts.prehash !== undefined) abool('prehash', opts.prehash);
}

// Instance for 3d XYZ points
export interface ProjPointType<T> extends Group<ProjPointType<T>> {
  readonly px: T;
  readonly py: T;
  readonly pz: T;
  get x(): T;
  get y(): T;
  multiply(scalar: bigint): ProjPointType<T>;
  toAffine(iz?: T): AffinePoint<T>;
  isTorsionFree(): boolean;
  clearCofactor(): ProjPointType<T>;
  assertValidity(): void;
  hasEvenY(): boolean;
  toRawBytes(isCompressed?: boolean): Uint8Array;
  toHex(isCompressed?: boolean): string;

  multiplyUnsafe(scalar: bigint): ProjPointType<T>;
  multiplyAndAddUnsafe(Q: ProjPointType<T>, a: bigint, b: bigint): ProjPointType<T> | undefined;
  _setWindowSize(windowSize: number): void;
}
// Static methods for 3d XYZ points
export interface ProjConstructor<T> extends GroupConstructor<ProjPointType<T>> {
  new (x: T, y: T, z: T): ProjPointType<T>;
  fromAffine(p: AffinePoint<T>): ProjPointType<T>;
  fromHex(hex: Hex): ProjPointType<T>;
  fromPrivateKey(privateKey: PrivKey): ProjPointType<T>;
  normalizeZ(points: ProjPointType<T>[]): ProjPointType<T>[];
  msm(points: ProjPointType<T>[], scalars: bigint[]): ProjPointType<T>;
}

export type CurvePointsType<T> = BasicWCurve<T> & {
  // Bytes
  fromBytes?: (bytes: Uint8Array) => AffinePoint<T>;
  toBytes?: (c: ProjConstructor<T>, point: ProjPointType<T>, isCompressed: boolean) => Uint8Array;
};

export type CurvePointsTypeWithLength<T> = Readonly<
  CurvePointsType<T> & { nByteLength: number; nBitLength: number }
>;

function validatePointOpts<T>(curve: CurvePointsType<T>): CurvePointsTypeWithLength<T> {
  const opts = validateBasic(curve);
  validateObject(
    opts,
    {
      a: 'field',
      b: 'field',
    },
    {
      allowedPrivateKeyLengths: 'array',
      wrapPrivateKey: 'boolean',
      isTorsionFree: 'function',
      clearCofactor: 'function',
      allowInfinityPoint: 'boolean',
      fromBytes: 'function',
      toBytes: 'function',
    }
  );
  const { endo, Fp, a } = opts;
  if (endo) {
    if (!Fp.eql(a, Fp.ZERO)) {
      throw new Error('invalid endomorphism, can only be defined for Koblitz curves that have a=0');
    }
    if (
      typeof endo !== 'object' ||
      typeof endo.beta !== 'bigint' ||
      typeof endo.splitScalar !== 'function'
    ) {
      throw new Error('invalid endomorphism, expected beta: bigint and splitScalar: function');
    }
  }
  return Object.freeze({ ...opts } as const);
}

export type CurvePointsRes<T> = {
  CURVE: ReturnType<typeof validatePointOpts<T>>;
  ProjectivePoint: ProjConstructor<T>;
  normPrivateKeyToScalar: (key: PrivKey) => bigint;
  weierstrassEquation: (x: T) => T;
  isWithinCurveOrder: (num: bigint) => boolean;
};

export class DERErr extends Error {
  constructor(m = '') {
    super(m);
  }
}
export type IDER = {
  // asn.1 DER encoding utils
  Err: typeof DERErr;
  // Basic building block is TLV (Tag-Length-Value)
  _tlv: {
    encode: (tag: number, data: string) => string;
    // v - value, l - left bytes (unparsed)
    decode(tag: number, data: Uint8Array): { v: Uint8Array; l: Uint8Array };
  };
  // https://crypto.stackexchange.com/a/57734 Leftmost bit of first byte is 'negative' flag,
  // since we always use positive integers here. It must always be empty:
  // - add zero byte if exists
  // - if next byte doesn't have a flag, leading zero is not allowed (minimal encoding)
  _int: {
    encode(num: bigint): string;
    decode(data: Uint8Array): bigint;
  };
  toSig(hex: string | Uint8Array): { r: bigint; s: bigint };
  hexFromSig(sig: { r: bigint; s: bigint }): string;
};
/**
 * ASN.1 DER encoding utilities. ASN is very complex & fragile. Format:
 *
 *     [0x30 (SEQUENCE), bytelength, 0x02 (INTEGER), intLength, R, 0x02 (INTEGER), intLength, S]
 *
 * Docs: https://letsencrypt.org/docs/a-warm-welcome-to-asn1-and-der/, https://luca.ntop.org/Teaching/Appunti/asn1.html
 */
export const DER: IDER = {
  // asn.1 DER encoding utils
  Err: DERErr,
  // Basic building block is TLV (Tag-Length-Value)
  _tlv: {
    encode: (tag: number, data: string): string => {
      const { Err: E } = DER;
      if (tag < 0 || tag > 256) throw new E('tlv.encode: wrong tag');
      if (data.length & 1) throw new E('tlv.encode: unpadded data');
      const dataLen = data.length / 2;
      const len = numberToHexUnpadded(dataLen);
      if ((len.length / 2) & 0b1000_0000) throw new E('tlv.encode: long form length too big');
      // length of length with long form flag
      const lenLen = dataLen > 127 ? numberToHexUnpadded((len.length / 2) | 0b1000_0000) : '';
      const t = numberToHexUnpadded(tag);
      return t + lenLen + len + data;
    },
    // v - value, l - left bytes (unparsed)
    decode(tag: number, data: Uint8Array): { v: Uint8Array; l: Uint8Array } {
      const { Err: E } = DER;
      let pos = 0;
      if (tag < 0 || tag > 256) throw new E('tlv.encode: wrong tag');
      if (data.length < 2 || data[pos++] !== tag) throw new E('tlv.decode: wrong tlv');
      const first = data[pos++];
      const isLong = !!(first & 0b1000_0000); // First bit of first length byte is flag for short/long form
      let length = 0;
      if (!isLong) length = first;
      else {
        // Long form: [longFlag(1bit), lengthLength(7bit), length (BE)]
        const lenLen = first & 0b0111_1111;
        if (!lenLen) throw new E('tlv.decode(long): indefinite length not supported');
        if (lenLen > 4) throw new E('tlv.decode(long): byte length is too big'); // this will overflow u32 in js
        const lengthBytes = data.subarray(pos, pos + lenLen);
        if (lengthBytes.length !== lenLen) throw new E('tlv.decode: length bytes not complete');
        if (lengthBytes[0] === 0) throw new E('tlv.decode(long): zero leftmost byte');
        for (const b of lengthBytes) length = (length << 8) | b;
        pos += lenLen;
        if (length < 128) throw new E('tlv.decode(long): not minimal encoding');
      }
      const v = data.subarray(pos, pos + length);
      if (v.length !== length) throw new E('tlv.decode: wrong value length');
      return { v, l: data.subarray(pos + length) };
    },
  },
  // https://crypto.stackexchange.com/a/57734 Leftmost bit of first byte is 'negative' flag,
  // since we always use positive integers here. It must always be empty:
  // - add zero byte if exists
  // - if next byte doesn't have a flag, leading zero is not allowed (minimal encoding)
  _int: {
    encode(num: bigint): string {
      const { Err: E } = DER;
      if (num < _0n) throw new E('integer: negative integers are not allowed');
      let hex = numberToHexUnpadded(num);
      // Pad with zero byte if negative flag is present
      if (Number.parseInt(hex[0], 16) & 0b1000) hex = '00' + hex;
      if (hex.length & 1) throw new E('unexpected DER parsing assertion: unpadded hex');
      return hex;
    },
    decode(data: Uint8Array): bigint {
      const { Err: E } = DER;
      if (data[0] & 0b1000_0000) throw new E('invalid signature integer: negative');
      if (data[0] === 0x00 && !(data[1] & 0b1000_0000))
        throw new E('invalid signature integer: unnecessary leading zero');
      return bytesToNumberBE(data);
    },
  },
  toSig(hex: string | Uint8Array): { r: bigint; s: bigint } {
    // parse DER signature
    const { Err: E, _int: int, _tlv: tlv } = DER;
    const data = ensureBytes('signature', hex);
    const { v: seqBytes, l: seqLeftBytes } = tlv.decode(0x30, data);
    if (seqLeftBytes.length) throw new E('invalid signature: left bytes after parsing');
    const { v: rBytes, l: rLeftBytes } = tlv.decode(0x02, seqBytes);
    const { v: sBytes, l: sLeftBytes } = tlv.decode(0x02, rLeftBytes);
    if (sLeftBytes.length) throw new E('invalid signature: left bytes after parsing');
    return { r: int.decode(rBytes), s: int.decode(sBytes) };
  },
  hexFromSig(sig: { r: bigint; s: bigint }): string {
    const { _tlv: tlv, _int: int } = DER;
    const rs = tlv.encode(0x02, int.encode(sig.r));
    const ss = tlv.encode(0x02, int.encode(sig.s));
    const seq = rs + ss;
    return tlv.encode(0x30, seq);
  },
};

// Be friendly to bad ECMAScript parsers by not using bigint literals
// prettier-ignore
const _0n = BigInt(0), _1n = BigInt(1), _2n = BigInt(2), _3n = BigInt(3), _4n = BigInt(4);

export function weierstrassPoints<T>(opts: CurvePointsType<T>): CurvePointsRes<T> {
  const CURVE = validatePointOpts(opts);
  const { Fp } = CURVE; // All curves has same field / group length as for now, but they can differ
  const Fn = Field(CURVE.n, CURVE.nBitLength);

  const toBytes =
    CURVE.toBytes ||
    ((_c: ProjConstructor<T>, point: ProjPointType<T>, _isCompressed: boolean) => {
      const a = point.toAffine();
      return concatBytes(Uint8Array.from([0x04]), Fp.toBytes(a.x), Fp.toBytes(a.y));
    });
  const fromBytes =
    CURVE.fromBytes ||
    ((bytes: Uint8Array) => {
      // const head = bytes[0];
      const tail = bytes.subarray(1);
      // if (head !== 0x04) throw new Error('Only non-compressed encoding is supported');
      const x = Fp.fromBytes(tail.subarray(0, Fp.BYTES));
      const y = Fp.fromBytes(tail.subarray(Fp.BYTES, 2 * Fp.BYTES));
      return { x, y };
    });

  /**
   * y² = x³ + ax + b: Short weierstrass curve formula. Takes x, returns y².
   * @returns y²
   */
  function weierstrassEquation(x: T): T {
    const { a, b } = CURVE;
    const x2 = Fp.sqr(x); // x * x
    const x3 = Fp.mul(x2, x); // x2 * x
    return Fp.add(Fp.add(x3, Fp.mul(x, a)), b); // x3 + a * x + b
  }
  // Validate whether the passed curve params are valid.
  // We check if curve equation works for generator point.
  // `assertValidity()` won't work: `isTorsionFree()` is not available at this point in bls12-381.
  // ProjectivePoint class has not been initialized yet.
  if (!Fp.eql(Fp.sqr(CURVE.Gy), weierstrassEquation(CURVE.Gx)))
    throw new Error('bad generator point: equation left != right');

  // Valid group elements reside in range 1..n-1
  function isWithinCurveOrder(num: bigint): boolean {
    return inRange(num, _1n, CURVE.n);
  }
  // Validates if priv key is valid and converts it to bigint.
  // Supports options allowedPrivateKeyLengths and wrapPrivateKey.
  function normPrivateKeyToScalar(key: PrivKey): bigint {
    const { allowedPrivateKeyLengths: lengths, nByteLength, wrapPrivateKey, n: N } = CURVE;
    if (lengths && typeof key !== 'bigint') {
      if (isBytes(key)) key = bytesToHex(key);
      // Normalize to hex string, pad. E.g. P521 would norm 130-132 char hex to 132-char bytes
      if (typeof key !== 'string' || !lengths.includes(key.length))
        throw new Error('invalid private key');
      key = key.padStart(nByteLength * 2, '0');
    }
    let num: bigint;
    try {
      num =
        typeof key === 'bigint'
          ? key
          : bytesToNumberBE(ensureBytes('private key', key, nByteLength));
    } catch (error) {
      throw new Error(
        'invalid private key, expected hex or ' + nByteLength + ' bytes, got ' + typeof key
      );
    }
    if (wrapPrivateKey) num = mod(num, N); // disabled by default, enabled for BLS
    aInRange('private key', num, _1n, N); // num in range [1..N-1]
    return num;
  }

  function aprjpoint(other: unknown) {
    if (!(other instanceof Point)) throw new Error('ProjectivePoint expected');
  }

  // Memoized toAffine / validity check. They are heavy. Points are immutable.

  // Converts Projective point to affine (x, y) coordinates.
  // Can accept precomputed Z^-1 - for example, from invertBatch.
  // (x, y, z) ∋ (x=x/z, y=y/z)
  const toAffineMemo = memoized((p: Point, iz?: T): AffinePoint<T> => {
    const { px: x, py: y, pz: z } = p;
    // Fast-path for normalized points
    if (Fp.eql(z, Fp.ONE)) return { x, y };
    const is0 = p.is0();
    // If invZ was 0, we return zero point. However we still want to execute
    // all operations, so we replace invZ with a random number, 1.
    if (iz == null) iz = is0 ? Fp.ONE : Fp.inv(z);
    const ax = Fp.mul(x, iz);
    const ay = Fp.mul(y, iz);
    const zz = Fp.mul(z, iz);
    if (is0) return { x: Fp.ZERO, y: Fp.ZERO };
    if (!Fp.eql(zz, Fp.ONE)) throw new Error('invZ was invalid');
    return { x: ax, y: ay };
  });
  // NOTE: on exception this will crash 'cached' and no value will be set.
  // Otherwise true will be return
  const assertValidMemo = memoized((p: Point) => {
    if (p.is0()) {
      // (0, 1, 0) aka ZERO is invalid in most contexts.
      // In BLS, ZERO can be serialized, so we allow it.
      // (0, 0, 0) is invalid representation of ZERO.
      if (CURVE.allowInfinityPoint && !Fp.is0(p.py)) return;
      throw new Error('bad point: ZERO');
    }
    // Some 3rd-party test vectors require different wording between here & `fromCompressedHex`
    const { x, y } = p.toAffine();
    // Check if x, y are valid field elements
    if (!Fp.isValid(x) || !Fp.isValid(y)) throw new Error('bad point: x or y not FE');
    const left = Fp.sqr(y); // y²
    const right = weierstrassEquation(x); // x³ + ax + b
    if (!Fp.eql(left, right)) throw new Error('bad point: equation left != right');
    if (!p.isTorsionFree()) throw new Error('bad point: not in prime-order subgroup');
    return true;
  });

  /**
   * Projective Point works in 3d / projective (homogeneous) coordinates: (x, y, z) ∋ (x=x/z, y=y/z)
   * Default Point works in 2d / affine coordinates: (x, y)
   * We're doing calculations in projective, because its operations don't require costly inversion.
   */
  class Point implements ProjPointType<T> {
    static readonly BASE = new Point(CURVE.Gx, CURVE.Gy, Fp.ONE);
    static readonly ZERO = new Point(Fp.ZERO, Fp.ONE, Fp.ZERO);
    readonly px: T;
    readonly py: T;
    readonly pz: T;

    constructor(px: T, py: T, pz: T) {
      if (px == null || !Fp.isValid(px)) throw new Error('x required');
      if (py == null || !Fp.isValid(py)) throw new Error('y required');
      if (pz == null || !Fp.isValid(pz)) throw new Error('z required');
      this.px = px;
      this.py = py;
      this.pz = pz;
      Object.freeze(this);
    }

    // Does not validate if the point is on-curve.
    // Use fromHex instead, or call assertValidity() later.
    static fromAffine(p: AffinePoint<T>): Point {
      const { x, y } = p || {};
      if (!p || !Fp.isValid(x) || !Fp.isValid(y)) throw new Error('invalid affine point');
      if (p instanceof Point) throw new Error('projective point not allowed');
      const is0 = (i: T) => Fp.eql(i, Fp.ZERO);
      // fromAffine(x:0, y:0) would produce (x:0, y:0, z:1), but we need (x:0, y:1, z:0)
      if (is0(x) && is0(y)) return Point.ZERO;
      return new Point(x, y, Fp.ONE);
    }

    get x(): T {
      return this.toAffine().x;
    }
    get y(): T {
      return this.toAffine().y;
    }

    /**
     * Takes a bunch of Projective Points but executes only one
     * inversion on all of them. Inversion is very slow operation,
     * so this improves performance massively.
     * Optimization: converts a list of projective points to a list of identical points with Z=1.
     */
    static normalizeZ(points: Point[]): Point[] {
      const toInv = Fp.invertBatch(points.map((p) => p.pz));
      return points.map((p, i) => p.toAffine(toInv[i])).map(Point.fromAffine);
    }

    /**
     * Converts hash string or Uint8Array to Point.
     * @param hex short/long ECDSA hex
     */
    static fromHex(hex: Hex): Point {
      const P = Point.fromAffine(fromBytes(ensureBytes('pointHex', hex)));
      P.assertValidity();
      return P;
    }

    // Multiplies generator point by privateKey.
    static fromPrivateKey(privateKey: PrivKey) {
      return Point.BASE.multiply(normPrivateKeyToScalar(privateKey));
    }

    // Multiscalar Multiplication
    static msm(points: Point[], scalars: bigint[]): Point {
      return pippenger(Point, Fn, points, scalars);
    }

    // "Private method", don't use it directly
    _setWindowSize(windowSize: number) {
      wnaf.setWindowSize(this, windowSize);
    }

    // A point on curve is valid if it conforms to equation.
    assertValidity(): void {
      assertValidMemo(this);
    }

    hasEvenY(): boolean {
      const { y } = this.toAffine();
      if (Fp.isOdd) return !Fp.isOdd(y);
      throw new Error("Field doesn't support isOdd");
    }

    /**
     * Compare one point to another.
     */
    equals(other: Point): boolean {
      aprjpoint(other);
      const { px: X1, py: Y1, pz: Z1 } = this;
      const { px: X2, py: Y2, pz: Z2 } = other;
      const U1 = Fp.eql(Fp.mul(X1, Z2), Fp.mul(X2, Z1));
      const U2 = Fp.eql(Fp.mul(Y1, Z2), Fp.mul(Y2, Z1));
      return U1 && U2;
    }

    /**
     * Flips point to one corresponding to (x, -y) in Affine coordinates.
     */
    negate(): Point {
      return new Point(this.px, Fp.neg(this.py), this.pz);
    }

    // Renes-Costello-Batina exception-free doubling formula.
    // There is 30% faster Jacobian formula, but it is not complete.
    // https://eprint.iacr.org/2015/1060, algorithm 3
    // Cost: 8M + 3S + 3*a + 2*b3 + 15add.
    double() {
      const { a, b } = CURVE;
      const b3 = Fp.mul(b, _3n);
      const { px: X1, py: Y1, pz: Z1 } = this;
      let X3 = Fp.ZERO, Y3 = Fp.ZERO, Z3 = Fp.ZERO; // prettier-ignore
      let t0 = Fp.mul(X1, X1); // step 1
      let t1 = Fp.mul(Y1, Y1);
      let t2 = Fp.mul(Z1, Z1);
      let t3 = Fp.mul(X1, Y1);
      t3 = Fp.add(t3, t3); // step 5
      Z3 = Fp.mul(X1, Z1);
      Z3 = Fp.add(Z3, Z3);
      X3 = Fp.mul(a, Z3);
      Y3 = Fp.mul(b3, t2);
      Y3 = Fp.add(X3, Y3); // step 10
      X3 = Fp.sub(t1, Y3);
      Y3 = Fp.add(t1, Y3);
      Y3 = Fp.mul(X3, Y3);
      X3 = Fp.mul(t3, X3);
      Z3 = Fp.mul(b3, Z3); // step 15
      t2 = Fp.mul(a, t2);
      t3 = Fp.sub(t0, t2);
      t3 = Fp.mul(a, t3);
      t3 = Fp.add(t3, Z3);
      Z3 = Fp.add(t0, t0); // step 20
      t0 = Fp.add(Z3, t0);
      t0 = Fp.add(t0, t2);
      t0 = Fp.mul(t0, t3);
      Y3 = Fp.add(Y3, t0);
      t2 = Fp.mul(Y1, Z1); // step 25
      t2 = Fp.add(t2, t2);
      t0 = Fp.mul(t2, t3);
      X3 = Fp.sub(X3, t0);
      Z3 = Fp.mul(t2, t1);
      Z3 = Fp.add(Z3, Z3); // step 30
      Z3 = Fp.add(Z3, Z3);
      return new Point(X3, Y3, Z3);
    }

    // Renes-Costello-Batina exception-free addition formula.
    // There is 30% faster Jacobian formula, but it is not complete.
    // https://eprint.iacr.org/2015/1060, algorithm 1
    // Cost: 12M + 0S + 3*a + 3*b3 + 23add.
    add(other: Point): Point {
      aprjpoint(other);
      const { px: X1, py: Y1, pz: Z1 } = this;
      const { px: X2, py: Y2, pz: Z2 } = other;
      let X3 = Fp.ZERO, Y3 = Fp.ZERO, Z3 = Fp.ZERO; // prettier-ignore
      const a = CURVE.a;
      const b3 = Fp.mul(CURVE.b, _3n);
      let t0 = Fp.mul(X1, X2); // step 1
      let t1 = Fp.mul(Y1, Y2);
      let t2 = Fp.mul(Z1, Z2);
      let t3 = Fp.add(X1, Y1);
      let t4 = Fp.add(X2, Y2); // step 5
      t3 = Fp.mul(t3, t4);
      t4 = Fp.add(t0, t1);
      t3 = Fp.sub(t3, t4);
      t4 = Fp.add(X1, Z1);
      let t5 = Fp.add(X2, Z2); // step 10
      t4 = Fp.mul(t4, t5);
      t5 = Fp.add(t0, t2);
      t4 = Fp.sub(t4, t5);
      t5 = Fp.add(Y1, Z1);
      X3 = Fp.add(Y2, Z2); // step 15
      t5 = Fp.mul(t5, X3);
      X3 = Fp.add(t1, t2);
      t5 = Fp.sub(t5, X3);
      Z3 = Fp.mul(a, t4);
      X3 = Fp.mul(b3, t2); // step 20
      Z3 = Fp.add(X3, Z3);
      X3 = Fp.sub(t1, Z3);
      Z3 = Fp.add(t1, Z3);
      Y3 = Fp.mul(X3, Z3);
      t1 = Fp.add(t0, t0); // step 25
      t1 = Fp.add(t1, t0);
      t2 = Fp.mul(a, t2);
      t4 = Fp.mul(b3, t4);
      t1 = Fp.add(t1, t2);
      t2 = Fp.sub(t0, t2); // step 30
      t2 = Fp.mul(a, t2);
      t4 = Fp.add(t4, t2);
      t0 = Fp.mul(t1, t4);
      Y3 = Fp.add(Y3, t0);
      t0 = Fp.mul(t5, t4); // step 35
      X3 = Fp.mul(t3, X3);
      X3 = Fp.sub(X3, t0);
      t0 = Fp.mul(t3, t1);
      Z3 = Fp.mul(t5, Z3);
      Z3 = Fp.add(Z3, t0); // step 40
      return new Point(X3, Y3, Z3);
    }

    subtract(other: Point) {
      return this.add(other.negate());
    }

    is0() {
      return this.equals(Point.ZERO);
    }
    private wNAF(n: bigint): { p: Point; f: Point } {
      return wnaf.wNAFCached(this, n, Point.normalizeZ);
    }

    /**
     * Non-constant-time multiplication. Uses double-and-add algorithm.
     * It's faster, but should only be used when you don't care about
     * an exposed private key e.g. sig verification, which works over *public* keys.
     */
    multiplyUnsafe(sc: bigint): Point {
      const { endo, n: N } = CURVE;
      aInRange('scalar', sc, _0n, N);
      const I = Point.ZERO;
      if (sc === _0n) return I;
      if (this.is0() || sc === _1n) return this;

      // Case a: no endomorphism. Case b: has precomputes.
      if (!endo || wnaf.hasPrecomputes(this))
        return wnaf.wNAFCachedUnsafe(this, sc, Point.normalizeZ);

      // Case c: endomorphism
      let { k1neg, k1, k2neg, k2 } = endo.splitScalar(sc);
      let k1p = I;
      let k2p = I;
      let d: Point = this;
      while (k1 > _0n || k2 > _0n) {
        if (k1 & _1n) k1p = k1p.add(d);
        if (k2 & _1n) k2p = k2p.add(d);
        d = d.double();
        k1 >>= _1n;
        k2 >>= _1n;
      }
      if (k1neg) k1p = k1p.negate();
      if (k2neg) k2p = k2p.negate();
      k2p = new Point(Fp.mul(k2p.px, endo.beta), k2p.py, k2p.pz);
      return k1p.add(k2p);
    }

    /**
     * Constant time multiplication.
     * Uses wNAF method. Windowed method may be 10% faster,
     * but takes 2x longer to generate and consumes 2x memory.
     * Uses precomputes when available.
     * Uses endomorphism for Koblitz curves.
     * @param scalar by which the point would be multiplied
     * @returns New point
     */
    multiply(scalar: bigint): Point {
      const { endo, n: N } = CURVE;
      aInRange('scalar', scalar, _1n, N);
      let point: Point, fake: Point; // Fake point is used to const-time mult
      if (endo) {
        const { k1neg, k1, k2neg, k2 } = endo.splitScalar(scalar);
        let { p: k1p, f: f1p } = this.wNAF(k1);
        let { p: k2p, f: f2p } = this.wNAF(k2);
        k1p = wnaf.constTimeNegate(k1neg, k1p);
        k2p = wnaf.constTimeNegate(k2neg, k2p);
        k2p = new Point(Fp.mul(k2p.px, endo.beta), k2p.py, k2p.pz);
        point = k1p.add(k2p);
        fake = f1p.add(f2p);
      } else {
        const { p, f } = this.wNAF(scalar);
        point = p;
        fake = f;
      }
      // Normalize `z` for both points, but return only real one
      return Point.normalizeZ([point, fake])[0];
    }

    /**
     * Efficiently calculate `aP + bQ`. Unsafe, can expose private key, if used incorrectly.
     * Not using Strauss-Shamir trick: precomputation tables are faster.
     * The trick could be useful if both P and Q are not G (not in our case).
     * @returns non-zero affine point
     */
    multiplyAndAddUnsafe(Q: Point, a: bigint, b: bigint): Point | undefined {
      const G = Point.BASE; // No Strauss-Shamir trick: we have 10% faster G precomputes
      const mul = (
        P: Point,
        a: bigint // Select faster multiply() method
      ) => (a === _0n || a === _1n || !P.equals(G) ? P.multiplyUnsafe(a) : P.multiply(a));
      const sum = mul(this, a).add(mul(Q, b));
      return sum.is0() ? undefined : sum;
    }

    // Converts Projective point to affine (x, y) coordinates.
    // Can accept precomputed Z^-1 - for example, from invertBatch.
    // (x, y, z) ∋ (x=x/z, y=y/z)
    toAffine(iz?: T): AffinePoint<T> {
      return toAffineMemo(this, iz);
    }
    isTorsionFree(): boolean {
      const { h: cofactor, isTorsionFree } = CURVE;
      if (cofactor === _1n) return true; // No subgroups, always torsion-free
      if (isTorsionFree) return isTorsionFree(Point, this);
      throw new Error('isTorsionFree() has not been declared for the elliptic curve');
    }
    clearCofactor(): Point {
      const { h: cofactor, clearCofactor } = CURVE;
      if (cofactor === _1n) return this; // Fast-path
      if (clearCofactor) return clearCofactor(Point, this) as Point;
      return this.multiplyUnsafe(CURVE.h);
    }

    toRawBytes(isCompressed = true): Uint8Array {
      abool('isCompressed', isCompressed);
      this.assertValidity();
      return toBytes(Point, this, isCompressed);
    }

    toHex(isCompressed = true): string {
      abool('isCompressed', isCompressed);
      return bytesToHex(this.toRawBytes(isCompressed));
    }
  }
  const _bits = CURVE.nBitLength;
  const wnaf = wNAF(Point, CURVE.endo ? Math.ceil(_bits / 2) : _bits);
  // Validate if generator point is on curve
  return {
    CURVE,
    ProjectivePoint: Point as ProjConstructor<T>,
    normPrivateKeyToScalar,
    weierstrassEquation,
    isWithinCurveOrder,
  };
}

// Instance
export interface SignatureType {
  readonly r: bigint;
  readonly s: bigint;
  readonly recovery?: number;
  assertValidity(): void;
  addRecoveryBit(recovery: number): RecoveredSignatureType;
  hasHighS(): boolean;
  normalizeS(): SignatureType;
  recoverPublicKey(msgHash: Hex): ProjPointType<bigint>;
  toCompactRawBytes(): Uint8Array;
  toCompactHex(): string;
  // DER-encoded
  toDERRawBytes(isCompressed?: boolean): Uint8Array;
  toDERHex(isCompressed?: boolean): string;
}
export type RecoveredSignatureType = SignatureType & {
  readonly recovery: number;
};
// Static methods
export type SignatureConstructor = {
  new (r: bigint, s: bigint): SignatureType;
  fromCompact(hex: Hex): SignatureType;
  fromDER(hex: Hex): SignatureType;
};
type SignatureLike = { r: bigint; s: bigint };

export type PubKey = Hex | ProjPointType<bigint>;

export type CurveType = BasicWCurve<bigint> & {
  hash: CHash; // CHash not FHash because we need outputLen for DRBG
  hmac: HmacFnSync;
  randomBytes: (bytesLength?: number) => Uint8Array;
  lowS?: boolean;
  bits2int?: (bytes: Uint8Array) => bigint;
  bits2int_modN?: (bytes: Uint8Array) => bigint;
};

function validateOpts(
  curve: CurveType
): Readonly<CurveType & { nByteLength: number; nBitLength: number }> {
  const opts = validateBasic(curve);
  validateObject(
    opts,
    {
      hash: 'hash',
      hmac: 'function',
      randomBytes: 'function',
    },
    {
      bits2int: 'function',
      bits2int_modN: 'function',
      lowS: 'boolean',
    }
  );
  return Object.freeze({ lowS: true, ...opts } as const);
}

export type CurveFn = {
  CURVE: ReturnType<typeof validateOpts>;
  getPublicKey: (privateKey: PrivKey, isCompressed?: boolean) => Uint8Array;
  getSharedSecret: (privateA: PrivKey, publicB: Hex, isCompressed?: boolean) => Uint8Array;
  sign: (msgHash: Hex, privKey: PrivKey, opts?: SignOpts) => RecoveredSignatureType;
  verify: (signature: Hex | SignatureLike, msgHash: Hex, publicKey: Hex, opts?: VerOpts) => boolean;
  ProjectivePoint: ProjConstructor<bigint>;
  Signature: SignatureConstructor;
  utils: {
    normPrivateKeyToScalar: (key: PrivKey) => bigint;
    isValidPrivateKey(privateKey: PrivKey): boolean;
    randomPrivateKey: () => Uint8Array;
    precompute: (windowSize?: number, point?: ProjPointType<bigint>) => ProjPointType<bigint>;
  };
};

/**
 * Creates short weierstrass curve and ECDSA signature methods for it.
 * @example
 * import { Field } from '@noble/curves/abstract/modular';
 * // Before that, define BigInt-s: a, b, p, n, Gx, Gy
 * const curve = weierstrass({ a, b, Fp: Field(p), n, Gx, Gy, h: 1n })
 */
export function weierstrass(curveDef: CurveType): CurveFn {
  const CURVE = validateOpts(curveDef) as ReturnType<typeof validateOpts>;
  const { Fp, n: CURVE_ORDER } = CURVE;
  const compressedLen = Fp.BYTES + 1; // e.g. 33 for 32
  const uncompressedLen = 2 * Fp.BYTES + 1; // e.g. 65 for 32

  function modN(a: bigint) {
    return mod(a, CURVE_ORDER);
  }
  function invN(a: bigint) {
    return invert(a, CURVE_ORDER);
  }

  const {
    ProjectivePoint: Point,
    normPrivateKeyToScalar,
    weierstrassEquation,
    isWithinCurveOrder,
  } = weierstrassPoints({
    ...CURVE,
    toBytes(_c, point, isCompressed: boolean): Uint8Array {
      const a = point.toAffine();
      const x = Fp.toBytes(a.x);
      const cat = concatBytes;
      abool('isCompressed', isCompressed);
      if (isCompressed) {
        return cat(Uint8Array.from([point.hasEvenY() ? 0x02 : 0x03]), x);
      } else {
        return cat(Uint8Array.from([0x04]), x, Fp.toBytes(a.y));
      }
    },
    fromBytes(bytes: Uint8Array) {
      const len = bytes.length;
      const head = bytes[0];
      const tail = bytes.subarray(1);
      // this.assertValidity() is done inside of fromHex
      if (len === compressedLen && (head === 0x02 || head === 0x03)) {
        const x = bytesToNumberBE(tail);
        if (!inRange(x, _1n, Fp.ORDER)) throw new Error('Point is not on curve');
        const y2 = weierstrassEquation(x); // y² = x³ + ax + b
        let y: bigint;
        try {
          y = Fp.sqrt(y2); // y = y² ^ (p+1)/4
        } catch (sqrtError) {
          const suffix = sqrtError instanceof Error ? ': ' + sqrtError.message : '';
          throw new Error('Point is not on curve' + suffix);
        }
        const isYOdd = (y & _1n) === _1n;
        // ECDSA
        const isHeadOdd = (head & 1) === 1;
        if (isHeadOdd !== isYOdd) y = Fp.neg(y);
        return { x, y };
      } else if (len === uncompressedLen && head === 0x04) {
        const x = Fp.fromBytes(tail.subarray(0, Fp.BYTES));
        const y = Fp.fromBytes(tail.subarray(Fp.BYTES, 2 * Fp.BYTES));
        return { x, y };
      } else {
        const cl = compressedLen;
        const ul = uncompressedLen;
        throw new Error(
          'invalid Point, expected length of ' + cl + ', or uncompressed ' + ul + ', got ' + len
        );
      }
    },
  });
  const numToNByteHex = (num: bigint): string =>
    bytesToHex(numberToBytesBE(num, CURVE.nByteLength));

  function isBiggerThanHalfOrder(number: bigint) {
    const HALF = CURVE_ORDER >> _1n;
    return number > HALF;
  }

  function normalizeS(s: bigint) {
    return isBiggerThanHalfOrder(s) ? modN(-s) : s;
  }
  // slice bytes num
  const slcNum = (b: Uint8Array, from: number, to: number) => bytesToNumberBE(b.slice(from, to));

  /**
   * ECDSA signature with its (r, s) properties. Supports DER & compact representations.
   */
  class Signature implements SignatureType {
    readonly r: bigint;
    readonly s: bigint;
    readonly recovery?: number;
    constructor(r: bigint, s: bigint, recovery?: number) {
      aInRange('r', r, _1n, CURVE_ORDER); // r in [1..N]
      aInRange('s', s, _1n, CURVE_ORDER); // s in [1..N]
      this.r = r;
      this.s = s;
      if (recovery != null) this.recovery = recovery;
      Object.freeze(this);
    }

    // pair (bytes of r, bytes of s)
    static fromCompact(hex: Hex) {
      const l = CURVE.nByteLength;
      hex = ensureBytes('compactSignature', hex, l * 2);
      return new Signature(slcNum(hex, 0, l), slcNum(hex, l, 2 * l));
    }

    // DER encoded ECDSA signature
    // https://bitcoin.stackexchange.com/questions/57644/what-are-the-parts-of-a-bitcoin-transaction-input-script
    static fromDER(hex: Hex) {
      const { r, s } = DER.toSig(ensureBytes('DER', hex));
      return new Signature(r, s);
    }

    /**
     * @todo remove
     * @deprecated
     */
    assertValidity(): void {}

    addRecoveryBit(recovery: number): RecoveredSignature {
      return new Signature(this.r, this.s, recovery) as RecoveredSignature;
    }

    recoverPublicKey(msgHash: Hex): typeof Point.BASE {
      const { r, s, recovery: rec } = this;
      const h = bits2int_modN(ensureBytes('msgHash', msgHash)); // Truncate hash
      if (rec == null || ![0, 1, 2, 3].includes(rec)) throw new Error('recovery id invalid');
      const radj = rec === 2 || rec === 3 ? r + CURVE.n : r;
      if (radj >= Fp.ORDER) throw new Error('recovery id 2 or 3 invalid');
      const prefix = (rec & 1) === 0 ? '02' : '03';
      const R = Point.fromHex(prefix + numToNByteHex(radj));
      const ir = invN(radj); // r^-1
      const u1 = modN(-h * ir); // -hr^-1
      const u2 = modN(s * ir); // sr^-1
      const Q = Point.BASE.multiplyAndAddUnsafe(R, u1, u2); // (sr^-1)R-(hr^-1)G = -(hr^-1)G + (sr^-1)
      if (!Q) throw new Error('point at infinify'); // unsafe is fine: no priv data leaked
      Q.assertValidity();
      return Q;
    }

    // Signatures should be low-s, to prevent malleability.
    hasHighS(): boolean {
      return isBiggerThanHalfOrder(this.s);
    }

    normalizeS() {
      return this.hasHighS() ? new Signature(this.r, modN(-this.s), this.recovery) : this;
    }

    // DER-encoded
    toDERRawBytes() {
      return hexToBytes(this.toDERHex());
    }
    toDERHex() {
      return DER.hexFromSig({ r: this.r, s: this.s });
    }

    // padded bytes of r, then padded bytes of s
    toCompactRawBytes() {
      return hexToBytes(this.toCompactHex());
    }
    toCompactHex() {
      return numToNByteHex(this.r) + numToNByteHex(this.s);
    }
  }
  type RecoveredSignature = Signature & { recovery: number };

  const utils = {
    isValidPrivateKey(privateKey: PrivKey) {
      try {
        normPrivateKeyToScalar(privateKey);
        return true;
      } catch (error) {
        return false;
      }
    },
    normPrivateKeyToScalar: normPrivateKeyToScalar,

    /**
     * Produces cryptographically secure private key from random of size
     * (groupLen + ceil(groupLen / 2)) with modulo bias being negligible.
     */
    randomPrivateKey: (): Uint8Array => {
      const length = getMinHashLength(CURVE.n);
      return mapHashToField(CURVE.randomBytes(length), CURVE.n);
    },

    /**
     * Creates precompute table for an arbitrary EC point. Makes point "cached".
     * Allows to massively speed-up `point.multiply(scalar)`.
     * @returns cached point
     * @example
     * const fast = utils.precompute(8, ProjectivePoint.fromHex(someonesPubKey));
     * fast.multiply(privKey); // much faster ECDH now
     */
    precompute(windowSize = 8, point = Point.BASE): typeof Point.BASE {
      point._setWindowSize(windowSize);
      point.multiply(BigInt(3)); // 3 is arbitrary, just need any number here
      return point;
    },
  };

  /**
   * Computes public key for a private key. Checks for validity of the private key.
   * @param privateKey private key
   * @param isCompressed whether to return compact (default), or full key
   * @returns Public key, full when isCompressed=false; short when isCompressed=true
   */
  function getPublicKey(privateKey: PrivKey, isCompressed = true): Uint8Array {
    return Point.fromPrivateKey(privateKey).toRawBytes(isCompressed);
  }

  /**
   * Quick and dirty check for item being public key. Does not validate hex, or being on-curve.
   */
  function isProbPub(item: PrivKey | PubKey): boolean {
    const arr = isBytes(item);
    const str = typeof item === 'string';
    const len = (arr || str) && (item as Hex).length;
    if (arr) return len === compressedLen || len === uncompressedLen;
    if (str) return len === 2 * compressedLen || len === 2 * uncompressedLen;
    if (item instanceof Point) return true;
    return false;
  }

  /**
   * ECDH (Elliptic Curve Diffie Hellman).
   * Computes shared public key from private key and public key.
   * Checks: 1) private key validity 2) shared key is on-curve.
   * Does NOT hash the result.
   * @param privateA private key
   * @param publicB different public key
   * @param isCompressed whether to return compact (default), or full key
   * @returns shared public key
   */
  function getSharedSecret(privateA: PrivKey, publicB: Hex, isCompressed = true): Uint8Array {
    if (isProbPub(privateA)) throw new Error('first arg must be private key');
    if (!isProbPub(publicB)) throw new Error('second arg must be public key');
    const b = Point.fromHex(publicB); // check for being on-curve
    return b.multiply(normPrivateKeyToScalar(privateA)).toRawBytes(isCompressed);
  }

  // RFC6979: ensure ECDSA msg is X bytes and < N. RFC suggests optional truncating via bits2octets.
  // FIPS 186-4 4.6 suggests the leftmost min(nBitLen, outLen) bits, which matches bits2int.
  // bits2int can produce res>N, we can do mod(res, N) since the bitLen is the same.
  // int2octets can't be used; pads small msgs with 0: unacceptatble for trunc as per RFC vectors
  const bits2int =
    CURVE.bits2int ||
    function (bytes: Uint8Array): bigint {
      // Our custom check "just in case"
      if (bytes.length > 8192) throw new Error('input is too large');
      // For curves with nBitLength % 8 !== 0: bits2octets(bits2octets(m)) !== bits2octets(m)
      // for some cases, since bytes.length * 8 is not actual bitLength.
      const num = bytesToNumberBE(bytes); // check for == u8 done here
      const delta = bytes.length * 8 - CURVE.nBitLength; // truncate to nBitLength leftmost bits
      return delta > 0 ? num >> BigInt(delta) : num;
    };
  const bits2int_modN =
    CURVE.bits2int_modN ||
    function (bytes: Uint8Array): bigint {
      return modN(bits2int(bytes)); // can't use bytesToNumberBE here
    };
  // NOTE: pads output with zero as per spec
  const ORDER_MASK = bitMask(CURVE.nBitLength);
  /**
   * Converts to bytes. Checks if num in `[0..ORDER_MASK-1]` e.g.: `[0..2^256-1]`.
   */
  function int2octets(num: bigint): Uint8Array {
    aInRange('num < 2^' + CURVE.nBitLength, num, _0n, ORDER_MASK);
    // works with order, can have different size than numToField!
    return numberToBytesBE(num, CURVE.nByteLength);
  }

  // Steps A, D of RFC6979 3.2
  // Creates RFC6979 seed; converts msg/privKey to numbers.
  // Used only in sign, not in verify.
  // NOTE: we cannot assume here that msgHash has same amount of bytes as curve order,
  // this will be invalid at least for P521. Also it can be bigger for P224 + SHA256
  function prepSig(msgHash: Hex, privateKey: PrivKey, opts = defaultSigOpts) {
    if (['recovered', 'canonical'].some((k) => k in opts))
      throw new Error('sign() legacy options not supported');
    const { hash, randomBytes } = CURVE;
    let { lowS, prehash, extraEntropy: ent } = opts; // generates low-s sigs by default
    if (lowS == null) lowS = true; // RFC6979 3.2: we skip step A, because we already provide hash
    msgHash = ensureBytes('msgHash', msgHash);
    validateSigVerOpts(opts);
    if (prehash) msgHash = ensureBytes('prehashed msgHash', hash(msgHash));

    // We can't later call bits2octets, since nested bits2int is broken for curves
    // with nBitLength % 8 !== 0. Because of that, we unwrap it here as int2octets call.
    // const bits2octets = (bits) => int2octets(bits2int_modN(bits))
    const h1int = bits2int_modN(msgHash);
    const d = normPrivateKeyToScalar(privateKey); // validate private key, convert to bigint
    const seedArgs = [int2octets(d), int2octets(h1int)];
    // extraEntropy. RFC6979 3.6: additional k' (optional).
    if (ent != null && ent !== false) {
      // K = HMAC_K(V || 0x00 || int2octets(x) || bits2octets(h1) || k')
      const e = ent === true ? randomBytes(Fp.BYTES) : ent; // generate random bytes OR pass as-is
      seedArgs.push(ensureBytes('extraEntropy', e)); // check for being bytes
    }
    const seed = concatBytes(...seedArgs); // Step D of RFC6979 3.2
    const m = h1int; // NOTE: no need to call bits2int second time here, it is inside truncateHash!
    // Converts signature params into point w r/s, checks result for validity.
    function k2sig(kBytes: Uint8Array): RecoveredSignature | undefined {
      // RFC 6979 Section 3.2, step 3: k = bits2int(T)
      const k = bits2int(kBytes); // Cannot use fields methods, since it is group element
      if (!isWithinCurveOrder(k)) return; // Important: all mod() calls here must be done over N
      const ik = invN(k); // k^-1 mod n
      const q = Point.BASE.multiply(k).toAffine(); // q = Gk
      const r = modN(q.x); // r = q.x mod n
      if (r === _0n) return;
      // Can use scalar blinding b^-1(bm + bdr) where b ∈ [1,q−1] according to
      // https://tches.iacr.org/index.php/TCHES/article/view/7337/6509. We've decided against it:
      // a) dependency on CSPRNG b) 15% slowdown c) doesn't really help since bigints are not CT
      const s = modN(ik * modN(m + r * d)); // Not using blinding here
      if (s === _0n) return;
      let recovery = (q.x === r ? 0 : 2) | Number(q.y & _1n); // recovery bit (2 or 3, when q.x > n)
      let normS = s;
      if (lowS && isBiggerThanHalfOrder(s)) {
        normS = normalizeS(s); // if lowS was passed, ensure s is always
        recovery ^= 1; // // in the bottom half of N
      }
      return new Signature(r, normS, recovery) as RecoveredSignature; // use normS, not s
    }
    return { seed, k2sig };
  }
  const defaultSigOpts: SignOpts = { lowS: CURVE.lowS, prehash: false };
  const defaultVerOpts: VerOpts = { lowS: CURVE.lowS, prehash: false };

  /**
   * Signs message hash with a private key.
   * ```
   * sign(m, d, k) where
   *   (x, y) = G × k
   *   r = x mod n
   *   s = (m + dr)/k mod n
   * ```
   * @param msgHash NOT message. msg needs to be hashed to `msgHash`, or use `prehash`.
   * @param privKey private key
   * @param opts lowS for non-malleable sigs. extraEntropy for mixing randomness into k. prehash will hash first arg.
   * @returns signature with recovery param
   */
  function sign(msgHash: Hex, privKey: PrivKey, opts = defaultSigOpts): RecoveredSignature {
    const { seed, k2sig } = prepSig(msgHash, privKey, opts); // Steps A, D of RFC6979 3.2.
    const C = CURVE;
    const drbg = createHmacDrbg<RecoveredSignature>(C.hash.outputLen, C.nByteLength, C.hmac);
    return drbg(seed, k2sig); // Steps B, C, D, E, F, G
  }

  // Enable precomputes. Slows down first publicKey computation by 20ms.
  Point.BASE._setWindowSize(8);
  // utils.precompute(8, ProjectivePoint.BASE)

  /**
   * Verifies a signature against message hash and public key.
   * Rejects lowS signatures by default: to override,
   * specify option `{lowS: false}`. Implements section 4.1.4 from https://www.secg.org/sec1-v2.pdf:
   *
   * ```
   * verify(r, s, h, P) where
   *   U1 = hs^-1 mod n
   *   U2 = rs^-1 mod n
   *   R = U1⋅G - U2⋅P
   *   mod(R.x, n) == r
   * ```
   */
  function verify(
    signature: Hex | SignatureLike,
    msgHash: Hex,
    publicKey: Hex,
    opts = defaultVerOpts
  ): boolean {
    const sg = signature;
    msgHash = ensureBytes('msgHash', msgHash);
    publicKey = ensureBytes('publicKey', publicKey);
    const { lowS, prehash, format } = opts;

    // Verify opts, deduce signature format
    validateSigVerOpts(opts);
    if ('strict' in opts) throw new Error('options.strict was renamed to lowS');
    if (format !== undefined && format !== 'compact' && format !== 'der')
      throw new Error('format must be compact or der');
    const isHex = typeof sg === 'string' || isBytes(sg);
    const isObj =
      !isHex &&
      !format &&
      typeof sg === 'object' &&
      sg !== null &&
      typeof sg.r === 'bigint' &&
      typeof sg.s === 'bigint';
    if (!isHex && !isObj)
      throw new Error('invalid signature, expected Uint8Array, hex string or Signature instance');

    let _sig: Signature | undefined = undefined;
    let P: ProjPointType<bigint>;
    try {
      if (isObj) _sig = new Signature(sg.r, sg.s);
      if (isHex) {
        // Signature can be represented in 2 ways: compact (2*nByteLength) & DER (variable-length).
        // Since DER can also be 2*nByteLength bytes, we check for it first.
        try {
          if (format !== 'compact') _sig = Signature.fromDER(sg);
        } catch (derError) {
          if (!(derError instanceof DER.Err)) throw derError;
        }
        if (!_sig && format !== 'der') _sig = Signature.fromCompact(sg);
      }
      P = Point.fromHex(publicKey);
    } catch (error) {
      return false;
    }
    if (!_sig) return false;
    if (lowS && _sig.hasHighS()) return false;
    if (prehash) msgHash = CURVE.hash(msgHash);
    const { r, s } = _sig;
    const h = bits2int_modN(msgHash); // Cannot use fields methods, since it is group element
    const is = invN(s); // s^-1
    const u1 = modN(h * is); // u1 = hs^-1 mod n
    const u2 = modN(r * is); // u2 = rs^-1 mod n
    const R = Point.BASE.multiplyAndAddUnsafe(P, u1, u2)?.toAffine(); // R = u1⋅G + u2⋅P
    if (!R) return false;
    const v = modN(R.x);
    return v === r;
  }
  return {
    CURVE,
    getPublicKey,
    getSharedSecret,
    sign,
    verify,
    ProjectivePoint: Point,
    Signature,
    utils,
  };
}

/**
 * Implementation of the Shallue and van de Woestijne method for any weierstrass curve.
 * TODO: check if there is a way to merge this with uvRatio in Edwards; move to modular.
 * b = True and y = sqrt(u / v) if (u / v) is square in F, and
 * b = False and y = sqrt(Z * (u / v)) otherwise.
 * @param Fp
 * @param Z
 * @returns
 */
export function SWUFpSqrtRatio<T>(
  Fp: IField<T>,
  Z: T
): (u: T, v: T) => { isValid: boolean; value: T } {
  // Generic implementation
  const q = Fp.ORDER;
  let l = _0n;
  for (let o = q - _1n; o % _2n === _0n; o /= _2n) l += _1n;
  const c1 = l; // 1. c1, the largest integer such that 2^c1 divides q - 1.
  // We need 2n ** c1 and 2n ** (c1-1). We can't use **; but we can use <<.
  // 2n ** c1 == 2n << (c1-1)
  const _2n_pow_c1_1 = _2n << (c1 - _1n - _1n);
  const _2n_pow_c1 = _2n_pow_c1_1 * _2n;
  const c2 = (q - _1n) / _2n_pow_c1; // 2. c2 = (q - 1) / (2^c1)  # Integer arithmetic
  const c3 = (c2 - _1n) / _2n; // 3. c3 = (c2 - 1) / 2            # Integer arithmetic
  const c4 = _2n_pow_c1 - _1n; // 4. c4 = 2^c1 - 1                # Integer arithmetic
  const c5 = _2n_pow_c1_1; // 5. c5 = 2^(c1 - 1)                  # Integer arithmetic
  const c6 = Fp.pow(Z, c2); // 6. c6 = Z^c2
  const c7 = Fp.pow(Z, (c2 + _1n) / _2n); // 7. c7 = Z^((c2 + 1) / 2)
  let sqrtRatio = (u: T, v: T): { isValid: boolean; value: T } => {
    let tv1 = c6; // 1. tv1 = c6
    let tv2 = Fp.pow(v, c4); // 2. tv2 = v^c4
    let tv3 = Fp.sqr(tv2); // 3. tv3 = tv2^2
    tv3 = Fp.mul(tv3, v); // 4. tv3 = tv3 * v
    let tv5 = Fp.mul(u, tv3); // 5. tv5 = u * tv3
    tv5 = Fp.pow(tv5, c3); // 6. tv5 = tv5^c3
    tv5 = Fp.mul(tv5, tv2); // 7. tv5 = tv5 * tv2
    tv2 = Fp.mul(tv5, v); // 8. tv2 = tv5 * v
    tv3 = Fp.mul(tv5, u); // 9. tv3 = tv5 * u
    let tv4 = Fp.mul(tv3, tv2); // 10. tv4 = tv3 * tv2
    tv5 = Fp.pow(tv4, c5); // 11. tv5 = tv4^c5
    let isQR = Fp.eql(tv5, Fp.ONE); // 12. isQR = tv5 == 1
    tv2 = Fp.mul(tv3, c7); // 13. tv2 = tv3 * c7
    tv5 = Fp.mul(tv4, tv1); // 14. tv5 = tv4 * tv1
    tv3 = Fp.cmov(tv2, tv3, isQR); // 15. tv3 = CMOV(tv2, tv3, isQR)
    tv4 = Fp.cmov(tv5, tv4, isQR); // 16. tv4 = CMOV(tv5, tv4, isQR)
    // 17. for i in (c1, c1 - 1, ..., 2):
    for (let i = c1; i > _1n; i--) {
      let tv5 = i - _2n; // 18.    tv5 = i - 2
      tv5 = _2n << (tv5 - _1n); // 19.    tv5 = 2^tv5
      let tvv5 = Fp.pow(tv4, tv5); // 20.    tv5 = tv4^tv5
      const e1 = Fp.eql(tvv5, Fp.ONE); // 21.    e1 = tv5 == 1
      tv2 = Fp.mul(tv3, tv1); // 22.    tv2 = tv3 * tv1
      tv1 = Fp.mul(tv1, tv1); // 23.    tv1 = tv1 * tv1
      tvv5 = Fp.mul(tv4, tv1); // 24.    tv5 = tv4 * tv1
      tv3 = Fp.cmov(tv2, tv3, e1); // 25.    tv3 = CMOV(tv2, tv3, e1)
      tv4 = Fp.cmov(tvv5, tv4, e1); // 26.    tv4 = CMOV(tv5, tv4, e1)
    }
    return { isValid: isQR, value: tv3 };
  };
  if (Fp.ORDER % _4n === _3n) {
    // sqrt_ratio_3mod4(u, v)
    const c1 = (Fp.ORDER - _3n) / _4n; // 1. c1 = (q - 3) / 4     # Integer arithmetic
    const c2 = Fp.sqrt(Fp.neg(Z)); // 2. c2 = sqrt(-Z)
    sqrtRatio = (u: T, v: T) => {
      let tv1 = Fp.sqr(v); // 1. tv1 = v^2
      const tv2 = Fp.mul(u, v); // 2. tv2 = u * v
      tv1 = Fp.mul(tv1, tv2); // 3. tv1 = tv1 * tv2
      let y1 = Fp.pow(tv1, c1); // 4. y1 = tv1^c1
      y1 = Fp.mul(y1, tv2); // 5. y1 = y1 * tv2
      const y2 = Fp.mul(y1, c2); // 6. y2 = y1 * c2
      const tv3 = Fp.mul(Fp.sqr(y1), v); // 7. tv3 = y1^2; 8. tv3 = tv3 * v
      const isQR = Fp.eql(tv3, u); // 9. isQR = tv3 == u
      let y = Fp.cmov(y2, y1, isQR); // 10. y = CMOV(y2, y1, isQR)
      return { isValid: isQR, value: y }; // 11. return (isQR, y) isQR ? y : y*c2
    };
  }
  // No curves uses that
  // if (Fp.ORDER % _8n === _5n) // sqrt_ratio_5mod8
  return sqrtRatio;
}
/**
 * Simplified Shallue-van de Woestijne-Ulas Method
 * https://www.rfc-editor.org/rfc/rfc9380#section-6.6.2
 */
export function mapToCurveSimpleSWU<T>(
  Fp: IField<T>,
  opts: {
    A: T;
    B: T;
    Z: T;
  }
): (u: T) => { x: T; y: T } {
  validateField(Fp);
  if (!Fp.isValid(opts.A) || !Fp.isValid(opts.B) || !Fp.isValid(opts.Z))
    throw new Error('mapToCurveSimpleSWU: invalid opts');
  const sqrtRatio = SWUFpSqrtRatio(Fp, opts.Z);
  if (!Fp.isOdd) throw new Error('Fp.isOdd is not implemented!');
  // Input: u, an element of F.
  // Output: (x, y), a point on E.
  return (u: T): { x: T; y: T } => {
    // prettier-ignore
    let tv1, tv2, tv3, tv4, tv5, tv6, x, y;
    tv1 = Fp.sqr(u); // 1.  tv1 = u^2
    tv1 = Fp.mul(tv1, opts.Z); // 2.  tv1 = Z * tv1
    tv2 = Fp.sqr(tv1); // 3.  tv2 = tv1^2
    tv2 = Fp.add(tv2, tv1); // 4.  tv2 = tv2 + tv1
    tv3 = Fp.add(tv2, Fp.ONE); // 5.  tv3 = tv2 + 1
    tv3 = Fp.mul(tv3, opts.B); // 6.  tv3 = B * tv3
    tv4 = Fp.cmov(opts.Z, Fp.neg(tv2), !Fp.eql(tv2, Fp.ZERO)); // 7.  tv4 = CMOV(Z, -tv2, tv2 != 0)
    tv4 = Fp.mul(tv4, opts.A); // 8.  tv4 = A * tv4
    tv2 = Fp.sqr(tv3); // 9.  tv2 = tv3^2
    tv6 = Fp.sqr(tv4); // 10. tv6 = tv4^2
    tv5 = Fp.mul(tv6, opts.A); // 11. tv5 = A * tv6
    tv2 = Fp.add(tv2, tv5); // 12. tv2 = tv2 + tv5
    tv2 = Fp.mul(tv2, tv3); // 13. tv2 = tv2 * tv3
    tv6 = Fp.mul(tv6, tv4); // 14. tv6 = tv6 * tv4
    tv5 = Fp.mul(tv6, opts.B); // 15. tv5 = B * tv6
    tv2 = Fp.add(tv2, tv5); // 16. tv2 = tv2 + tv5
    x = Fp.mul(tv1, tv3); // 17.   x = tv1 * tv3
    const { isValid, value } = sqrtRatio(tv2, tv6); // 18. (is_gx1_square, y1) = sqrt_ratio(tv2, tv6)
    y = Fp.mul(tv1, u); // 19.   y = tv1 * u  -> Z * u^3 * y1
    y = Fp.mul(y, value); // 20.   y = y * y1
    x = Fp.cmov(x, tv3, isValid); // 21.   x = CMOV(x, tv3, is_gx1_square)
    y = Fp.cmov(y, value, isValid); // 22.   y = CMOV(y, y1, is_gx1_square)
    const e1 = Fp.isOdd!(u) === Fp.isOdd!(y); // 23.  e1 = sgn0(u) == sgn0(y)
    y = Fp.cmov(Fp.neg(y), y, e1); // 24.   y = CMOV(-y, y, e1)
    x = Fp.div(x, tv4); // 25.   x = x / tv4
    return { x, y };
  };
}
