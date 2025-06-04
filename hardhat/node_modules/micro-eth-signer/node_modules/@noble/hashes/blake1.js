"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.blake512 = exports.blake384 = exports.blake256 = exports.blake224 = exports.Blake384 = exports.Blake512 = exports.Blake256 = exports.Blake224 = void 0;
/**
 * Blake1 legacy hash function, one of SHA3 proposals.
 * Rarely used. Check out blake2 or blake3 instead.
 * https://www.aumasson.jp/blake/blake.pdf
 *
 * In the best case, there are 0 allocations.
 *
 * Differences from blake2:
 *
 * - BE instead of LE
 * - Paddings, similar to MD5, RIPEMD, SHA1, SHA2, but:
 *     - length flag is located before actual length
 *     - padding block is compressed differently (no lengths)
 * Instead of msg[sigma[k]], we have `msg[sigma[k]] ^ constants[sigma[k-1]]`
 * (-1 for g1, g2 without -1)
 * - Salt is XOR-ed into constants instead of state
 * - Salt is XOR-ed with output in `compress`
 * - Additional rows (+64 bytes) in SIGMA for new rounds
 * - Different round count:
 *     - 14 / 10 rounds in blake256 / blake2s
 *     - 16 / 12 rounds in blake512 / blake2b
 * - blake512: G1b: rotr 24 -> 25, G2b: rotr 63 -> 11
 * @module
 */
const _assert_ts_1 = require("./_assert.js");
const _blake_ts_1 = require("./_blake.js");
const _md_ts_1 = require("./_md.js");
const _u64_ts_1 = require("./_u64.js");
const blake2s_ts_1 = require("./blake2s.js");
const utils_ts_1 = require("./utils.js");
// Empty zero-filled salt
const EMPTY_SALT = /* @__PURE__ */ new Uint32Array(8);
class Blake1 extends utils_ts_1.Hash {
    constructor(blockLen, outputLen, lengthFlag, counterLen, saltLen, constants, opts = {}) {
        super();
        this.finished = false;
        this.length = 0;
        this.pos = 0;
        this.destroyed = false;
        this.blockLen = blockLen;
        this.outputLen = outputLen;
        this.lengthFlag = lengthFlag;
        this.counterLen = counterLen;
        this.buffer = new Uint8Array(blockLen);
        this.view = (0, utils_ts_1.createView)(this.buffer);
        if (opts.salt) {
            const salt = (0, utils_ts_1.toBytes)(opts.salt);
            if (salt.length !== 4 * saltLen)
                throw new Error('wrong salt length');
            const salt32 = (this.salt = new Uint32Array(saltLen));
            const sv = (0, utils_ts_1.createView)(salt);
            this.constants = constants.slice();
            for (let i = 0, offset = 0; i < salt32.length; i++, offset += 4) {
                salt32[i] = sv.getUint32(offset, false);
                this.constants[i] ^= salt32[i];
            }
        }
        else {
            this.salt = EMPTY_SALT;
            this.constants = constants;
        }
    }
    update(data) {
        (0, _assert_ts_1.aexists)(this);
        data = (0, utils_ts_1.toBytes)(data);
        // From _md, but update length before each compress
        const { view, buffer, blockLen } = this;
        const len = data.length;
        let dataView;
        for (let pos = 0; pos < len;) {
            const take = Math.min(blockLen - this.pos, len - pos);
            // Fast path: we have at least one block in input, cast it to view and process
            if (take === blockLen) {
                if (!dataView)
                    dataView = (0, utils_ts_1.createView)(data);
                for (; blockLen <= len - pos; pos += blockLen) {
                    this.length += blockLen;
                    this.compress(dataView, pos);
                }
                continue;
            }
            buffer.set(data.subarray(pos, pos + take), this.pos);
            this.pos += take;
            pos += take;
            if (this.pos === blockLen) {
                this.length += blockLen;
                this.compress(view, 0, true);
                this.pos = 0;
            }
        }
        return this;
    }
    destroy() {
        this.destroyed = true;
        if (this.salt !== EMPTY_SALT) {
            this.salt.fill(0);
            this.constants.fill(0);
        }
    }
    _cloneInto(to) {
        to || (to = new this.constructor());
        to.set(...this.get());
        const { buffer, length, finished, destroyed, constants, salt, pos } = this;
        to.length = length;
        to.pos = pos;
        to.finished = finished;
        to.destroyed = destroyed;
        to.constants = constants.slice();
        to.salt = salt.slice();
        to.buffer.set(buffer);
        return to;
    }
    digestInto(out) {
        (0, _assert_ts_1.aexists)(this);
        (0, _assert_ts_1.aoutput)(out, this);
        this.finished = true;
        // Padding
        const { buffer, blockLen, counterLen, lengthFlag, view } = this;
        buffer.subarray(this.pos).fill(0); // clean buf
        const counter = BigInt((this.length + this.pos) * 8);
        const counterPos = blockLen - counterLen - 1;
        buffer[this.pos] |= 128; // End block flag
        this.length += this.pos; // add unwritten length
        // Not enough in buffer for length: write what we have.
        if (this.pos > counterPos) {
            this.compress(view, 0);
            buffer.fill(0);
            this.pos = 0;
        }
        // Difference with md: here we have lengthFlag!
        buffer[counterPos] |= lengthFlag; // Length flag
        // We always set 8 byte length flag. Because length will overflow significantly sooner.
        (0, _md_ts_1.setBigUint64)(view, blockLen - 8, counter, false);
        this.compress(view, 0, this.pos !== 0); // don't add length if length is not empty block?
        // Write output
        buffer.fill(0);
        const v = (0, utils_ts_1.createView)(out);
        const state = this.get();
        for (let i = 0; i < this.outputLen / 4; ++i)
            v.setUint32(i * 4, state[i]);
    }
    digest() {
        const { buffer, outputLen } = this;
        this.digestInto(buffer);
        const res = buffer.slice(0, outputLen);
        this.destroy();
        return res;
    }
}
// Same as blake2s!
const C256 = /* @__PURE__ */ Uint32Array.from([
    0x243f6a88, 0x85a308d3, 0x13198a2e, 0x03707344, 0xa4093822, 0x299f31d0, 0x082efa98, 0xec4e6c89,
    0x452821e6, 0x38d01377, 0xbe5466cf, 0x34e90c6c, 0xc0ac29b7, 0xc97c50dd, 0x3f84d5b5, 0xb5470917,
]);
function generateTBL256() {
    const TBL = [];
    for (let i = 0, j = 0; i < 14; i++, j += 16) {
        for (let offset = 1; offset < 16; offset += 2) {
            TBL.push(C256[_blake_ts_1.SIGMA[j + offset]]);
            TBL.push(C256[_blake_ts_1.SIGMA[j + offset - 1]]);
        }
    }
    return new Uint32Array(TBL);
}
const TBL256 = /* @__PURE__ */ generateTBL256(); // C256[SIGMA[X]] precompute
// Temporary buffer, not used to store anything between runs
// Named after SHA256_W buffer (works same)
const BLAKE256_W = /* @__PURE__ */ new Uint32Array(16);
// == SHA224_IV
const B224_IV = new Uint32Array([
    0xc1059ed8, 0x367cd507, 0x3070dd17, 0xf70e5939, 0xffc00b31, 0x68581511, 0x64f98fa7, 0xbefa4fa4,
]);
class Blake1_32 extends Blake1 {
    constructor(outputLen, IV, lengthFlag, opts = {}) {
        super(64, outputLen, lengthFlag, 8, 4, C256, opts);
        this.v0 = IV[0] | 0;
        this.v1 = IV[1] | 0;
        this.v2 = IV[2] | 0;
        this.v3 = IV[3] | 0;
        this.v4 = IV[4] | 0;
        this.v5 = IV[5] | 0;
        this.v6 = IV[6] | 0;
        this.v7 = IV[7] | 0;
    }
    get() {
        const { v0, v1, v2, v3, v4, v5, v6, v7 } = this;
        return [v0, v1, v2, v3, v4, v5, v6, v7];
    }
    // prettier-ignore
    set(v0, v1, v2, v3, v4, v5, v6, v7) {
        this.v0 = v0 | 0;
        this.v1 = v1 | 0;
        this.v2 = v2 | 0;
        this.v3 = v3 | 0;
        this.v4 = v4 | 0;
        this.v5 = v5 | 0;
        this.v6 = v6 | 0;
        this.v7 = v7 | 0;
    }
    destroy() {
        super.destroy();
        this.set(0, 0, 0, 0, 0, 0, 0, 0);
    }
    compress(view, offset, withLength = true) {
        for (let i = 0; i < 16; i++, offset += 4)
            BLAKE256_W[i] = view.getUint32(offset, false);
        // NOTE: we cannot re-use compress from blake2s, since there is additional xor over u256[SIGMA[e]]
        let v00 = this.v0 | 0;
        let v01 = this.v1 | 0;
        let v02 = this.v2 | 0;
        let v03 = this.v3 | 0;
        let v04 = this.v4 | 0;
        let v05 = this.v5 | 0;
        let v06 = this.v6 | 0;
        let v07 = this.v7 | 0;
        let v08 = this.constants[0] | 0;
        let v09 = this.constants[1] | 0;
        let v10 = this.constants[2] | 0;
        let v11 = this.constants[3] | 0;
        const { h, l } = (0, _u64_ts_1.fromBig)(BigInt(withLength ? this.length * 8 : 0));
        let v12 = (this.constants[4] ^ l) >>> 0;
        let v13 = (this.constants[5] ^ l) >>> 0;
        let v14 = (this.constants[6] ^ h) >>> 0;
        let v15 = (this.constants[7] ^ h) >>> 0;
        // prettier-ignore
        for (let i = 0, k = 0, j = 0; i < 14; i++) {
            ({ a: v00, b: v04, c: v08, d: v12 } = (0, blake2s_ts_1.G1s)(v00, v04, v08, v12, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v00, b: v04, c: v08, d: v12 } = (0, blake2s_ts_1.G2s)(v00, v04, v08, v12, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v01, b: v05, c: v09, d: v13 } = (0, blake2s_ts_1.G1s)(v01, v05, v09, v13, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v01, b: v05, c: v09, d: v13 } = (0, blake2s_ts_1.G2s)(v01, v05, v09, v13, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v02, b: v06, c: v10, d: v14 } = (0, blake2s_ts_1.G1s)(v02, v06, v10, v14, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v02, b: v06, c: v10, d: v14 } = (0, blake2s_ts_1.G2s)(v02, v06, v10, v14, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v03, b: v07, c: v11, d: v15 } = (0, blake2s_ts_1.G1s)(v03, v07, v11, v15, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v03, b: v07, c: v11, d: v15 } = (0, blake2s_ts_1.G2s)(v03, v07, v11, v15, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v00, b: v05, c: v10, d: v15 } = (0, blake2s_ts_1.G1s)(v00, v05, v10, v15, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v00, b: v05, c: v10, d: v15 } = (0, blake2s_ts_1.G2s)(v00, v05, v10, v15, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v01, b: v06, c: v11, d: v12 } = (0, blake2s_ts_1.G1s)(v01, v06, v11, v12, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v01, b: v06, c: v11, d: v12 } = (0, blake2s_ts_1.G2s)(v01, v06, v11, v12, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v02, b: v07, c: v08, d: v13 } = (0, blake2s_ts_1.G1s)(v02, v07, v08, v13, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v02, b: v07, c: v08, d: v13 } = (0, blake2s_ts_1.G2s)(v02, v07, v08, v13, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v03, b: v04, c: v09, d: v14 } = (0, blake2s_ts_1.G1s)(v03, v04, v09, v14, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
            ({ a: v03, b: v04, c: v09, d: v14 } = (0, blake2s_ts_1.G2s)(v03, v04, v09, v14, BLAKE256_W[_blake_ts_1.SIGMA[k++]] ^ TBL256[j++]));
        }
        this.v0 = (this.v0 ^ v00 ^ v08 ^ this.salt[0]) >>> 0;
        this.v1 = (this.v1 ^ v01 ^ v09 ^ this.salt[1]) >>> 0;
        this.v2 = (this.v2 ^ v02 ^ v10 ^ this.salt[2]) >>> 0;
        this.v3 = (this.v3 ^ v03 ^ v11 ^ this.salt[3]) >>> 0;
        this.v4 = (this.v4 ^ v04 ^ v12 ^ this.salt[0]) >>> 0;
        this.v5 = (this.v5 ^ v05 ^ v13 ^ this.salt[1]) >>> 0;
        this.v6 = (this.v6 ^ v06 ^ v14 ^ this.salt[2]) >>> 0;
        this.v7 = (this.v7 ^ v07 ^ v15 ^ this.salt[3]) >>> 0;
        BLAKE256_W.fill(0);
    }
}
// Constants
const C512 = /* @__PURE__ */ Uint32Array.from([
    0x243f6a88, 0x85a308d3, 0x13198a2e, 0x03707344, 0xa4093822, 0x299f31d0, 0x082efa98, 0xec4e6c89,
    0x452821e6, 0x38d01377, 0xbe5466cf, 0x34e90c6c, 0xc0ac29b7, 0xc97c50dd, 0x3f84d5b5, 0xb5470917,
    0x9216d5d9, 0x8979fb1b, 0xd1310ba6, 0x98dfb5ac, 0x2ffd72db, 0xd01adfb7, 0xb8e1afed, 0x6a267e96,
    0xba7c9045, 0xf12c7f99, 0x24a19947, 0xb3916cf7, 0x0801f2e2, 0x858efc16, 0x636920d8, 0x71574e69,
]);
// State of SHA512
const B512_IV = /* @__PURE__ */ Uint32Array.from([
    0x6a09e667, 0xf3bcc908, 0xbb67ae85, 0x84caa73b, 0x3c6ef372, 0xfe94f82b, 0xa54ff53a, 0x5f1d36f1,
    0x510e527f, 0xade682d1, 0x9b05688c, 0x2b3e6c1f, 0x1f83d9ab, 0xfb41bd6b, 0x5be0cd19, 0x137e2179,
]);
// State of SHA384
const B384_IV = /* @__PURE__ */ Uint32Array.from([
    0xcbbb9d5d, 0xc1059ed8, 0x629a292a, 0x367cd507, 0x9159015a, 0x3070dd17, 0x152fecd8, 0xf70e5939,
    0x67332667, 0xffc00b31, 0x8eb44a87, 0x68581511, 0xdb0c2e0d, 0x64f98fa7, 0x47b5481d, 0xbefa4fa4,
]);
const BBUF = /* @__PURE__ */ new Uint32Array(32);
const BLAKE512_W = /* @__PURE__ */ new Uint32Array(32);
function generateTBL512() {
    const TBL = [];
    for (let r = 0, k = 0; r < 16; r++, k += 16) {
        for (let offset = 1; offset < 16; offset += 2) {
            TBL.push(C512[_blake_ts_1.SIGMA[k + offset] * 2 + 0]);
            TBL.push(C512[_blake_ts_1.SIGMA[k + offset] * 2 + 1]);
            TBL.push(C512[_blake_ts_1.SIGMA[k + offset - 1] * 2 + 0]);
            TBL.push(C512[_blake_ts_1.SIGMA[k + offset - 1] * 2 + 1]);
        }
    }
    return new Uint32Array(TBL);
}
const TBL512 = /* @__PURE__ */ generateTBL512(); // C512[SIGMA[X]] precompute
// Mixing function G splitted in two halfs
function G1b(a, b, c, d, msg, k) {
    const Xpos = 2 * _blake_ts_1.SIGMA[k];
    const Xl = msg[Xpos + 1] ^ TBL512[k * 2 + 1], Xh = msg[Xpos] ^ TBL512[k * 2]; // prettier-ignore
    let Al = BBUF[2 * a + 1], Ah = BBUF[2 * a]; // prettier-ignore
    let Bl = BBUF[2 * b + 1], Bh = BBUF[2 * b]; // prettier-ignore
    let Cl = BBUF[2 * c + 1], Ch = BBUF[2 * c]; // prettier-ignore
    let Dl = BBUF[2 * d + 1], Dh = BBUF[2 * d]; // prettier-ignore
    // v[a] = (v[a] + v[b] + x) | 0;
    let ll = _u64_ts_1.default.add3L(Al, Bl, Xl);
    Ah = _u64_ts_1.default.add3H(ll, Ah, Bh, Xh) >>> 0;
    Al = (ll | 0) >>> 0;
    // v[d] = rotr(v[d] ^ v[a], 32)
    ({ Dh, Dl } = { Dh: Dh ^ Ah, Dl: Dl ^ Al });
    ({ Dh, Dl } = { Dh: _u64_ts_1.default.rotr32H(Dh, Dl), Dl: _u64_ts_1.default.rotr32L(Dh, Dl) });
    // v[c] = (v[c] + v[d]) | 0;
    ({ h: Ch, l: Cl } = _u64_ts_1.default.add(Ch, Cl, Dh, Dl));
    // v[b] = rotr(v[b] ^ v[c], 25)
    ({ Bh, Bl } = { Bh: Bh ^ Ch, Bl: Bl ^ Cl });
    ({ Bh, Bl } = { Bh: _u64_ts_1.default.rotrSH(Bh, Bl, 25), Bl: _u64_ts_1.default.rotrSL(Bh, Bl, 25) });
    (BBUF[2 * a + 1] = Al), (BBUF[2 * a] = Ah);
    (BBUF[2 * b + 1] = Bl), (BBUF[2 * b] = Bh);
    (BBUF[2 * c + 1] = Cl), (BBUF[2 * c] = Ch);
    (BBUF[2 * d + 1] = Dl), (BBUF[2 * d] = Dh);
}
function G2b(a, b, c, d, msg, k) {
    const Xpos = 2 * _blake_ts_1.SIGMA[k];
    const Xl = msg[Xpos + 1] ^ TBL512[k * 2 + 1], Xh = msg[Xpos] ^ TBL512[k * 2]; // prettier-ignore
    let Al = BBUF[2 * a + 1], Ah = BBUF[2 * a]; // prettier-ignore
    let Bl = BBUF[2 * b + 1], Bh = BBUF[2 * b]; // prettier-ignore
    let Cl = BBUF[2 * c + 1], Ch = BBUF[2 * c]; // prettier-ignore
    let Dl = BBUF[2 * d + 1], Dh = BBUF[2 * d]; // prettier-ignore
    // v[a] = (v[a] + v[b] + x) | 0;
    let ll = _u64_ts_1.default.add3L(Al, Bl, Xl);
    Ah = _u64_ts_1.default.add3H(ll, Ah, Bh, Xh);
    Al = ll | 0;
    // v[d] = rotr(v[d] ^ v[a], 16)
    ({ Dh, Dl } = { Dh: Dh ^ Ah, Dl: Dl ^ Al });
    ({ Dh, Dl } = { Dh: _u64_ts_1.default.rotrSH(Dh, Dl, 16), Dl: _u64_ts_1.default.rotrSL(Dh, Dl, 16) });
    // v[c] = (v[c] + v[d]) | 0;
    ({ h: Ch, l: Cl } = _u64_ts_1.default.add(Ch, Cl, Dh, Dl));
    // v[b] = rotr(v[b] ^ v[c], 11)
    ({ Bh, Bl } = { Bh: Bh ^ Ch, Bl: Bl ^ Cl });
    ({ Bh, Bl } = { Bh: _u64_ts_1.default.rotrSH(Bh, Bl, 11), Bl: _u64_ts_1.default.rotrSL(Bh, Bl, 11) });
    (BBUF[2 * a + 1] = Al), (BBUF[2 * a] = Ah);
    (BBUF[2 * b + 1] = Bl), (BBUF[2 * b] = Bh);
    (BBUF[2 * c + 1] = Cl), (BBUF[2 * c] = Ch);
    (BBUF[2 * d + 1] = Dl), (BBUF[2 * d] = Dh);
}
class Blake1_64 extends Blake1 {
    constructor(outputLen, IV, lengthFlag, opts = {}) {
        super(128, outputLen, lengthFlag, 16, 8, C512, opts);
        this.v0l = IV[0] | 0;
        this.v0h = IV[1] | 0;
        this.v1l = IV[2] | 0;
        this.v1h = IV[3] | 0;
        this.v2l = IV[4] | 0;
        this.v2h = IV[5] | 0;
        this.v3l = IV[6] | 0;
        this.v3h = IV[7] | 0;
        this.v4l = IV[8] | 0;
        this.v4h = IV[9] | 0;
        this.v5l = IV[10] | 0;
        this.v5h = IV[11] | 0;
        this.v6l = IV[12] | 0;
        this.v6h = IV[13] | 0;
        this.v7l = IV[14] | 0;
        this.v7h = IV[15] | 0;
    }
    // prettier-ignore
    get() {
        let { v0l, v0h, v1l, v1h, v2l, v2h, v3l, v3h, v4l, v4h, v5l, v5h, v6l, v6h, v7l, v7h } = this;
        return [v0l, v0h, v1l, v1h, v2l, v2h, v3l, v3h, v4l, v4h, v5l, v5h, v6l, v6h, v7l, v7h];
    }
    // prettier-ignore
    set(v0l, v0h, v1l, v1h, v2l, v2h, v3l, v3h, v4l, v4h, v5l, v5h, v6l, v6h, v7l, v7h) {
        this.v0l = v0l | 0;
        this.v0h = v0h | 0;
        this.v1l = v1l | 0;
        this.v1h = v1h | 0;
        this.v2l = v2l | 0;
        this.v2h = v2h | 0;
        this.v3l = v3l | 0;
        this.v3h = v3h | 0;
        this.v4l = v4l | 0;
        this.v4h = v4h | 0;
        this.v5l = v5l | 0;
        this.v5h = v5h | 0;
        this.v6l = v6l | 0;
        this.v6h = v6h | 0;
        this.v7l = v7l | 0;
        this.v7h = v7h | 0;
    }
    destroy() {
        super.destroy();
        this.set(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0);
    }
    compress(view, offset, withLength = true) {
        for (let i = 0; i < 32; i++, offset += 4)
            BLAKE512_W[i] = view.getUint32(offset, false);
        this.get().forEach((v, i) => (BBUF[i] = v)); // First half from state.
        BBUF.set(this.constants.subarray(0, 16), 16);
        if (withLength) {
            const { h, l } = (0, _u64_ts_1.fromBig)(BigInt(this.length * 8));
            BBUF[24] = (BBUF[24] ^ h) >>> 0;
            BBUF[25] = (BBUF[25] ^ l) >>> 0;
            BBUF[26] = (BBUF[26] ^ h) >>> 0;
            BBUF[27] = (BBUF[27] ^ l) >>> 0;
        }
        for (let i = 0, k = 0; i < 16; i++) {
            G1b(0, 4, 8, 12, BLAKE512_W, k++);
            G2b(0, 4, 8, 12, BLAKE512_W, k++);
            G1b(1, 5, 9, 13, BLAKE512_W, k++);
            G2b(1, 5, 9, 13, BLAKE512_W, k++);
            G1b(2, 6, 10, 14, BLAKE512_W, k++);
            G2b(2, 6, 10, 14, BLAKE512_W, k++);
            G1b(3, 7, 11, 15, BLAKE512_W, k++);
            G2b(3, 7, 11, 15, BLAKE512_W, k++);
            G1b(0, 5, 10, 15, BLAKE512_W, k++);
            G2b(0, 5, 10, 15, BLAKE512_W, k++);
            G1b(1, 6, 11, 12, BLAKE512_W, k++);
            G2b(1, 6, 11, 12, BLAKE512_W, k++);
            G1b(2, 7, 8, 13, BLAKE512_W, k++);
            G2b(2, 7, 8, 13, BLAKE512_W, k++);
            G1b(3, 4, 9, 14, BLAKE512_W, k++);
            G2b(3, 4, 9, 14, BLAKE512_W, k++);
        }
        this.v0l ^= BBUF[0] ^ BBUF[16] ^ this.salt[0];
        this.v0h ^= BBUF[1] ^ BBUF[17] ^ this.salt[1];
        this.v1l ^= BBUF[2] ^ BBUF[18] ^ this.salt[2];
        this.v1h ^= BBUF[3] ^ BBUF[19] ^ this.salt[3];
        this.v2l ^= BBUF[4] ^ BBUF[20] ^ this.salt[4];
        this.v2h ^= BBUF[5] ^ BBUF[21] ^ this.salt[5];
        this.v3l ^= BBUF[6] ^ BBUF[22] ^ this.salt[6];
        this.v3h ^= BBUF[7] ^ BBUF[23] ^ this.salt[7];
        this.v4l ^= BBUF[8] ^ BBUF[24] ^ this.salt[0];
        this.v4h ^= BBUF[9] ^ BBUF[25] ^ this.salt[1];
        this.v5l ^= BBUF[10] ^ BBUF[26] ^ this.salt[2];
        this.v5h ^= BBUF[11] ^ BBUF[27] ^ this.salt[3];
        this.v6l ^= BBUF[12] ^ BBUF[28] ^ this.salt[4];
        this.v6h ^= BBUF[13] ^ BBUF[29] ^ this.salt[5];
        this.v7l ^= BBUF[14] ^ BBUF[30] ^ this.salt[6];
        this.v7h ^= BBUF[15] ^ BBUF[31] ^ this.salt[7];
        BBUF.fill(0);
        BLAKE512_W.fill(0);
    }
}
class Blake224 extends Blake1_32 {
    constructor(opts = {}) {
        super(28, B224_IV, 0, opts);
    }
}
exports.Blake224 = Blake224;
class Blake256 extends Blake1_32 {
    constructor(opts = {}) {
        // == SHA256_IV
        super(32, blake2s_ts_1.B2S_IV, 1, opts);
    }
}
exports.Blake256 = Blake256;
class Blake512 extends Blake1_64 {
    constructor(opts = {}) {
        super(64, B512_IV, 1, opts);
    }
}
exports.Blake512 = Blake512;
class Blake384 extends Blake1_64 {
    constructor(opts = {}) {
        super(48, B384_IV, 0, opts);
    }
}
exports.Blake384 = Blake384;
/** blake1-224 hash function */
exports.blake224 = (0, utils_ts_1.wrapConstructorWithOpts)((opts) => new Blake224(opts));
/** blake1-256 hash function */
exports.blake256 = (0, utils_ts_1.wrapConstructorWithOpts)((opts) => new Blake256(opts));
/** blake1-384 hash function */
exports.blake384 = (0, utils_ts_1.wrapConstructorWithOpts)((opts) => new Blake384(opts));
/** blake1-512 hash function */
exports.blake512 = (0, utils_ts_1.wrapConstructorWithOpts)((opts) => new Blake512(opts));
//# sourceMappingURL=blake1.js.map