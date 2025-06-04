"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.blake2s = exports.BLAKE2s = exports.B2S_IV = void 0;
exports.G1s = G1s;
exports.G2s = G2s;
exports.compress = compress;
/**
 * Blake2s hash function. Focuses on 8-bit to 32-bit platforms. blake2b for 64-bit, but in JS it is slower.
 * @module
 */
const _blake_ts_1 = require("./_blake.js");
const _u64_ts_1 = require("./_u64.js");
const utils_ts_1 = require("./utils.js");
/**
 * Initial state: same as SHA256. First 32 bits of the fractional parts of the square roots
 * of the first 8 primes 2..19.
 */
// prettier-ignore
exports.B2S_IV = new Uint32Array([
    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19
]);
// Mixing function G splitted in two halfs
function G1s(a, b, c, d, x) {
    a = (a + b + x) | 0;
    d = (0, utils_ts_1.rotr)(d ^ a, 16);
    c = (c + d) | 0;
    b = (0, utils_ts_1.rotr)(b ^ c, 12);
    return { a, b, c, d };
}
function G2s(a, b, c, d, x) {
    a = (a + b + x) | 0;
    d = (0, utils_ts_1.rotr)(d ^ a, 8);
    c = (c + d) | 0;
    b = (0, utils_ts_1.rotr)(b ^ c, 7);
    return { a, b, c, d };
}
// prettier-ignore
function compress(s, offset, msg, rounds, v0, v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11, v12, v13, v14, v15) {
    let j = 0;
    for (let i = 0; i < rounds; i++) {
        ({ a: v0, b: v4, c: v8, d: v12 } = G1s(v0, v4, v8, v12, msg[offset + s[j++]]));
        ({ a: v0, b: v4, c: v8, d: v12 } = G2s(v0, v4, v8, v12, msg[offset + s[j++]]));
        ({ a: v1, b: v5, c: v9, d: v13 } = G1s(v1, v5, v9, v13, msg[offset + s[j++]]));
        ({ a: v1, b: v5, c: v9, d: v13 } = G2s(v1, v5, v9, v13, msg[offset + s[j++]]));
        ({ a: v2, b: v6, c: v10, d: v14 } = G1s(v2, v6, v10, v14, msg[offset + s[j++]]));
        ({ a: v2, b: v6, c: v10, d: v14 } = G2s(v2, v6, v10, v14, msg[offset + s[j++]]));
        ({ a: v3, b: v7, c: v11, d: v15 } = G1s(v3, v7, v11, v15, msg[offset + s[j++]]));
        ({ a: v3, b: v7, c: v11, d: v15 } = G2s(v3, v7, v11, v15, msg[offset + s[j++]]));
        ({ a: v0, b: v5, c: v10, d: v15 } = G1s(v0, v5, v10, v15, msg[offset + s[j++]]));
        ({ a: v0, b: v5, c: v10, d: v15 } = G2s(v0, v5, v10, v15, msg[offset + s[j++]]));
        ({ a: v1, b: v6, c: v11, d: v12 } = G1s(v1, v6, v11, v12, msg[offset + s[j++]]));
        ({ a: v1, b: v6, c: v11, d: v12 } = G2s(v1, v6, v11, v12, msg[offset + s[j++]]));
        ({ a: v2, b: v7, c: v8, d: v13 } = G1s(v2, v7, v8, v13, msg[offset + s[j++]]));
        ({ a: v2, b: v7, c: v8, d: v13 } = G2s(v2, v7, v8, v13, msg[offset + s[j++]]));
        ({ a: v3, b: v4, c: v9, d: v14 } = G1s(v3, v4, v9, v14, msg[offset + s[j++]]));
        ({ a: v3, b: v4, c: v9, d: v14 } = G2s(v3, v4, v9, v14, msg[offset + s[j++]]));
    }
    return { v0, v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11, v12, v13, v14, v15 };
}
class BLAKE2s extends _blake_ts_1.BLAKE {
    constructor(opts = {}) {
        super(64, opts.dkLen === undefined ? 32 : opts.dkLen, opts, 32, 8, 8);
        // Internal state, same as SHA-256
        this.v0 = exports.B2S_IV[0] | 0;
        this.v1 = exports.B2S_IV[1] | 0;
        this.v2 = exports.B2S_IV[2] | 0;
        this.v3 = exports.B2S_IV[3] | 0;
        this.v4 = exports.B2S_IV[4] | 0;
        this.v5 = exports.B2S_IV[5] | 0;
        this.v6 = exports.B2S_IV[6] | 0;
        this.v7 = exports.B2S_IV[7] | 0;
        const keyLength = opts.key ? opts.key.length : 0;
        this.v0 ^= this.outputLen | (keyLength << 8) | (0x01 << 16) | (0x01 << 24);
        if (opts.salt) {
            const salt = (0, utils_ts_1.u32)((0, utils_ts_1.toBytes)(opts.salt));
            this.v4 ^= (0, utils_ts_1.byteSwapIfBE)(salt[0]);
            this.v5 ^= (0, utils_ts_1.byteSwapIfBE)(salt[1]);
        }
        if (opts.personalization) {
            const pers = (0, utils_ts_1.u32)((0, utils_ts_1.toBytes)(opts.personalization));
            this.v6 ^= (0, utils_ts_1.byteSwapIfBE)(pers[0]);
            this.v7 ^= (0, utils_ts_1.byteSwapIfBE)(pers[1]);
        }
        if (opts.key) {
            // Pad to blockLen and update
            const tmp = new Uint8Array(this.blockLen);
            tmp.set((0, utils_ts_1.toBytes)(opts.key));
            this.update(tmp);
        }
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
    compress(msg, offset, isLast) {
        const { h, l } = (0, _u64_ts_1.fromBig)(BigInt(this.length));
        // prettier-ignore
        const { v0, v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11, v12, v13, v14, v15 } = compress(_blake_ts_1.SIGMA, offset, msg, 10, this.v0, this.v1, this.v2, this.v3, this.v4, this.v5, this.v6, this.v7, exports.B2S_IV[0], exports.B2S_IV[1], exports.B2S_IV[2], exports.B2S_IV[3], l ^ exports.B2S_IV[4], h ^ exports.B2S_IV[5], isLast ? ~exports.B2S_IV[6] : exports.B2S_IV[6], exports.B2S_IV[7]);
        this.v0 ^= v0 ^ v8;
        this.v1 ^= v1 ^ v9;
        this.v2 ^= v2 ^ v10;
        this.v3 ^= v3 ^ v11;
        this.v4 ^= v4 ^ v12;
        this.v5 ^= v5 ^ v13;
        this.v6 ^= v6 ^ v14;
        this.v7 ^= v7 ^ v15;
    }
    destroy() {
        this.destroyed = true;
        this.buffer32.fill(0);
        this.set(0, 0, 0, 0, 0, 0, 0, 0);
    }
}
exports.BLAKE2s = BLAKE2s;
/**
 * Blake2s hash function. Focuses on 8-bit to 32-bit platforms. blake2b for 64-bit, but in JS it is slower.
 * @param msg - message that would be hashed
 * @param opts - dkLen output length, key for MAC mode, salt, personalization
 */
exports.blake2s = (0, utils_ts_1.wrapConstructorWithOpts)((opts) => new BLAKE2s(opts));
//# sourceMappingURL=blake2s.js.map