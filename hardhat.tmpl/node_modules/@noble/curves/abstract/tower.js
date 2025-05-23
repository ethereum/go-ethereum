"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.psiFrobenius = psiFrobenius;
exports.tower12 = tower12;
/**
 * Towered extension fields.
 * Rather than implementing a massive 12th-degree extension directly, it is more efficient
 * to build it up from smaller extensions: a tower of extensions.
 *
 * For BLS12-381, the Fp12 field is implemented as a quadratic (degree two) extension,
 * on top of a cubic (degree three) extension, on top of a quadratic extension of Fp.
 *
 * For more info: "Pairings for beginners" by Costello, section 7.3.
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
const mod = require("./modular.js");
const utils_ts_1 = require("./utils.js");
// Be friendly to bad ECMAScript parsers by not using bigint literals
// prettier-ignore
const _0n = BigInt(0), _1n = BigInt(1), _2n = BigInt(2), _3n = BigInt(3);
function calcFrobeniusCoefficients(Fp, nonResidue, modulus, degree, num = 1, divisor) {
    const _divisor = BigInt(divisor === undefined ? degree : divisor);
    const towerModulus = modulus ** BigInt(degree);
    const res = [];
    for (let i = 0; i < num; i++) {
        const a = BigInt(i + 1);
        const powers = [];
        for (let j = 0, qPower = _1n; j < degree; j++) {
            const power = ((a * qPower - a) / _divisor) % towerModulus;
            powers.push(Fp.pow(nonResidue, power));
            qPower *= modulus;
        }
        res.push(powers);
    }
    return res;
}
// This works same at least for bls12-381, bn254 and bls12-377
function psiFrobenius(Fp, Fp2, base) {
    // Ψ endomorphism
    const PSI_X = Fp2.pow(base, (Fp.ORDER - _1n) / _3n); // u^((p-1)/3)
    const PSI_Y = Fp2.pow(base, (Fp.ORDER - _1n) / _2n); // u^((p-1)/2)
    function psi(x, y) {
        // This x10 faster than previous version in bls12-381
        const x2 = Fp2.mul(Fp2.frobeniusMap(x, 1), PSI_X);
        const y2 = Fp2.mul(Fp2.frobeniusMap(y, 1), PSI_Y);
        return [x2, y2];
    }
    // Ψ²(P) endomorphism (psi2(x) = psi(psi(x)))
    const PSI2_X = Fp2.pow(base, (Fp.ORDER ** _2n - _1n) / _3n); // u^((p^2 - 1)/3)
    // This equals -1, which causes y to be Fp2.neg(y).
    // But not sure if there are case when this is not true?
    const PSI2_Y = Fp2.pow(base, (Fp.ORDER ** _2n - _1n) / _2n); // u^((p^2 - 1)/3)
    if (!Fp2.eql(PSI2_Y, Fp2.neg(Fp2.ONE)))
        throw new Error('psiFrobenius: PSI2_Y!==-1');
    function psi2(x, y) {
        return [Fp2.mul(x, PSI2_X), Fp2.neg(y)];
    }
    // Map points
    const mapAffine = (fn) => (c, P) => {
        const affine = P.toAffine();
        const p = fn(affine.x, affine.y);
        return c.fromAffine({ x: p[0], y: p[1] });
    };
    const G2psi = mapAffine(psi);
    const G2psi2 = mapAffine(psi2);
    return { psi, psi2, G2psi, G2psi2, PSI_X, PSI_Y, PSI2_X, PSI2_Y };
}
function tower12(opts) {
    const { ORDER } = opts;
    // Fp
    const Fp = mod.Field(ORDER);
    const FpNONRESIDUE = Fp.create(opts.NONRESIDUE || BigInt(-1));
    const FpLegendre = mod.FpLegendre(ORDER);
    const Fpdiv2 = Fp.div(Fp.ONE, _2n); // 1/2
    // Fp2
    const FP2_FROBENIUS_COEFFICIENTS = calcFrobeniusCoefficients(Fp, FpNONRESIDUE, Fp.ORDER, 2)[0];
    const Fp2Add = ({ c0, c1 }, { c0: r0, c1: r1 }) => ({
        c0: Fp.add(c0, r0),
        c1: Fp.add(c1, r1),
    });
    const Fp2Subtract = ({ c0, c1 }, { c0: r0, c1: r1 }) => ({
        c0: Fp.sub(c0, r0),
        c1: Fp.sub(c1, r1),
    });
    const Fp2Multiply = ({ c0, c1 }, rhs) => {
        if (typeof rhs === 'bigint')
            return { c0: Fp.mul(c0, rhs), c1: Fp.mul(c1, rhs) };
        // (a+bi)(c+di) = (ac−bd) + (ad+bc)i
        const { c0: r0, c1: r1 } = rhs;
        let t1 = Fp.mul(c0, r0); // c0 * o0
        let t2 = Fp.mul(c1, r1); // c1 * o1
        // (T1 - T2) + ((c0 + c1) * (r0 + r1) - (T1 + T2))*i
        const o0 = Fp.sub(t1, t2);
        const o1 = Fp.sub(Fp.mul(Fp.add(c0, c1), Fp.add(r0, r1)), Fp.add(t1, t2));
        return { c0: o0, c1: o1 };
    };
    const Fp2Square = ({ c0, c1 }) => {
        const a = Fp.add(c0, c1);
        const b = Fp.sub(c0, c1);
        const c = Fp.add(c0, c0);
        return { c0: Fp.mul(a, b), c1: Fp.mul(c, c1) };
    };
    const Fp2fromBigTuple = (tuple) => {
        if (tuple.length !== 2)
            throw new Error('invalid tuple');
        const fps = tuple.map((n) => Fp.create(n));
        return { c0: fps[0], c1: fps[1] };
    };
    const FP2_ORDER = ORDER * ORDER;
    const Fp2Nonresidue = Fp2fromBigTuple(opts.FP2_NONRESIDUE);
    const Fp2 = {
        ORDER: FP2_ORDER,
        isLE: Fp.isLE,
        NONRESIDUE: Fp2Nonresidue,
        BITS: (0, utils_ts_1.bitLen)(FP2_ORDER),
        BYTES: Math.ceil((0, utils_ts_1.bitLen)(FP2_ORDER) / 8),
        MASK: (0, utils_ts_1.bitMask)((0, utils_ts_1.bitLen)(FP2_ORDER)),
        ZERO: { c0: Fp.ZERO, c1: Fp.ZERO },
        ONE: { c0: Fp.ONE, c1: Fp.ZERO },
        create: (num) => num,
        isValid: ({ c0, c1 }) => typeof c0 === 'bigint' && typeof c1 === 'bigint',
        is0: ({ c0, c1 }) => Fp.is0(c0) && Fp.is0(c1),
        eql: ({ c0, c1 }, { c0: r0, c1: r1 }) => Fp.eql(c0, r0) && Fp.eql(c1, r1),
        neg: ({ c0, c1 }) => ({ c0: Fp.neg(c0), c1: Fp.neg(c1) }),
        pow: (num, power) => mod.FpPow(Fp2, num, power),
        invertBatch: (nums) => mod.FpInvertBatch(Fp2, nums),
        // Normalized
        add: Fp2Add,
        sub: Fp2Subtract,
        mul: Fp2Multiply,
        sqr: Fp2Square,
        // NonNormalized stuff
        addN: Fp2Add,
        subN: Fp2Subtract,
        mulN: Fp2Multiply,
        sqrN: Fp2Square,
        // Why inversion for bigint inside Fp instead of Fp2? it is even used in that context?
        div: (lhs, rhs) => Fp2.mul(lhs, typeof rhs === 'bigint' ? Fp.inv(Fp.create(rhs)) : Fp2.inv(rhs)),
        inv: ({ c0: a, c1: b }) => {
            // We wish to find the multiplicative inverse of a nonzero
            // element a + bu in Fp2. We leverage an identity
            //
            // (a + bu)(a - bu) = a² + b²
            //
            // which holds because u² = -1. This can be rewritten as
            //
            // (a + bu)(a - bu)/(a² + b²) = 1
            //
            // because a² + b² = 0 has no nonzero solutions for (a, b).
            // This gives that (a - bu)/(a² + b²) is the inverse
            // of (a + bu). Importantly, this can be computing using
            // only a single inversion in Fp.
            const factor = Fp.inv(Fp.create(a * a + b * b));
            return { c0: Fp.mul(factor, Fp.create(a)), c1: Fp.mul(factor, Fp.create(-b)) };
        },
        sqrt: (num) => {
            if (opts.Fp2sqrt)
                return opts.Fp2sqrt(num);
            // This is generic for all quadratic extensions (Fp2)
            const { c0, c1 } = num;
            if (Fp.is0(c1)) {
                // if c0 is quadratic residue
                if (Fp.eql(FpLegendre(Fp, c0), Fp.ONE))
                    return Fp2.create({ c0: Fp.sqrt(c0), c1: Fp.ZERO });
                else
                    return Fp2.create({ c0: Fp.ZERO, c1: Fp.sqrt(Fp.div(c0, FpNONRESIDUE)) });
            }
            const a = Fp.sqrt(Fp.sub(Fp.sqr(c0), Fp.mul(Fp.sqr(c1), FpNONRESIDUE)));
            let d = Fp.mul(Fp.add(a, c0), Fpdiv2);
            const legendre = FpLegendre(Fp, d);
            // -1, Quadratic non residue
            if (!Fp.is0(legendre) && !Fp.eql(legendre, Fp.ONE))
                d = Fp.sub(d, a);
            const a0 = Fp.sqrt(d);
            const candidateSqrt = Fp2.create({ c0: a0, c1: Fp.div(Fp.mul(c1, Fpdiv2), a0) });
            if (!Fp2.eql(Fp2.sqr(candidateSqrt), num))
                throw new Error('Cannot find square root');
            // Normalize root: at this point candidateSqrt ** 2 = num, but also -candidateSqrt ** 2 = num
            const x1 = candidateSqrt;
            const x2 = Fp2.neg(x1);
            const { re: re1, im: im1 } = Fp2.reim(x1);
            const { re: re2, im: im2 } = Fp2.reim(x2);
            if (im1 > im2 || (im1 === im2 && re1 > re2))
                return x1;
            return x2;
        },
        // Same as sgn0_m_eq_2 in RFC 9380
        isOdd: (x) => {
            const { re: x0, im: x1 } = Fp2.reim(x);
            const sign_0 = x0 % _2n;
            const zero_0 = x0 === _0n;
            const sign_1 = x1 % _2n;
            return BigInt(sign_0 || (zero_0 && sign_1)) == _1n;
        },
        // Bytes util
        fromBytes(b) {
            if (b.length !== Fp2.BYTES)
                throw new Error('fromBytes invalid length=' + b.length);
            return { c0: Fp.fromBytes(b.subarray(0, Fp.BYTES)), c1: Fp.fromBytes(b.subarray(Fp.BYTES)) };
        },
        toBytes: ({ c0, c1 }) => (0, utils_ts_1.concatBytes)(Fp.toBytes(c0), Fp.toBytes(c1)),
        cmov: ({ c0, c1 }, { c0: r0, c1: r1 }, c) => ({
            c0: Fp.cmov(c0, r0, c),
            c1: Fp.cmov(c1, r1, c),
        }),
        reim: ({ c0, c1 }) => ({ re: c0, im: c1 }),
        // multiply by u + 1
        mulByNonresidue: ({ c0, c1 }) => Fp2.mul({ c0, c1 }, Fp2Nonresidue),
        mulByB: opts.Fp2mulByB,
        fromBigTuple: Fp2fromBigTuple,
        frobeniusMap: ({ c0, c1 }, power) => ({
            c0,
            c1: Fp.mul(c1, FP2_FROBENIUS_COEFFICIENTS[power % 2]),
        }),
    };
    // Fp6
    const Fp6Add = ({ c0, c1, c2 }, { c0: r0, c1: r1, c2: r2 }) => ({
        c0: Fp2.add(c0, r0),
        c1: Fp2.add(c1, r1),
        c2: Fp2.add(c2, r2),
    });
    const Fp6Subtract = ({ c0, c1, c2 }, { c0: r0, c1: r1, c2: r2 }) => ({
        c0: Fp2.sub(c0, r0),
        c1: Fp2.sub(c1, r1),
        c2: Fp2.sub(c2, r2),
    });
    const Fp6Multiply = ({ c0, c1, c2 }, rhs) => {
        if (typeof rhs === 'bigint') {
            return {
                c0: Fp2.mul(c0, rhs),
                c1: Fp2.mul(c1, rhs),
                c2: Fp2.mul(c2, rhs),
            };
        }
        const { c0: r0, c1: r1, c2: r2 } = rhs;
        const t0 = Fp2.mul(c0, r0); // c0 * o0
        const t1 = Fp2.mul(c1, r1); // c1 * o1
        const t2 = Fp2.mul(c2, r2); // c2 * o2
        return {
            // t0 + (c1 + c2) * (r1 * r2) - (T1 + T2) * (u + 1)
            c0: Fp2.add(t0, Fp2.mulByNonresidue(Fp2.sub(Fp2.mul(Fp2.add(c1, c2), Fp2.add(r1, r2)), Fp2.add(t1, t2)))),
            // (c0 + c1) * (r0 + r1) - (T0 + T1) + T2 * (u + 1)
            c1: Fp2.add(Fp2.sub(Fp2.mul(Fp2.add(c0, c1), Fp2.add(r0, r1)), Fp2.add(t0, t1)), Fp2.mulByNonresidue(t2)),
            // T1 + (c0 + c2) * (r0 + r2) - T0 + T2
            c2: Fp2.sub(Fp2.add(t1, Fp2.mul(Fp2.add(c0, c2), Fp2.add(r0, r2))), Fp2.add(t0, t2)),
        };
    };
    const Fp6Square = ({ c0, c1, c2 }) => {
        let t0 = Fp2.sqr(c0); // c0²
        let t1 = Fp2.mul(Fp2.mul(c0, c1), _2n); // 2 * c0 * c1
        let t3 = Fp2.mul(Fp2.mul(c1, c2), _2n); // 2 * c1 * c2
        let t4 = Fp2.sqr(c2); // c2²
        return {
            c0: Fp2.add(Fp2.mulByNonresidue(t3), t0), // T3 * (u + 1) + T0
            c1: Fp2.add(Fp2.mulByNonresidue(t4), t1), // T4 * (u + 1) + T1
            // T1 + (c0 - c1 + c2)² + T3 - T0 - T4
            c2: Fp2.sub(Fp2.sub(Fp2.add(Fp2.add(t1, Fp2.sqr(Fp2.add(Fp2.sub(c0, c1), c2))), t3), t0), t4),
        };
    };
    const [FP6_FROBENIUS_COEFFICIENTS_1, FP6_FROBENIUS_COEFFICIENTS_2] = calcFrobeniusCoefficients(Fp2, Fp2Nonresidue, Fp.ORDER, 6, 2, 3);
    const Fp6 = {
        ORDER: Fp2.ORDER, // TODO: unused, but need to verify
        isLE: Fp2.isLE,
        BITS: 3 * Fp2.BITS,
        BYTES: 3 * Fp2.BYTES,
        MASK: (0, utils_ts_1.bitMask)(3 * Fp2.BITS),
        ZERO: { c0: Fp2.ZERO, c1: Fp2.ZERO, c2: Fp2.ZERO },
        ONE: { c0: Fp2.ONE, c1: Fp2.ZERO, c2: Fp2.ZERO },
        create: (num) => num,
        isValid: ({ c0, c1, c2 }) => Fp2.isValid(c0) && Fp2.isValid(c1) && Fp2.isValid(c2),
        is0: ({ c0, c1, c2 }) => Fp2.is0(c0) && Fp2.is0(c1) && Fp2.is0(c2),
        neg: ({ c0, c1, c2 }) => ({ c0: Fp2.neg(c0), c1: Fp2.neg(c1), c2: Fp2.neg(c2) }),
        eql: ({ c0, c1, c2 }, { c0: r0, c1: r1, c2: r2 }) => Fp2.eql(c0, r0) && Fp2.eql(c1, r1) && Fp2.eql(c2, r2),
        sqrt: utils_ts_1.notImplemented,
        // Do we need division by bigint at all? Should be done via order:
        div: (lhs, rhs) => Fp6.mul(lhs, typeof rhs === 'bigint' ? Fp.inv(Fp.create(rhs)) : Fp6.inv(rhs)),
        pow: (num, power) => mod.FpPow(Fp6, num, power),
        invertBatch: (nums) => mod.FpInvertBatch(Fp6, nums),
        // Normalized
        add: Fp6Add,
        sub: Fp6Subtract,
        mul: Fp6Multiply,
        sqr: Fp6Square,
        // NonNormalized stuff
        addN: Fp6Add,
        subN: Fp6Subtract,
        mulN: Fp6Multiply,
        sqrN: Fp6Square,
        inv: ({ c0, c1, c2 }) => {
            let t0 = Fp2.sub(Fp2.sqr(c0), Fp2.mulByNonresidue(Fp2.mul(c2, c1))); // c0² - c2 * c1 * (u + 1)
            let t1 = Fp2.sub(Fp2.mulByNonresidue(Fp2.sqr(c2)), Fp2.mul(c0, c1)); // c2² * (u + 1) - c0 * c1
            let t2 = Fp2.sub(Fp2.sqr(c1), Fp2.mul(c0, c2)); // c1² - c0 * c2
            // 1/(((c2 * T1 + c1 * T2) * v) + c0 * T0)
            let t4 = Fp2.inv(Fp2.add(Fp2.mulByNonresidue(Fp2.add(Fp2.mul(c2, t1), Fp2.mul(c1, t2))), Fp2.mul(c0, t0)));
            return { c0: Fp2.mul(t4, t0), c1: Fp2.mul(t4, t1), c2: Fp2.mul(t4, t2) };
        },
        // Bytes utils
        fromBytes: (b) => {
            if (b.length !== Fp6.BYTES)
                throw new Error('fromBytes invalid length=' + b.length);
            return {
                c0: Fp2.fromBytes(b.subarray(0, Fp2.BYTES)),
                c1: Fp2.fromBytes(b.subarray(Fp2.BYTES, 2 * Fp2.BYTES)),
                c2: Fp2.fromBytes(b.subarray(2 * Fp2.BYTES)),
            };
        },
        toBytes: ({ c0, c1, c2 }) => (0, utils_ts_1.concatBytes)(Fp2.toBytes(c0), Fp2.toBytes(c1), Fp2.toBytes(c2)),
        cmov: ({ c0, c1, c2 }, { c0: r0, c1: r1, c2: r2 }, c) => ({
            c0: Fp2.cmov(c0, r0, c),
            c1: Fp2.cmov(c1, r1, c),
            c2: Fp2.cmov(c2, r2, c),
        }),
        fromBigSix: (t) => {
            if (!Array.isArray(t) || t.length !== 6)
                throw new Error('invalid Fp6 usage');
            return {
                c0: Fp2.fromBigTuple(t.slice(0, 2)),
                c1: Fp2.fromBigTuple(t.slice(2, 4)),
                c2: Fp2.fromBigTuple(t.slice(4, 6)),
            };
        },
        frobeniusMap: ({ c0, c1, c2 }, power) => ({
            c0: Fp2.frobeniusMap(c0, power),
            c1: Fp2.mul(Fp2.frobeniusMap(c1, power), FP6_FROBENIUS_COEFFICIENTS_1[power % 6]),
            c2: Fp2.mul(Fp2.frobeniusMap(c2, power), FP6_FROBENIUS_COEFFICIENTS_2[power % 6]),
        }),
        mulByFp2: ({ c0, c1, c2 }, rhs) => ({
            c0: Fp2.mul(c0, rhs),
            c1: Fp2.mul(c1, rhs),
            c2: Fp2.mul(c2, rhs),
        }),
        mulByNonresidue: ({ c0, c1, c2 }) => ({ c0: Fp2.mulByNonresidue(c2), c1: c0, c2: c1 }),
        // Sparse multiplication
        mul1: ({ c0, c1, c2 }, b1) => ({
            c0: Fp2.mulByNonresidue(Fp2.mul(c2, b1)),
            c1: Fp2.mul(c0, b1),
            c2: Fp2.mul(c1, b1),
        }),
        // Sparse multiplication
        mul01({ c0, c1, c2 }, b0, b1) {
            let t0 = Fp2.mul(c0, b0); // c0 * b0
            let t1 = Fp2.mul(c1, b1); // c1 * b1
            return {
                // ((c1 + c2) * b1 - T1) * (u + 1) + T0
                c0: Fp2.add(Fp2.mulByNonresidue(Fp2.sub(Fp2.mul(Fp2.add(c1, c2), b1), t1)), t0),
                // (b0 + b1) * (c0 + c1) - T0 - T1
                c1: Fp2.sub(Fp2.sub(Fp2.mul(Fp2.add(b0, b1), Fp2.add(c0, c1)), t0), t1),
                // (c0 + c2) * b0 - T0 + T1
                c2: Fp2.add(Fp2.sub(Fp2.mul(Fp2.add(c0, c2), b0), t0), t1),
            };
        },
    };
    // Fp12
    const FP12_FROBENIUS_COEFFICIENTS = calcFrobeniusCoefficients(Fp2, Fp2Nonresidue, Fp.ORDER, 12, 1, 6)[0];
    const Fp12Add = ({ c0, c1 }, { c0: r0, c1: r1 }) => ({
        c0: Fp6.add(c0, r0),
        c1: Fp6.add(c1, r1),
    });
    const Fp12Subtract = ({ c0, c1 }, { c0: r0, c1: r1 }) => ({
        c0: Fp6.sub(c0, r0),
        c1: Fp6.sub(c1, r1),
    });
    const Fp12Multiply = ({ c0, c1 }, rhs) => {
        if (typeof rhs === 'bigint')
            return { c0: Fp6.mul(c0, rhs), c1: Fp6.mul(c1, rhs) };
        let { c0: r0, c1: r1 } = rhs;
        let t1 = Fp6.mul(c0, r0); // c0 * r0
        let t2 = Fp6.mul(c1, r1); // c1 * r1
        return {
            c0: Fp6.add(t1, Fp6.mulByNonresidue(t2)), // T1 + T2 * v
            // (c0 + c1) * (r0 + r1) - (T1 + T2)
            c1: Fp6.sub(Fp6.mul(Fp6.add(c0, c1), Fp6.add(r0, r1)), Fp6.add(t1, t2)),
        };
    };
    const Fp12Square = ({ c0, c1 }) => {
        let ab = Fp6.mul(c0, c1); // c0 * c1
        return {
            // (c1 * v + c0) * (c0 + c1) - AB - AB * v
            c0: Fp6.sub(Fp6.sub(Fp6.mul(Fp6.add(Fp6.mulByNonresidue(c1), c0), Fp6.add(c0, c1)), ab), Fp6.mulByNonresidue(ab)),
            c1: Fp6.add(ab, ab),
        }; // AB + AB
    };
    function Fp4Square(a, b) {
        const a2 = Fp2.sqr(a);
        const b2 = Fp2.sqr(b);
        return {
            first: Fp2.add(Fp2.mulByNonresidue(b2), a2), // b² * Nonresidue + a²
            second: Fp2.sub(Fp2.sub(Fp2.sqr(Fp2.add(a, b)), a2), b2), // (a + b)² - a² - b²
        };
    }
    const Fp12 = {
        ORDER: Fp2.ORDER, // TODO: unused, but need to verify
        isLE: Fp6.isLE,
        BITS: 2 * Fp6.BITS,
        BYTES: 2 * Fp6.BYTES,
        MASK: (0, utils_ts_1.bitMask)(2 * Fp6.BITS),
        ZERO: { c0: Fp6.ZERO, c1: Fp6.ZERO },
        ONE: { c0: Fp6.ONE, c1: Fp6.ZERO },
        create: (num) => num,
        isValid: ({ c0, c1 }) => Fp6.isValid(c0) && Fp6.isValid(c1),
        is0: ({ c0, c1 }) => Fp6.is0(c0) && Fp6.is0(c1),
        neg: ({ c0, c1 }) => ({ c0: Fp6.neg(c0), c1: Fp6.neg(c1) }),
        eql: ({ c0, c1 }, { c0: r0, c1: r1 }) => Fp6.eql(c0, r0) && Fp6.eql(c1, r1),
        sqrt: utils_ts_1.notImplemented,
        inv: ({ c0, c1 }) => {
            let t = Fp6.inv(Fp6.sub(Fp6.sqr(c0), Fp6.mulByNonresidue(Fp6.sqr(c1)))); // 1 / (c0² - c1² * v)
            return { c0: Fp6.mul(c0, t), c1: Fp6.neg(Fp6.mul(c1, t)) }; // ((C0 * T) * T) + (-C1 * T) * w
        },
        div: (lhs, rhs) => Fp12.mul(lhs, typeof rhs === 'bigint' ? Fp.inv(Fp.create(rhs)) : Fp12.inv(rhs)),
        pow: (num, power) => mod.FpPow(Fp12, num, power),
        invertBatch: (nums) => mod.FpInvertBatch(Fp12, nums),
        // Normalized
        add: Fp12Add,
        sub: Fp12Subtract,
        mul: Fp12Multiply,
        sqr: Fp12Square,
        // NonNormalized stuff
        addN: Fp12Add,
        subN: Fp12Subtract,
        mulN: Fp12Multiply,
        sqrN: Fp12Square,
        // Bytes utils
        fromBytes: (b) => {
            if (b.length !== Fp12.BYTES)
                throw new Error('fromBytes invalid length=' + b.length);
            return {
                c0: Fp6.fromBytes(b.subarray(0, Fp6.BYTES)),
                c1: Fp6.fromBytes(b.subarray(Fp6.BYTES)),
            };
        },
        toBytes: ({ c0, c1 }) => (0, utils_ts_1.concatBytes)(Fp6.toBytes(c0), Fp6.toBytes(c1)),
        cmov: ({ c0, c1 }, { c0: r0, c1: r1 }, c) => ({
            c0: Fp6.cmov(c0, r0, c),
            c1: Fp6.cmov(c1, r1, c),
        }),
        // Utils
        // toString() {
        //   return '' + 'Fp12(' + this.c0 + this.c1 + '* w');
        // },
        // fromTuple(c: [Fp6, Fp6]) {
        //   return new Fp12(...c);
        // }
        fromBigTwelve: (t) => ({
            c0: Fp6.fromBigSix(t.slice(0, 6)),
            c1: Fp6.fromBigSix(t.slice(6, 12)),
        }),
        // Raises to q**i -th power
        frobeniusMap(lhs, power) {
            const { c0, c1, c2 } = Fp6.frobeniusMap(lhs.c1, power);
            const coeff = FP12_FROBENIUS_COEFFICIENTS[power % 12];
            return {
                c0: Fp6.frobeniusMap(lhs.c0, power),
                c1: Fp6.create({
                    c0: Fp2.mul(c0, coeff),
                    c1: Fp2.mul(c1, coeff),
                    c2: Fp2.mul(c2, coeff),
                }),
            };
        },
        mulByFp2: ({ c0, c1 }, rhs) => ({
            c0: Fp6.mulByFp2(c0, rhs),
            c1: Fp6.mulByFp2(c1, rhs),
        }),
        conjugate: ({ c0, c1 }) => ({ c0, c1: Fp6.neg(c1) }),
        // Sparse multiplication
        mul014: ({ c0, c1 }, o0, o1, o4) => {
            let t0 = Fp6.mul01(c0, o0, o1);
            let t1 = Fp6.mul1(c1, o4);
            return {
                c0: Fp6.add(Fp6.mulByNonresidue(t1), t0), // T1 * v + T0
                // (c1 + c0) * [o0, o1+o4] - T0 - T1
                c1: Fp6.sub(Fp6.sub(Fp6.mul01(Fp6.add(c1, c0), o0, Fp2.add(o1, o4)), t0), t1),
            };
        },
        mul034: ({ c0, c1 }, o0, o3, o4) => {
            const a = Fp6.create({
                c0: Fp2.mul(c0.c0, o0),
                c1: Fp2.mul(c0.c1, o0),
                c2: Fp2.mul(c0.c2, o0),
            });
            const b = Fp6.mul01(c1, o3, o4);
            const e = Fp6.mul01(Fp6.add(c0, c1), Fp2.add(o0, o3), o4);
            return {
                c0: Fp6.add(Fp6.mulByNonresidue(b), a),
                c1: Fp6.sub(e, Fp6.add(a, b)),
            };
        },
        // A cyclotomic group is a subgroup of Fp^n defined by
        //   GΦₙ(p) = {α ∈ Fpⁿ : α^Φₙ(p) = 1}
        // The result of any pairing is in a cyclotomic subgroup
        // https://eprint.iacr.org/2009/565.pdf
        _cyclotomicSquare: opts.Fp12cyclotomicSquare,
        _cyclotomicExp: opts.Fp12cyclotomicExp,
        // https://eprint.iacr.org/2010/354.pdf
        // https://eprint.iacr.org/2009/565.pdf
        finalExponentiate: opts.Fp12finalExponentiate,
    };
    return { Fp, Fp2, Fp6, Fp4Square, Fp12 };
}
//# sourceMappingURL=tower.js.map