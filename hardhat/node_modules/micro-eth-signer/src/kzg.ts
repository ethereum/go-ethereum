import { bitLen, bytesToNumberBE, numberToBytesBE } from '@noble/curves/abstract/utils';
import { bls12_381 as bls } from '@noble/curves/bls12-381';
import { sha256 } from '@noble/hashes/sha256';
import { bytesToHex, utf8ToBytes } from '@noble/hashes/utils';
import { add0x, hexToNumber, strip0x } from './utils.ts';

/*
KZG for [EIP-4844](https://eips.ethereum.org/EIPS/eip-4844).

Docs:
- https://github.com/ethereum/c-kzg-4844
- https://github.com/ethereum/consensus-specs/blob/dev/specs/deneb/polynomial-commitments.md

TODO(high-level):
- data converted into blob by prepending 0x00 prefix on each chunk and ends with 0x80 terminator
  - Unsure how generic is this
  - There are up to 6 blob per tx
  - Terminator only added to the last blob
- sidecar: {blob, commitment, proof}
- Calculate versionedHash from commitment, which is included inside of tx
- if 'sidecars' inside of tx enabled:
  - envelope turns into 'wrapper'
  - rlp([tx, blobs, commitments, proofs])
  - this means there are two eip4844 txs: with sidecars and without

TODO(EIP7594):
https://eips.ethereum.org/EIPS/eip-7594
compute_cells_and_kzg_proofs(cells, proofs, blob);
recover_cells_and_kzg_proofs(recovered_cells, recovered_proofs, cell_indices, cells, num_cells);
verify_cell_kzg_proof_batch(commitments_bytes, cell_indices, cells, proofs_bytes, num_cells);
*/

const { Fr, Fp12 } = bls.fields;
const G1 = bls.G1.ProjectivePoint;
const G2 = bls.G2.ProjectivePoint;
type G1Point = typeof bls.G1.ProjectivePoint.BASE;
type G2Point = typeof bls.G2.ProjectivePoint.BASE;
type Scalar = string | bigint;
type Blob = string | string[] | bigint[];
const BLOB_REGEX = /.{1,64}/g; // TODO: is this valid?

function parseScalar(s: Scalar): bigint {
  if (typeof s === 'string') {
    s = strip0x(s);
    if (s.length !== 2 * Fr.BYTES) throw new Error('parseScalar: wrong format');
    s = BigInt(`0x${s}`);
  }
  if (!Fr.isValid(s)) throw new Error('parseScalar: invalid field element');
  return s;
}

function formatScalar(n: bigint) {
  return add0x(bytesToHex(numberToBytesBE(n, Fr.BYTES)));
}
function isPowerOfTwo(x: number) {
  return (x & (x - 1)) === 0 && x !== 0;
}

function reverseBits(n: number, bits: number): number {
  let reversed = 0;
  for (let i = 0; i < bits; i++, n >>>= 1) reversed = (reversed << 1) | (n & 1);
  return reversed;
}

// FFTish stuff, reverses bit in index
function bitReversalPermutation<T>(values: T[]): T[] {
  const n = values.length;
  if (n < 2 || !isPowerOfTwo(n))
    throw new Error(`n must be a power of 2 and greater than 1. Got ${n}`);
  const bits = bitLen(BigInt(n)) - 1;
  const res = new Array(n);
  for (let i = 0; i < n; i++) res[reverseBits(i, bits)] = values[i];
  return res;
}

function computeRootsOfUnity(count: number) {
  if (count < 2) throw new Error('expected at least two roots');
  const PRIMITIVE_ROOT_OF_UNITY = 7;
  const order = BigInt(Math.log2(count));
  const power = (Fr.ORDER - BigInt(1)) / BigInt(2) ** order;
  const ROOT = Fr.pow(BigInt(PRIMITIVE_ROOT_OF_UNITY), power);
  const roots = [Fr.ONE, ROOT];
  for (let i = 2; i <= count; i++) {
    roots[i] = Fr.mul(roots[i - 1], ROOT);
    if (Fr.eql(roots[i], Fr.ONE)) break;
  }
  if (!Fr.eql(roots[roots.length - 1], Fr.ONE)) throw new Error('last root should be 1');
  roots.pop();
  if (roots.length !== count) throw new Error('invalid amount of roots');
  return bitReversalPermutation(roots);
}

function pairingVerify(a1: G1Point, a2: G2Point, b1: G1Point, b2: G2Point) {
  // Filter-out points at infinity, because pairingBatch will throw an error
  const pairs = [
    { g1: a1.negate(), g2: a2 },
    { g1: b1, g2: b2 },
  ].filter(({ g1, g2 }) => !G1.ZERO.equals(g1) && !G2.ZERO.equals(g2));
  const f = bls.pairingBatch(pairs, true);
  return Fp12.eql(f, Fp12.ONE);
}

// Official JSON format
export type SetupData = {
  // g1_monomial: string[]; // Not needed until EIP7594 is live
  g1_lagrange: string[];
  g2_monomial: string[];
};

/**
 * KZG from [EIP-4844](https://eips.ethereum.org/EIPS/eip-4844).
 * @example
 * const kzg = new KZG(trustedSetupData);
 */
export class KZG {
  private readonly POLY_NUM: number;
  private readonly G1LB: G1Point[]; // lagrange brp
  private readonly G2M: G2Point[];
  private readonly ROOTS_OF_UNITY: bigint[];
  // Should they be configurable?
  private readonly FIAT_SHAMIR_PROTOCOL_DOMAIN = utf8ToBytes('FSBLOBVERIFY_V1_');
  private readonly RANDOM_CHALLENGE_KZG_BATCH_DOMAIN = utf8ToBytes('RCKZGBATCH___V1_');
  private readonly POLY_NUM_BYTES: Uint8Array;

  constructor(setup: SetupData & { encoding?: 'fast_v1' }) {
    if (setup == null || typeof setup !== 'object') throw new Error('expected valid setup data');
    if (!Array.isArray(setup.g1_lagrange) || !Array.isArray(setup.g2_monomial))
      throw new Error('expected valid setup data');
    // The slowest part
    let fastSetup = false;
    if ('encoding' in setup) {
      fastSetup = setup.encoding === 'fast_v1';
      if (!fastSetup) throw new Error('unknown encoding ' + setup.encoding);
    }
    const G1L = setup.g1_lagrange.map(fastSetup ? this.parseG1Unchecked : this.parseG1);
    this.POLY_NUM = G1L.length;
    this.G2M = setup.g2_monomial.map(fastSetup ? this.parseG2Unchecked : this.parseG2);
    this.G1LB = bitReversalPermutation(G1L);
    this.ROOTS_OF_UNITY = computeRootsOfUnity(this.POLY_NUM);
    this.POLY_NUM_BYTES = numberToBytesBE(this.POLY_NUM, 8);
  }
  // Internal
  private parseG1(p: string | G1Point) {
    if (typeof p === 'string') p = G1.fromHex(strip0x(p));
    return p;
  }
  private parseG1Unchecked(p: string) {
    if (typeof p !== 'string') throw new Error('string expected');
    const [x, y] = p.split(' ').map(hexToNumber);
    return G1.fromAffine({ x, y });
  }
  private parseG2(p: string) {
    return G2.fromHex(strip0x(p));
  }
  private parseG2Unchecked(p: string) {
    const xy = strip0x(p)
      .split(' ')
      .map((c) => c.split(',').map((c) => BigInt('0x' + c))) as unknown as [bigint, bigint][];
    const x = bls.fields.Fp2.fromBigTuple(xy[0]);
    const y = bls.fields.Fp2.fromBigTuple(xy[1]);
    return G2.fromAffine({ x, y });
  }
  private parseBlob(blob: Blob) {
    if (typeof blob === 'string') {
      blob = strip0x(blob);
      if (blob.length !== this.POLY_NUM * Fr.BYTES * 2) throw new Error('Wrong blob length');
      const m = blob.match(BLOB_REGEX);
      if (!m) throw new Error('Wrong blob');
      blob = m;
    }
    return blob.map(parseScalar);
  }
  private invSafe(inverses: bigint[]) {
    inverses = Fr.invertBatch(inverses);
    for (const i of inverses) if (i === undefined) throw new Error('invSafe: division by zero');
    return inverses;
  }
  private G1msm(points: G1Point[], scalars: bigint[]) {
    // Filters zero scalars, non-const time, but improves computeProof up to x93 for empty blobs
    const _points = [];
    const _scalars = [];
    for (let i = 0; i < scalars.length; i++) {
      const s = scalars[i];
      if (Fr.is0(s)) continue;
      _points.push(points[i]);
      _scalars.push(s);
    }
    return G1.msm(_points, _scalars);
  }
  private computeChallenge(blob: bigint[], commitment: G1Point): bigint {
    const h = sha256
      .create()
      .update(this.FIAT_SHAMIR_PROTOCOL_DOMAIN)
      .update(numberToBytesBE(0, 8))
      .update(this.POLY_NUM_BYTES);
    for (const b of blob) h.update(numberToBytesBE(b, Fr.BYTES));
    h.update(commitment.toRawBytes(true));
    const res = Fr.create(bytesToNumberBE(h.digest()));
    h.destroy();
    return res;
  }
  // Evaluate polynominal at the point x
  private evalPoly(poly: bigint[], x: bigint) {
    if (poly.length !== this.POLY_NUM) throw new Error('The polynomial length is incorrect');
    const batch = [];
    for (let i = 0; i < this.POLY_NUM; i++) {
      // This enforces that we don't try inverse of zero here
      if (Fr.eql(x, this.ROOTS_OF_UNITY[i])) return poly[i];
      batch.push(Fr.sub(x, this.ROOTS_OF_UNITY[i]));
    }
    const inverses = this.invSafe(batch);
    let res = Fr.ZERO;
    for (let i = 0; i < this.POLY_NUM; i++)
      res = Fr.add(res, Fr.mul(Fr.mul(inverses[i], this.ROOTS_OF_UNITY[i]), poly[i]));
    res = Fr.div(res, Fr.create(BigInt(this.POLY_NUM)));
    res = Fr.mul(res, Fr.sub(Fr.pow(x, BigInt(this.POLY_NUM)), Fr.ONE));
    return res;
  }
  // Basic
  computeProof(blob: Blob, z: bigint | string): [string, string] {
    z = parseScalar(z);
    blob = this.parseBlob(blob);
    const y = this.evalPoly(blob, z);
    const batch = [];
    let rootOfUnityPos: undefined | number;
    const poly = new Array(this.POLY_NUM).fill(Fr.ZERO);
    for (let i = 0; i < this.POLY_NUM; i++) {
      if (Fr.eql(z, this.ROOTS_OF_UNITY[i])) {
        rootOfUnityPos = i;
        batch.push(Fr.ONE);
        continue;
      }
      poly[i] = Fr.sub(blob[i], y);
      batch.push(Fr.sub(this.ROOTS_OF_UNITY[i], z));
    }
    const inverses = this.invSafe(batch);
    for (let i = 0; i < this.POLY_NUM; i++) poly[i] = Fr.mul(poly[i], inverses[i]);
    if (rootOfUnityPos !== undefined) {
      poly[rootOfUnityPos] = Fr.ZERO;
      for (let i = 0; i < this.POLY_NUM; i++) {
        if (i === rootOfUnityPos) continue;
        batch[i] = Fr.mul(Fr.sub(z, this.ROOTS_OF_UNITY[i]), z);
      }
      const inverses = this.invSafe(batch);
      for (let i = 0; i < this.POLY_NUM; i++) {
        if (i === rootOfUnityPos) continue;
        poly[rootOfUnityPos] = Fr.add(
          poly[rootOfUnityPos],
          Fr.mul(Fr.mul(Fr.sub(blob[i], y), this.ROOTS_OF_UNITY[i]), inverses[i])
        );
      }
    }
    const proof = add0x(this.G1msm(this.G1LB, poly).toHex(true));
    return [proof, formatScalar(y)];
  }
  verifyProof(commitment: string, z: Scalar, y: Scalar, proof: string): boolean {
    try {
      z = parseScalar(z);
      y = parseScalar(y);
      const g2x = Fr.is0(z) ? G2.ZERO : G2.BASE.multiply(z);
      const g1y = Fr.is0(y) ? G1.ZERO : G1.BASE.multiply(y);
      const XminusZ = this.G2M[1].subtract(g2x);
      const PminusY = this.parseG1(commitment).subtract(g1y);
      return pairingVerify(PminusY, G2.BASE, this.parseG1(proof), XminusZ);
    } catch (e) {
      return false;
    }
  }
  // There are no test vectors for this
  private verifyProofBatch(commitments: G1Point[], zs: bigint[], ys: bigint[], proofs: string[]) {
    const n = commitments.length;
    const p: G1Point[] = proofs.map((i) => this.parseG1(i));
    const h = sha256
      .create()
      .update(this.RANDOM_CHALLENGE_KZG_BATCH_DOMAIN)
      .update(this.POLY_NUM_BYTES)
      .update(numberToBytesBE(n, 8));
    for (let i = 0; i < n; i++) {
      h.update(commitments[i].toRawBytes(true));
      h.update(Fr.toBytes(zs[i]));
      h.update(Fr.toBytes(ys[i]));
      h.update(p[i].toRawBytes(true));
    }
    const r = Fr.create(bytesToNumberBE(h.digest()));
    h.destroy();
    const rPowers = [];
    if (n !== 0) {
      rPowers.push(Fr.ONE);
      for (let i = 1; i < n; i++) rPowers[i] = Fr.mul(rPowers[i - 1], r);
    }
    const proofPowers = this.G1msm(p, rPowers);
    const CminusY = commitments.map((c, i) =>
      c.subtract(Fr.is0(ys[i]) ? G1.ZERO : G1.BASE.multiply(ys[i]))
    );
    const RtimesZ = rPowers.map((p, i) => Fr.mul(p, zs[i]));
    const rhs = this.G1msm(p.concat(CminusY), RtimesZ.concat(rPowers));
    return pairingVerify(proofPowers, this.G2M[1], rhs, G2.BASE);
  }
  // Blobs
  blobToKzgCommitment(blob: Blob): string {
    return add0x(this.G1msm(this.G1LB, this.parseBlob(blob)).toHex(true));
  }
  computeBlobProof(blob: Blob, commitment: string): string {
    blob = this.parseBlob(blob);
    const challenge = this.computeChallenge(blob, G1.fromHex(strip0x(commitment)));
    const [proof, _] = this.computeProof(blob, challenge);
    return proof;
  }
  verifyBlobProof(blob: Blob, commitment: string, proof: string): boolean {
    try {
      blob = this.parseBlob(blob);
      const c = G1.fromHex(strip0x(commitment));
      const challenge = this.computeChallenge(blob, c);
      const y = this.evalPoly(blob, challenge);
      return this.verifyProof(commitment, challenge, y, proof);
    } catch (e) {
      return false;
    }
  }
  verifyBlobProofBatch(blobs: string[], commitments: string[], proofs: string[]): boolean {
    if (!Array.isArray(blobs) || !Array.isArray(commitments) || !Array.isArray(proofs))
      throw new Error('invalid arguments');
    if (blobs.length !== commitments.length || blobs.length !== proofs.length) return false;
    if (blobs.length === 1) return this.verifyBlobProof(blobs[0], commitments[0], proofs[0]);
    try {
      const b = blobs.map((i) => this.parseBlob(i));
      const c = commitments.map((i) => G1.fromHex(strip0x(i)));
      const challenges = b.map((b, i) => this.computeChallenge(b, c[i]));
      const ys = b.map((_, i) => this.evalPoly(b[i], challenges[i]));
      return this.verifyProofBatch(c, challenges, ys, proofs);
    } catch (e) {
      return false;
    }
  }
  // High-level method
  // commitmentToVersionedHash(commitment: Uint8Array) {
  //   const VERSION = 1; // Currently only 1 version is supported
  //   // commitment is G1 point in hex?
  //   return concatBytes(new Uint8Array([VERSION]), sha256(commitment));
  // }
}
