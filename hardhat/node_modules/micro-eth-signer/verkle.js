"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.__tests = exports.verifyProof = exports.createProof = exports.serializeCommitment = exports.zeroCommitment = exports.updateCommitment = exports.getTreeKey = exports.getTreeKeyHash = exports.hashCommitments = exports.commitToScalars = exports.hashCommitment = void 0;
exports.verifyExecutionWitnessPreState = verifyExecutionWitnessPreState;
const curve_1 = require("@noble/curves/abstract/curve");
const edwards_1 = require("@noble/curves/abstract/edwards");
const modular_1 = require("@noble/curves/abstract/modular");
const utils_1 = require("@noble/curves/abstract/utils");
const sha256_1 = require("@noble/hashes/sha256");
const utils_2 = require("@noble/hashes/utils");
const P = require("micro-packed");
const utils_ts_1 = require("./utils.js");
const DOMAIN_SIZE = 256;
const DOMAIN_SIZE_LOG2 = Math.log2(DOMAIN_SIZE);
// 256 uses a lot of memory
// 5 - default? 500 ops -> 9k ops for getTreeKey
// 256 - uses a lot of memory (169mb), but 40 -> 6.8k ops for commitToScalars
const MSM_PRECOMPUTE_SMALL = 5;
const MSM_PRECOMPUTE_WINDOW = 8;
const MSM_PRECOMPUTE_2_SIZE = 8;
const TWO_POW_128 = BigInt(2) ** BigInt(128);
const Fp = (0, modular_1.Field)(BigInt('52435875175126190479447740508185965837690552500527637822603658699938581184513'), undefined, true);
const Fr = (0, modular_1.Field)(BigInt('13108968793781547619861935127046491459309155893440570251786403306729687672801'), undefined, true);
const bandersnatch = (0, edwards_1.twistedEdwards)({
    Fp: Fp,
    a: BigInt(-5),
    d: BigInt('45022363124591815672509500913686876175488063829319466900776701791074614335719'),
    n: BigInt('52435875175126190479447740508185965837690552500527637822603658699938581184513'),
    h: BigInt(4),
    Gx: BigInt('18886178867200960497001835917649091219057080094937609519140440539760939937304'),
    Gy: BigInt('19188667384257783945677642223292697773471335439753913231509108946878080696678'),
    hash: sha256_1.sha256,
    randomBytes: utils_2.randomBytes,
});
const { ExtendedPoint: Point } = bandersnatch;
const PolyZero = () => new Array(DOMAIN_SIZE).fill(Fr.ZERO);
function validateIndex(idx) {
    if (!Number.isSafeInteger(idx) || idx < 0 || idx >= DOMAIN_SIZE)
        throw new Error(`wrong index=${idx}`);
}
// Creates 32 byte scalar from smaller u8a
const extendScalar = (b) => {
    if (b.length > 32)
        throw new Error('scalar bytes bigger than 32 bytes');
    const res = new Uint8Array(32);
    res.set(b);
    return res;
};
// == Commitment
const uncompressed = P.apply(P.struct({ x: P.U256LE, y: P.U256LE }), {
    encode: Point.fromAffine,
    decode: (x) => x.toAffine(),
});
const isPositive = (n) => n > Fp.neg(n);
// == serializedCommitment
const compressed = P.apply(P.U256BE, {
    encode: (b) => {
        const x = Fp.create(b);
        const x2 = Fp.sqr(x);
        const dx2 = Fp.sub(Fp.mul(bandersnatch.CURVE.d, x2), Fp.ONE); // dx^2-1
        const ax2 = Fp.sub(Fp.mul(bandersnatch.CURVE.a, x2), Fp.ONE); // ax^2-1
        const y = Fp.sqrt(Fp.div(ax2, dx2)); // sqrt((ax^2-1)/(dx^2-1))
        const yRes = isPositive(y) ? y : Fp.neg(y);
        const p = Point.fromAffine({ x, y: yRes });
        p.assertValidity();
        const t = Fp.sub(Fp.ONE, Fp.mul(bandersnatch.CURVE.a, Fp.sqr(x)));
        const l = (0, modular_1.FpLegendre)(Fp.ORDER)(Fp, t);
        // Check if 1 - ax^2 is a quadratic residue
        if (!Fp.eql(l, Fp.ONE))
            throw new Error('subgroup check failed');
        return p;
    },
    decode: (p) => {
        const affine = p.toAffine();
        return isPositive(affine.y) ? affine.x : Fp.neg(affine.x);
    },
});
function splitHalf(lst) {
    const middle = Math.floor(lst.length / 2);
    return [lst.slice(0, middle), lst.slice(middle)];
}
const multipointProof = P.struct({
    D: compressed,
    cl: P.array(DOMAIN_SIZE_LOG2, compressed),
    cr: P.array(DOMAIN_SIZE_LOG2, compressed),
    a: P.validate(P.U256LE, Fr.create),
});
function generateCRSPoints(seed, points) {
    const res = [];
    const h = sha256_1.sha256.create().update(seed);
    for (let i = 0; res.length < points; i++) {
        const hash = h.clone().update((0, utils_1.numberToBytesBE)(i, 8)).digest();
        const x = Fp.create((0, utils_1.bytesToNumberBE)(hash));
        const xBytes = Fp.toBytes(x);
        xBytes.reverse();
        try {
            res.push(compressed.decode(xBytes));
        }
        catch (e) { }
    }
    return res;
}
// This is pedersen like hashes
const CRS_Q = Point.BASE;
let CRS_G;
let precomputed = false;
let CRS_G_PREC;
let CRS_G0_TREEKEY;
function precomputeOnFirstRun() {
    if (precomputed)
        return;
    CRS_G = generateCRSPoints('eth_verkle_oct_2021', DOMAIN_SIZE);
    for (let i = 0; i < MSM_PRECOMPUTE_SMALL; i++)
        bandersnatch.utils.precompute(MSM_PRECOMPUTE_WINDOW, CRS_G[i]);
    CRS_G_PREC = (0, curve_1.precomputeMSMUnsafe)(Point, Fr, CRS_G, MSM_PRECOMPUTE_2_SIZE);
    CRS_G0_TREEKEY = CRS_G[0].multiplyUnsafe(BigInt(16386));
    precomputed = true;
}
const crsMSM = (scalars) => {
    precomputeOnFirstRun();
    return CRS_G_PREC(scalars);
};
// Transcript
class Transcript {
    constructor(label) {
        this.state = [];
        this.domainSeparator(label);
    }
    domainSeparator(label) {
        this.state.push((0, utils_2.utf8ToBytes)(label));
    }
    appendMessage(message, label) {
        this.domainSeparator(label);
        this.state.push(message);
    }
    appendScalar(label, scalar) {
        this.appendMessage(Fr.toBytes(Fr.create(scalar)), label);
    }
    appendPoint(label, point) {
        this.appendMessage(compressed.encode(point), label);
    }
    challengeScalar(label) {
        this.domainSeparator(label);
        const scalar = Fr.create(Fr.fromBytes((0, sha256_1.sha256)((0, utils_2.concatBytes)(...this.state))));
        this.state = [];
        this.appendScalar(label, scalar);
        return scalar;
    }
}
// /Transcript
function mapToField(p) {
    const { x, y } = p.toAffine();
    return Fr.create(Fp.div(x, y));
}
function getBarycentricWeights(domainSize) {
    const res = [];
    for (let i = 0; i < domainSize; i++) {
        const elm = Fr.create(BigInt(i));
        let weight = Fr.ONE;
        for (let j = 0; j < domainSize; j++) {
            if (j === i)
                continue; // Skip the current domain element
            weight = Fr.mul(weight, Fr.sub(elm, Fr.create(BigInt(j))));
        }
        res.push(weight);
    }
    return res;
}
function getInvertedWeights(domainSize) {
    const res = [];
    for (let i = 1; i < domainSize; i++)
        res.push(Fr.inv(Fr.create(BigInt(i))));
    return res;
}
const WEIGTHS_BARYCENTRIC = getBarycentricWeights(DOMAIN_SIZE);
const WEIGTHS_BARYCENTRIC_INV = Fr.invertBatch(WEIGTHS_BARYCENTRIC);
const WEIGHTS_INVERTED = getInvertedWeights(DOMAIN_SIZE);
const WEIGHTS_INVERTED_NEG = WEIGHTS_INVERTED.map(Fr.neg);
function divideByLinearVanishing(poly, idx) {
    const q = new Array(poly.length).fill(Fr.ZERO);
    const y = poly[idx];
    for (let i = 0; i < poly.length; i++) {
        if (i === idx)
            continue;
        const den = i - idx;
        const isNegative = den < 0;
        const weights = isNegative ? WEIGHTS_INVERTED_NEG : WEIGHTS_INVERTED;
        const denInv = weights[Math.abs(den) - 1];
        const qi = Fr.mul(Fr.sub(poly[i], y), denInv);
        q[i] = qi;
        const weightRatio = Fr.mul(WEIGTHS_BARYCENTRIC[idx], WEIGTHS_BARYCENTRIC_INV[i]);
        q[idx] = Fr.sub(q[idx], Fr.mul(weightRatio, qi));
    }
    return q;
}
function evaluateLagrangeCoefficients(point) {
    const res = [];
    for (let i = 0; i < DOMAIN_SIZE; i++)
        res.push(Fr.mul(WEIGTHS_BARYCENTRIC[i], Fr.sub(point, Fr.create(BigInt(i)))));
    let az = Fr.ONE;
    for (let i = 0; i < DOMAIN_SIZE; i++)
        az = Fr.mul(az, Fr.sub(point, Fr.create(BigInt(i))));
    return Fr.invertBatch(res).map((i) => Fr.mul(i, az));
}
function innerProduct(a, b) {
    let res = Fr.ZERO;
    for (let i = 0; i < a.length; i++)
        res = Fr.add(res, Fr.mul(a[i], b[i]));
    return res;
}
function multiproofR(transcript, queries) {
    transcript.domainSeparator('multiproof');
    for (const q of queries) {
        transcript.appendPoint('C', q.commitment);
        transcript.appendScalar('z', Fr.create(BigInt(q.point)));
        transcript.appendScalar('y', q.result);
    }
    const r = transcript.challengeScalar('r');
    const powers = [Fr.ONE];
    for (let i = 1; i < queries.length; i++)
        powers.push(Fr.mul(powers[i - 1], r));
    return powers;
}
function ipaW(transcript, C, input, output) {
    transcript.domainSeparator('ipa');
    transcript.appendPoint('C', C);
    transcript.appendScalar('input point', input);
    transcript.appendScalar('output point', output);
    return transcript.challengeScalar('w');
}
function ipaX(transcript, L, R) {
    transcript.appendPoint('L', L);
    transcript.appendPoint('R', R);
    return transcript.challengeScalar('x');
}
function multiproofCheck(proof, queries, transcript) {
    const powers = multiproofR(transcript, queries);
    transcript.appendPoint('D', proof.D);
    const t = transcript.challengeScalar('t');
    const g2den = Fr.invertBatch(queries.map((q) => Fr.sub(t, Fr.create(BigInt(q.point)))));
    const helperScalars = [];
    for (let i = 0; i < powers.length; i++)
        helperScalars.push(Fr.mul(powers[i], g2den[i]));
    let g2t = Fr.ZERO;
    for (let i = 0; i < helperScalars.length; i++)
        g2t = Fr.add(g2t, Fr.mul(helperScalars[i], queries[i].result));
    const E = Point.msm(queries.map((q) => q.commitment), helperScalars);
    transcript.appendPoint('E', E);
    const C = E.subtract(proof.D);
    const b = evaluateLagrangeCoefficients(t);
    // IPA
    if (proof.cl.length !== DOMAIN_SIZE_LOG2 || proof.cr.length !== DOMAIN_SIZE_LOG2)
        throw new Error('wrong cl/cr');
    const w = ipaW(transcript, C, t, g2t);
    const challenges = [];
    for (let i = 0; i < DOMAIN_SIZE_LOG2; i++)
        challenges.push(ipaX(transcript, proof.cl[i], proof.cr[i]));
    const challengesInv = Fr.invertBatch(challenges);
    const gi = [];
    const bi = [];
    for (let i = 0; i < DOMAIN_SIZE; i++) {
        let b = Fr.neg(Fr.ONE);
        for (let j = 0; j < DOMAIN_SIZE_LOG2; j++) {
            if ((i >> (DOMAIN_SIZE_LOG2 - j - 1)) & 1)
                b = Fr.mul(b, challengesInv[j]);
        }
        bi.push(b);
        gi.push(Fr.mul(proof.a, b));
    }
    const b0 = innerProduct(b, bi);
    const qi = Fr.mul(w, Fr.add(g2t, Fr.mul(proof.a, b0)));
    // TODO: this is fast only if we have precomputes, otherwise concat is better?
    const tmp = crsMSM(gi);
    const points = proof.cl.concat(proof.cr).concat([C, CRS_Q]);
    const scalars = challenges.concat(challengesInv).concat([Fr.ONE, qi]);
    return Point.msm(points, scalars).add(tmp).equals(Point.ZERO);
}
const scalarMulIndex = (bytes, index) => {
    precomputeOnFirstRun();
    return uncompressed.encode(CRS_G[index].multiplyUnsafe(Fr.fromBytes(bytes)));
};
const hashCommitment = (commitment) => Fr.toBytes(mapToField(uncompressed.decode(commitment)));
exports.hashCommitment = hashCommitment;
const commitToScalars = (vector) => {
    if (vector.length > DOMAIN_SIZE)
        throw new Error('vector length greater than DOMAIN_SIZE');
    const scalars = vector.map(Fr.fromBytes);
    return uncompressed.encode(crsMSM(scalars));
};
exports.commitToScalars = commitToScalars;
// TODO: implement optimization (batch inv inside mapToField)
const hashCommitments = (commitments) => commitments.map(exports.hashCommitment);
exports.hashCommitments = hashCommitments;
const getTreeKeyHash = (address, treeIndexLE) => {
    if (address.length !== 32)
        throw new Error('Address must be 32 bytes');
    if (treeIndexLE.length !== 32)
        throw new Error('Tree index must be 32 bytes');
    precomputeOnFirstRun();
    const P0 = CRS_G[1].multiplyUnsafe((0, utils_1.bytesToNumberLE)(address.subarray(0, 16)));
    const P1 = CRS_G[2].multiplyUnsafe((0, utils_1.bytesToNumberLE)(address.subarray(16, 32)));
    const P2 = CRS_G[3].multiplyUnsafe((0, utils_1.bytesToNumberLE)(treeIndexLE.subarray(0, 16)));
    const P3 = CRS_G[4].multiplyUnsafe((0, utils_1.bytesToNumberLE)(treeIndexLE.subarray(16, 32)));
    const acc = CRS_G0_TREEKEY.add(P0).add(P1).add(P2).add(P3);
    return Fr.toBytes(mapToField(acc));
};
exports.getTreeKeyHash = getTreeKeyHash;
const getTreeKey = (address, treeIndex, subIndex) => {
    const keyHash = (0, exports.getTreeKeyHash)(address, treeIndex);
    keyHash[keyHash.length - 1] = subIndex;
    return keyHash;
};
exports.getTreeKey = getTreeKey;
const updateCommitment = (commitment, commitmentIndex, oldScalarValue, newScalarValue) => {
    const oldCommitment = uncompressed.decode(commitment);
    const delta = Fr.sub(Fr.fromBytes(newScalarValue), Fr.fromBytes(oldScalarValue));
    precomputeOnFirstRun();
    const deltaCommitment = CRS_G[commitmentIndex].multiplyUnsafe(delta);
    return uncompressed.encode(oldCommitment.add(deltaCommitment));
};
exports.updateCommitment = updateCommitment;
exports.zeroCommitment = (0, utils_2.hexToBytes)('00000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000');
const serializeCommitment = (commitment) => compressed.encode(uncompressed.decode(commitment));
exports.serializeCommitment = serializeCommitment;
const createProof = (proverInputs) => {
    const proverQueries = [];
    for (const q of proverInputs) {
        const commitment = compressed.decode(q.serializedCommitment);
        const vector = q.vector.map((i) => {
            const res = Fr.fromBytes(i);
            if (!Fr.isValid(res))
                throw new Error('invalid poly item');
            return res;
        });
        for (const idx of q.indices) {
            validateIndex(idx);
            proverQueries.push({ commitment, poly: vector, point: idx, result: vector[idx] });
        }
    }
    const transcript = new Transcript('verkle');
    const powers = multiproofR(transcript, proverQueries);
    // Aggregate queries
    const aggQueries = {};
    for (let i = 0; i < proverQueries.length; i++) {
        const query = proverQueries[i];
        const point = query.point;
        if (!aggQueries[point])
            aggQueries[point] = PolyZero();
        const res = aggQueries[point];
        for (let j = 0; j < DOMAIN_SIZE; j++)
            res[j] = Fr.add(res[j], Fr.mul(query.poly[j], powers[i]));
    }
    const aggPoints = Object.keys(aggQueries).map((i) => Fr.create(BigInt(i)));
    const gx = PolyZero();
    for (const [point, agg] of Object.entries(aggQueries)) {
        const t = divideByLinearVanishing(agg, Number(point));
        for (let i = 0; i < DOMAIN_SIZE; i++)
            gx[i] = Fr.add(gx[i], t[i]);
    }
    const D = crsMSM(gx);
    transcript.appendPoint('D', D);
    const t = transcript.challengeScalar('t');
    const g1den = Fr.invertBatch(aggPoints.map((i) => Fr.sub(t, i)));
    const g1x = PolyZero();
    const aggPolys = Object.values(aggQueries);
    for (let i = 0; i < aggPolys.length; i++) {
        for (let j = 0; j < DOMAIN_SIZE; j++)
            g1x[j] = Fr.add(g1x[j], Fr.mul(g1den[i], aggPolys[i][j]));
    }
    const E = crsMSM(g1x);
    transcript.appendPoint('E', E);
    const C = E.subtract(D);
    let b = evaluateLagrangeCoefficients(t);
    const g3x = g1x.map((i, j) => Fr.sub(i, gx[j]));
    let a = g3x;
    //
    let G = CRS_G;
    if (a.length !== DOMAIN_SIZE || b.length !== DOMAIN_SIZE)
        throw new Error('Wrong polynominals length');
    // IPA
    const w = ipaW(transcript, C, t, innerProduct(a, b));
    const Q = CRS_Q.multiply(w);
    const cl = [];
    const cr = [];
    for (let _k = 0; _k < DOMAIN_SIZE_LOG2; _k++) {
        const [aL, aR] = splitHalf(a);
        const [bL, bR] = splitHalf(b);
        const [GL, GR] = splitHalf(G);
        const zL = innerProduct(aR, bL);
        const zR = innerProduct(aL, bR);
        const L = Point.msm(GL.concat(Q), aR.concat(zL));
        const R = Point.msm(GR.concat(Q), aL.concat(zR));
        cl.push(L);
        cr.push(R);
        const x = ipaX(transcript, L, R);
        // TODO: batch this?
        const xInv = Fr.inv(x);
        for (let i = 0; i < aL.length; i++) {
            aL[i] = Fr.add(aL[i], Fr.mul(x, aR[i]));
            bL[i] = Fr.add(bL[i], Fr.mul(xInv, bR[i]));
            GL[i] = GL[i].add(GR[i].multiply(xInv));
        }
        a = aL;
        b = bL;
        G = GL;
    }
    return multipointProof.encode({ D, cl, cr, a: a[0] });
};
exports.createProof = createProof;
const verifyProof = (proofBytes, verifierInputs) => {
    const verifierQueries = [];
    for (const i of verifierInputs) {
        const commitment = compressed.decode(i.serializedCommitment);
        for (const { index, value } of i.indexValuePairs)
            verifierQueries.push({ commitment, point: index, result: Fr.fromBytes(value) });
    }
    return multiproofCheck(multipointProof.decode(proofBytes), verifierQueries, new Transcript('verkle'));
};
exports.verifyProof = verifyProof;
const EXTPresent = {
    None: 0,
    DifferentStem: 1,
    Present: 2,
};
function verifyExecutionWitnessPreState(rootHex, executionWitnessJson) {
    let root;
    try {
        root = compressed.decode(utils_ts_1.ethHex.decode(rootHex));
    }
    catch (e) {
        return false;
    }
    const executionWitness = JSON.parse(executionWitnessJson);
    const stateDiffs = executionWitness.stateDiff.map((i) => ({
        stem: utils_ts_1.ethHex.decode(i.stem),
        suffixDiffs: i.suffixDiffs.map((i) => ({
            suffix: i.suffix,
            currentValue: i.currentValue ? utils_ts_1.ethHex.decode(i.currentValue) : undefined,
            newValue: i.newValue ? utils_ts_1.ethHex.decode(i.newValue) : undefined,
        })),
    }));
    const otherStems = executionWitness.verkleProof.otherStems.map(utils_ts_1.ethHex.decode);
    const proof = {
        d: utils_ts_1.ethHex.decode(executionWitness.verkleProof.d),
        cl: executionWitness.verkleProof.ipaProof.cl.map(utils_ts_1.ethHex.decode),
        cr: executionWitness.verkleProof.ipaProof.cr.map(utils_ts_1.ethHex.decode),
        finalEvaluation: utils_ts_1.ethHex.decode(executionWitness.verkleProof.ipaProof.finalEvaluation),
    };
    const depthExtensionPresent = utils_ts_1.ethHex.decode(executionWitness.verkleProof.depthExtensionPresent);
    const depths = [];
    const extensionPresent = [];
    for (const byte of depthExtensionPresent) {
        extensionPresent.push(byte & 3);
        depths.push(byte >> 3);
    }
    const addZ = (key, z) => (0, utils_2.concatBytes)(key, new Uint8Array([z]));
    const keys = [];
    const currentValues = [];
    for (const sd of stateDiffs) {
        const stem = sd.stem;
        for (const diff of sd.suffixDiffs) {
            keys.push(addZ(stem, diff.suffix));
            currentValues.push(diff.currentValue);
        }
    }
    const stemsSet = new Map();
    for (const key of keys) {
        const stem = key.slice(0, 31);
        const stemHex = (0, utils_2.bytesToHex)(stem);
        if (!stemsSet.has(stemHex))
            stemsSet.set(stemHex, stem);
    }
    const stems = Array.from(stemsSet.values());
    const depthsAndExtByStem = new Map();
    const stemsWithExtension = new Map();
    for (let i = 0; i < stems.length; i++) {
        const stem = stems[i];
        const extPres = extensionPresent[i];
        const stemHex = (0, utils_2.bytesToHex)(stem);
        depthsAndExtByStem.set(stemHex, [extPres, depths[i]]);
        if (extPres === EXTPresent.Present)
            stemsWithExtension.set(stemHex, stem);
    }
    const allPathsSet = new Set();
    const allPathsAndZsSet = new Map();
    const leafValuesByPathAndZ = new Map();
    function arraysEqual(a, b) {
        if (a.length !== b.length)
            return false;
        for (let i = 0; i < a.length; ++i)
            if (a[i] !== b[i])
                return false;
        return true;
    }
    const addLeafValuesByPathAndZ = (path, z, value) => {
        leafValuesByPathAndZ.set((0, utils_2.bytesToHex)(addZ(path, z)), value);
    };
    const addAllPathsAndZsSet = (path, z, value) => {
        allPathsSet.add((0, utils_2.bytesToHex)(path));
        allPathsAndZsSet.set((0, utils_2.bytesToHex)(addZ(path, z)), [path, z]);
        if (value !== undefined)
            leafValuesByPathAndZ.set((0, utils_2.bytesToHex)(addZ(path, z)), value);
    };
    for (let i = 0; i < keys.length; i++) {
        const key = keys[i];
        const value = currentValues[i];
        const stem = key.slice(0, 31);
        const stemHex = (0, utils_2.bytesToHex)(stem);
        const [extPres, depth] = depthsAndExtByStem.get(stemHex) || [undefined, undefined];
        if (extPres === undefined || depth === undefined)
            throw new Error(`Stem not found in depths and extensions map`);
        for (let j = 0; j < depth; j++)
            addAllPathsAndZsSet(stem.subarray(0, j), stem[j]);
        if (extPres === EXTPresent.DifferentStem || extPres === EXTPresent.Present) {
            const path = stem.subarray(0, depth);
            addAllPathsAndZsSet(path, 0, Fr.ONE);
            addAllPathsAndZsSet(path, 1);
            if (extPres === EXTPresent.Present) {
                const suffix = key[31];
                const openingIndex = suffix < 128 ? 2 : 3;
                addAllPathsAndZsSet(path, openingIndex);
                addLeafValuesByPathAndZ(path, 1, Fr.fromBytes(extendScalar(stem)));
                const suffixPath = addZ(path, openingIndex);
                const lowIdx = 2 * (suffix % 128);
                addAllPathsAndZsSet(suffixPath, lowIdx, value ? Fr.add(Fr.fromBytes(extendScalar(value.subarray(0, 16))), TWO_POW_128) : Fr.ZERO);
                addAllPathsAndZsSet(suffixPath, lowIdx + 1, value ? Fr.fromBytes(extendScalar(value.subarray(16, 32))) : Fr.ZERO);
            }
            else if (extPres === EXTPresent.DifferentStem) {
                if (value !== undefined)
                    return false;
                let otherStem = undefined;
                const found = [];
                for (const [_, stemValue] of stemsWithExtension) {
                    if (arraysEqual(stemValue.slice(0, depth), stem.slice(0, depth)))
                        found.push(stemValue);
                }
                if (found.length > 1) {
                    throw new Error(`Found more than one instance of stems with extension at depth ${depth}: ${found}`);
                }
                else if (found.length === 1) {
                    otherStem = found[0];
                }
                else {
                    for (const diffStem of otherStems) {
                        if (arraysEqual(diffStem.slice(0, depth), stem.slice(0, depth))) {
                            otherStem = diffStem;
                            break;
                        }
                    }
                    if (!otherStem)
                        throw new Error(`ExtPresent::DifferentStem flag but cannot find the encountered stem`);
                    addLeafValuesByPathAndZ(path, 1, Fr.fromBytes(extendScalar(otherStem)));
                }
            }
        }
        else if (extPres === EXTPresent.None) {
            if (value !== undefined)
                return false;
            addLeafValuesByPathAndZ(depth === 1 ? new Uint8Array() : stem.slice(0, depth), stem[depth - 1], Fr.ZERO);
        }
    }
    // TODO: this seems broken?, we need to sort arrays
    const commitmentsByPath2 = new Map();
    const allPathsArray = Array.from(allPathsSet);
    const commitmentsSortedByPath = [
        root,
        ...executionWitness.verkleProof.commitmentsByPath.map(utils_ts_1.ethHex.decode).map(compressed.decode),
    ];
    if (commitmentsSortedByPath.length !== allPathsArray.length)
        throw new Error('Mismatch between commitments and paths length');
    for (let i = 0; i < allPathsArray.length; i++)
        commitmentsByPath2.set(allPathsArray[i], commitmentsSortedByPath[i]);
    const queries = [];
    for (const [key, [path, z]] of allPathsAndZsSet) {
        const commitment = commitmentsByPath2.get((0, utils_2.bytesToHex)(path)); //without z
        if (!commitment)
            throw new Error('Commitment not found for the given path and z');
        let y = leafValuesByPathAndZ.get(key);
        if (y === undefined) {
            const commitment = commitmentsByPath2.get(key); // with z
            y = commitment ? mapToField(commitment) : Fr.ZERO;
        }
        queries.push({
            path,
            commitment,
            point: z,
            result: y,
        });
    }
    queries.sort((a, b) => {
        const minLength = Math.min(a.path.length, b.path.length);
        for (let i = 0; i < minLength; i++)
            if (a.path[i] !== b.path[i])
                return a.path[i] - b.path[i];
        if (a.path.length !== b.path.length)
            return a.path.length - b.path.length;
        return Number(a.point - b.point);
    });
    const multiproof = {
        cl: proof.cl.map(compressed.decode),
        cr: proof.cr.map(compressed.decode),
        a: (0, utils_1.bytesToNumberBE)(proof.finalEvaluation),
        D: compressed.decode(proof.d),
    };
    return multiproofCheck(multiproof, queries, new Transcript('vt'));
}
// NOTE: for tests only, don't use
exports.__tests = {
    scalarMulIndex,
    WEIGHTS_INVERTED,
    WEIGTHS_BARYCENTRIC,
    WEIGTHS_BARYCENTRIC_INV,
    WEIGHTS_INVERTED_NEG,
    bandersnatch,
    evaluateLagrangeCoefficients,
    divideByLinearVanishing,
    Transcript,
};
//# sourceMappingURL=verkle.js.map