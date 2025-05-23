/**
 * Implements [Poseidon](https://www.poseidon-hash.info) ZK-friendly hash.
 *
 * There are many poseidon variants with different constants.
 * We don't provide them: you should construct them manually.
 * Check out [micro-starknet](https://github.com/paulmillr/micro-starknet) package for a proper example.
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
import { FpPow, validateField } from "./modular.js";
export function validateOpts(opts) {
    const { Fp, mds, reversePartialPowIdx: rev, roundConstants: rc } = opts;
    const { roundsFull, roundsPartial, sboxPower, t } = opts;
    validateField(Fp);
    for (const i of ['t', 'roundsFull', 'roundsPartial']) {
        if (typeof opts[i] !== 'number' || !Number.isSafeInteger(opts[i]))
            throw new Error('invalid number ' + i);
    }
    // MDS is TxT matrix
    if (!Array.isArray(mds) || mds.length !== t)
        throw new Error('Poseidon: invalid MDS matrix');
    const _mds = mds.map((mdsRow) => {
        if (!Array.isArray(mdsRow) || mdsRow.length !== t)
            throw new Error('invalid MDS matrix row: ' + mdsRow);
        return mdsRow.map((i) => {
            if (typeof i !== 'bigint')
                throw new Error('invalid MDS matrix bigint: ' + i);
            return Fp.create(i);
        });
    });
    if (rev !== undefined && typeof rev !== 'boolean')
        throw new Error('invalid param reversePartialPowIdx=' + rev);
    if (roundsFull & 1)
        throw new Error('roundsFull is not even' + roundsFull);
    const rounds = roundsFull + roundsPartial;
    if (!Array.isArray(rc) || rc.length !== rounds)
        throw new Error('Poseidon: invalid round constants');
    const roundConstants = rc.map((rc) => {
        if (!Array.isArray(rc) || rc.length !== t)
            throw new Error('invalid round constants');
        return rc.map((i) => {
            if (typeof i !== 'bigint' || !Fp.isValid(i))
                throw new Error('invalid round constant');
            return Fp.create(i);
        });
    });
    if (!sboxPower || ![3, 5, 7].includes(sboxPower))
        throw new Error('invalid sboxPower');
    const _sboxPower = BigInt(sboxPower);
    let sboxFn = (n) => FpPow(Fp, n, _sboxPower);
    // Unwrapped sbox power for common cases (195->142μs)
    if (sboxPower === 3)
        sboxFn = (n) => Fp.mul(Fp.sqrN(n), n);
    else if (sboxPower === 5)
        sboxFn = (n) => Fp.mul(Fp.sqrN(Fp.sqrN(n)), n);
    return Object.freeze({ ...opts, rounds, sboxFn, roundConstants, mds: _mds });
}
export function splitConstants(rc, t) {
    if (typeof t !== 'number')
        throw new Error('poseidonSplitConstants: invalid t');
    if (!Array.isArray(rc) || rc.length % t)
        throw new Error('poseidonSplitConstants: invalid rc');
    const res = [];
    let tmp = [];
    for (let i = 0; i < rc.length; i++) {
        tmp.push(rc[i]);
        if (tmp.length === t) {
            res.push(tmp);
            tmp = [];
        }
    }
    return res;
}
/** Poseidon NTT-friendly hash. */
export function poseidon(opts) {
    const _opts = validateOpts(opts);
    const { Fp, mds, roundConstants, rounds: totalRounds, roundsPartial, sboxFn, t } = _opts;
    const halfRoundsFull = _opts.roundsFull / 2;
    const partialIdx = _opts.reversePartialPowIdx ? t - 1 : 0;
    const poseidonRound = (values, isFull, idx) => {
        values = values.map((i, j) => Fp.add(i, roundConstants[idx][j]));
        if (isFull)
            values = values.map((i) => sboxFn(i));
        else
            values[partialIdx] = sboxFn(values[partialIdx]);
        // Matrix multiplication
        values = mds.map((i) => i.reduce((acc, i, j) => Fp.add(acc, Fp.mulN(i, values[j])), Fp.ZERO));
        return values;
    };
    const poseidonHash = function poseidonHash(values) {
        if (!Array.isArray(values) || values.length !== t)
            throw new Error('invalid values, expected array of bigints with length ' + t);
        values = values.map((i) => {
            if (typeof i !== 'bigint')
                throw new Error('invalid bigint=' + i);
            return Fp.create(i);
        });
        let lastRound = 0;
        // Apply r_f/2 full rounds.
        for (let i = 0; i < halfRoundsFull; i++)
            values = poseidonRound(values, true, lastRound++);
        // Apply r_p partial rounds.
        for (let i = 0; i < roundsPartial; i++)
            values = poseidonRound(values, false, lastRound++);
        // Apply r_f/2 full rounds.
        for (let i = 0; i < halfRoundsFull; i++)
            values = poseidonRound(values, true, lastRound++);
        if (lastRound !== totalRounds)
            throw new Error('invalid number of rounds');
        return values;
    };
    // For verification in tests
    poseidonHash.roundConstants = roundConstants;
    return poseidonHash;
}
//# sourceMappingURL=poseidon.js.map