"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.KZG = void 0;
const utils_1 = require("@noble/curves/abstract/utils");
const bls12_381_1 = require("@noble/curves/bls12-381");
const sha256_1 = require("@noble/hashes/sha256");
const utils_2 = require("@noble/hashes/utils");
const utils_ts_1 = require("./utils.js");
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
const { Fr, Fp12 } = bls12_381_1.bls12_381.fields;
const G1 = bls12_381_1.bls12_381.G1.ProjectivePoint;
const G2 = bls12_381_1.bls12_381.G2.ProjectivePoint;
const BLOB_REGEX = /.{1,64}/g; // TODO: is this valid?
function parseScalar(s) {
    if (typeof s === 'string') {
        s = (0, utils_ts_1.strip0x)(s);
        if (s.length !== 2 * Fr.BYTES)
            throw new Error('parseScalar: wrong format');
        s = BigInt(`0x${s}`);
    }
    if (!Fr.isValid(s))
        throw new Error('parseScalar: invalid field element');
    return s;
}
function formatScalar(n) {
    return (0, utils_ts_1.add0x)((0, utils_2.bytesToHex)((0, utils_1.numberToBytesBE)(n, Fr.BYTES)));
}
function isPowerOfTwo(x) {
    return (x & (x - 1)) === 0 && x !== 0;
}
function reverseBits(n, bits) {
    let reversed = 0;
    for (let i = 0; i < bits; i++, n >>>= 1)
        reversed = (reversed << 1) | (n & 1);
    return reversed;
}
// FFTish stuff, reverses bit in index
function bitReversalPermutation(values) {
    const n = values.length;
    if (n < 2 || !isPowerOfTwo(n))
        throw new Error(`n must be a power of 2 and greater than 1. Got ${n}`);
    const bits = (0, utils_1.bitLen)(BigInt(n)) - 1;
    const res = new Array(n);
    for (let i = 0; i < n; i++)
        res[reverseBits(i, bits)] = values[i];
    return res;
}
function computeRootsOfUnity(count) {
    if (count < 2)
        throw new Error('expected at least two roots');
    const PRIMITIVE_ROOT_OF_UNITY = 7;
    const order = BigInt(Math.log2(count));
    const power = (Fr.ORDER - BigInt(1)) / BigInt(2) ** order;
    const ROOT = Fr.pow(BigInt(PRIMITIVE_ROOT_OF_UNITY), power);
    const roots = [Fr.ONE, ROOT];
    for (let i = 2; i <= count; i++) {
        roots[i] = Fr.mul(roots[i - 1], ROOT);
        if (Fr.eql(roots[i], Fr.ONE))
            break;
    }
    if (!Fr.eql(roots[roots.length - 1], Fr.ONE))
        throw new Error('last root should be 1');
    roots.pop();
    if (roots.length !== count)
        throw new Error('invalid amount of roots');
    return bitReversalPermutation(roots);
}
function pairingVerify(a1, a2, b1, b2) {
    // Filter-out points at infinity, because pairingBatch will throw an error
    const pairs = [
        { g1: a1.negate(), g2: a2 },
        { g1: b1, g2: b2 },
    ].filter(({ g1, g2 }) => !G1.ZERO.equals(g1) && !G2.ZERO.equals(g2));
    const f = bls12_381_1.bls12_381.pairingBatch(pairs, true);
    return Fp12.eql(f, Fp12.ONE);
}
/**
 * KZG from [EIP-4844](https://eips.ethereum.org/EIPS/eip-4844).
 * @example
 * const kzg = new KZG(trustedSetupData);
 */
class KZG {
    constructor(setup) {
        // Should they be configurable?
        this.FIAT_SHAMIR_PROTOCOL_DOMAIN = (0, utils_2.utf8ToBytes)('FSBLOBVERIFY_V1_');
        this.RANDOM_CHALLENGE_KZG_BATCH_DOMAIN = (0, utils_2.utf8ToBytes)('RCKZGBATCH___V1_');
        if (setup == null || typeof setup !== 'object')
            throw new Error('expected valid setup data');
        if (!Array.isArray(setup.g1_lagrange) || !Array.isArray(setup.g2_monomial))
            throw new Error('expected valid setup data');
        // The slowest part
        let fastSetup = false;
        if ('encoding' in setup) {
            fastSetup = setup.encoding === 'fast_v1';
            if (!fastSetup)
                throw new Error('unknown encoding ' + setup.encoding);
        }
        const G1L = setup.g1_lagrange.map(fastSetup ? this.parseG1Unchecked : this.parseG1);
        this.POLY_NUM = G1L.length;
        this.G2M = setup.g2_monomial.map(fastSetup ? this.parseG2Unchecked : this.parseG2);
        this.G1LB = bitReversalPermutation(G1L);
        this.ROOTS_OF_UNITY = computeRootsOfUnity(this.POLY_NUM);
        this.POLY_NUM_BYTES = (0, utils_1.numberToBytesBE)(this.POLY_NUM, 8);
    }
    // Internal
    parseG1(p) {
        if (typeof p === 'string')
            p = G1.fromHex((0, utils_ts_1.strip0x)(p));
        return p;
    }
    parseG1Unchecked(p) {
        if (typeof p !== 'string')
            throw new Error('string expected');
        const [x, y] = p.split(' ').map(utils_ts_1.hexToNumber);
        return G1.fromAffine({ x, y });
    }
    parseG2(p) {
        return G2.fromHex((0, utils_ts_1.strip0x)(p));
    }
    parseG2Unchecked(p) {
        const xy = (0, utils_ts_1.strip0x)(p)
            .split(' ')
            .map((c) => c.split(',').map((c) => BigInt('0x' + c)));
        const x = bls12_381_1.bls12_381.fields.Fp2.fromBigTuple(xy[0]);
        const y = bls12_381_1.bls12_381.fields.Fp2.fromBigTuple(xy[1]);
        return G2.fromAffine({ x, y });
    }
    parseBlob(blob) {
        if (typeof blob === 'string') {
            blob = (0, utils_ts_1.strip0x)(blob);
            if (blob.length !== this.POLY_NUM * Fr.BYTES * 2)
                throw new Error('Wrong blob length');
            const m = blob.match(BLOB_REGEX);
            if (!m)
                throw new Error('Wrong blob');
            blob = m;
        }
        return blob.map(parseScalar);
    }
    invSafe(inverses) {
        inverses = Fr.invertBatch(inverses);
        for (const i of inverses)
            if (i === undefined)
                throw new Error('invSafe: division by zero');
        return inverses;
    }
    G1msm(points, scalars) {
        // Filters zero scalars, non-const time, but improves computeProof up to x93 for empty blobs
        const _points = [];
        const _scalars = [];
        for (let i = 0; i < scalars.length; i++) {
            const s = scalars[i];
            if (Fr.is0(s))
                continue;
            _points.push(points[i]);
            _scalars.push(s);
        }
        return G1.msm(_points, _scalars);
    }
    computeChallenge(blob, commitment) {
        const h = sha256_1.sha256
            .create()
            .update(this.FIAT_SHAMIR_PROTOCOL_DOMAIN)
            .update((0, utils_1.numberToBytesBE)(0, 8))
            .update(this.POLY_NUM_BYTES);
        for (const b of blob)
            h.update((0, utils_1.numberToBytesBE)(b, Fr.BYTES));
        h.update(commitment.toRawBytes(true));
        const res = Fr.create((0, utils_1.bytesToNumberBE)(h.digest()));
        h.destroy();
        return res;
    }
    // Evaluate polynominal at the point x
    evalPoly(poly, x) {
        if (poly.length !== this.POLY_NUM)
            throw new Error('The polynomial length is incorrect');
        const batch = [];
        for (let i = 0; i < this.POLY_NUM; i++) {
            // This enforces that we don't try inverse of zero here
            if (Fr.eql(x, this.ROOTS_OF_UNITY[i]))
                return poly[i];
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
    computeProof(blob, z) {
        z = parseScalar(z);
        blob = this.parseBlob(blob);
        const y = this.evalPoly(blob, z);
        const batch = [];
        let rootOfUnityPos;
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
        for (let i = 0; i < this.POLY_NUM; i++)
            poly[i] = Fr.mul(poly[i], inverses[i]);
        if (rootOfUnityPos !== undefined) {
            poly[rootOfUnityPos] = Fr.ZERO;
            for (let i = 0; i < this.POLY_NUM; i++) {
                if (i === rootOfUnityPos)
                    continue;
                batch[i] = Fr.mul(Fr.sub(z, this.ROOTS_OF_UNITY[i]), z);
            }
            const inverses = this.invSafe(batch);
            for (let i = 0; i < this.POLY_NUM; i++) {
                if (i === rootOfUnityPos)
                    continue;
                poly[rootOfUnityPos] = Fr.add(poly[rootOfUnityPos], Fr.mul(Fr.mul(Fr.sub(blob[i], y), this.ROOTS_OF_UNITY[i]), inverses[i]));
            }
        }
        const proof = (0, utils_ts_1.add0x)(this.G1msm(this.G1LB, poly).toHex(true));
        return [proof, formatScalar(y)];
    }
    verifyProof(commitment, z, y, proof) {
        try {
            z = parseScalar(z);
            y = parseScalar(y);
            const g2x = Fr.is0(z) ? G2.ZERO : G2.BASE.multiply(z);
            const g1y = Fr.is0(y) ? G1.ZERO : G1.BASE.multiply(y);
            const XminusZ = this.G2M[1].subtract(g2x);
            const PminusY = this.parseG1(commitment).subtract(g1y);
            return pairingVerify(PminusY, G2.BASE, this.parseG1(proof), XminusZ);
        }
        catch (e) {
            return false;
        }
    }
    // There are no test vectors for this
    verifyProofBatch(commitments, zs, ys, proofs) {
        const n = commitments.length;
        const p = proofs.map((i) => this.parseG1(i));
        const h = sha256_1.sha256
            .create()
            .update(this.RANDOM_CHALLENGE_KZG_BATCH_DOMAIN)
            .update(this.POLY_NUM_BYTES)
            .update((0, utils_1.numberToBytesBE)(n, 8));
        for (let i = 0; i < n; i++) {
            h.update(commitments[i].toRawBytes(true));
            h.update(Fr.toBytes(zs[i]));
            h.update(Fr.toBytes(ys[i]));
            h.update(p[i].toRawBytes(true));
        }
        const r = Fr.create((0, utils_1.bytesToNumberBE)(h.digest()));
        h.destroy();
        const rPowers = [];
        if (n !== 0) {
            rPowers.push(Fr.ONE);
            for (let i = 1; i < n; i++)
                rPowers[i] = Fr.mul(rPowers[i - 1], r);
        }
        const proofPowers = this.G1msm(p, rPowers);
        const CminusY = commitments.map((c, i) => c.subtract(Fr.is0(ys[i]) ? G1.ZERO : G1.BASE.multiply(ys[i])));
        const RtimesZ = rPowers.map((p, i) => Fr.mul(p, zs[i]));
        const rhs = this.G1msm(p.concat(CminusY), RtimesZ.concat(rPowers));
        return pairingVerify(proofPowers, this.G2M[1], rhs, G2.BASE);
    }
    // Blobs
    blobToKzgCommitment(blob) {
        return (0, utils_ts_1.add0x)(this.G1msm(this.G1LB, this.parseBlob(blob)).toHex(true));
    }
    computeBlobProof(blob, commitment) {
        blob = this.parseBlob(blob);
        const challenge = this.computeChallenge(blob, G1.fromHex((0, utils_ts_1.strip0x)(commitment)));
        const [proof, _] = this.computeProof(blob, challenge);
        return proof;
    }
    verifyBlobProof(blob, commitment, proof) {
        try {
            blob = this.parseBlob(blob);
            const c = G1.fromHex((0, utils_ts_1.strip0x)(commitment));
            const challenge = this.computeChallenge(blob, c);
            const y = this.evalPoly(blob, challenge);
            return this.verifyProof(commitment, challenge, y, proof);
        }
        catch (e) {
            return false;
        }
    }
    verifyBlobProofBatch(blobs, commitments, proofs) {
        if (!Array.isArray(blobs) || !Array.isArray(commitments) || !Array.isArray(proofs))
            throw new Error('invalid arguments');
        if (blobs.length !== commitments.length || blobs.length !== proofs.length)
            return false;
        if (blobs.length === 1)
            return this.verifyBlobProof(blobs[0], commitments[0], proofs[0]);
        try {
            const b = blobs.map((i) => this.parseBlob(i));
            const c = commitments.map((i) => G1.fromHex((0, utils_ts_1.strip0x)(i)));
            const challenges = b.map((b, i) => this.computeChallenge(b, c[i]));
            const ys = b.map((_, i) => this.evalPoly(b[i], challenges[i]));
            return this.verifyProofBatch(c, challenges, ys, proofs);
        }
        catch (e) {
            return false;
        }
    }
}
exports.KZG = KZG;
//# sourceMappingURL=kzg.js.map