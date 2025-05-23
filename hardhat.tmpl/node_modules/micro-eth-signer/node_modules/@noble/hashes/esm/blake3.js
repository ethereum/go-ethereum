/**
 * Blake3 fast hash is Blake2 with reduced security (round count). Can also be used as MAC & KDF.
 *
 * It is advertised as "the fastest cryptographic hash". However, it isn't true in JS.
 * Why is this so slow? While it should be 6x faster than blake2b, perf diff is only 20%:
 *
 * * There is only 30% reduction in number of rounds from blake2s
 * * Speed-up comes from tree structure, which is parallelized using SIMD & threading.
 *   These features are not present in JS, so we only get overhead from trees.
 * * Parallelization only happens on 1024-byte chunks: there is no benefit for small inputs.
 * * It is still possible to make it faster using: a) loop unrolling b) web workers c) wasm
 * @module
 */
import { abytes, aexists, anumber, aoutput } from "./_assert.js";
import { BLAKE } from "./_blake.js";
import { fromBig } from "./_u64.js";
import { B2S_IV, compress } from "./blake2s.js";
import { byteSwap32, isLE, toBytes, u32, u8, wrapXOFConstructorWithOpts, } from "./utils.js";
// Flag bitset
const B3_Flags = {
    CHUNK_START: 1 << 0,
    CHUNK_END: 1 << 1,
    PARENT: 1 << 2,
    ROOT: 1 << 3,
    KEYED_HASH: 1 << 4,
    DERIVE_KEY_CONTEXT: 1 << 5,
    DERIVE_KEY_MATERIAL: 1 << 6,
};
const SIGMA = /* @__PURE__ */ (() => {
    const Id = Array.from({ length: 16 }, (_, i) => i);
    const permute = (arr) => [2, 6, 3, 10, 7, 0, 4, 13, 1, 11, 12, 5, 9, 14, 15, 8].map((i) => arr[i]);
    const res = [];
    for (let i = 0, v = Id; i < 7; i++, v = permute(v))
        res.push(...v);
    return Uint8Array.from(res);
})();
/** Blake3 hash. Can be used as MAC and KDF. */
export class BLAKE3 extends BLAKE {
    constructor(opts = {}, flags = 0) {
        const olen = opts.dkLen === undefined ? 32 : opts.dkLen;
        super(64, olen, {}, Number.MAX_SAFE_INTEGER, 0, 0);
        this.flags = 0 | 0;
        this.chunkPos = 0; // Position of current block in chunk
        this.chunksDone = 0; // How many chunks we already have
        this.stack = [];
        // Output
        this.posOut = 0;
        this.bufferOut32 = new Uint32Array(16);
        this.chunkOut = 0; // index of output chunk
        this.enableXOF = true;
        anumber(this.outputLen);
        if (opts.key !== undefined && opts.context !== undefined)
            throw new Error('Blake3: only key or context can be specified at same time');
        else if (opts.key !== undefined) {
            const key = toBytes(opts.key).slice();
            if (key.length !== 32)
                throw new Error('Blake3: key should be 32 byte');
            this.IV = u32(key);
            if (!isLE)
                byteSwap32(this.IV);
            this.flags = flags | B3_Flags.KEYED_HASH;
        }
        else if (opts.context !== undefined) {
            const context_key = new BLAKE3({ dkLen: 32 }, B3_Flags.DERIVE_KEY_CONTEXT)
                .update(opts.context)
                .digest();
            this.IV = u32(context_key);
            if (!isLE)
                byteSwap32(this.IV);
            this.flags = flags | B3_Flags.DERIVE_KEY_MATERIAL;
        }
        else {
            this.IV = B2S_IV.slice();
            this.flags = flags;
        }
        this.state = this.IV.slice();
        this.bufferOut = u8(this.bufferOut32);
    }
    // Unused
    get() {
        return [];
    }
    set() { }
    b2Compress(counter, flags, buf, bufPos = 0) {
        const { state: s, pos } = this;
        const { h, l } = fromBig(BigInt(counter), true);
        // prettier-ignore
        const { v0, v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11, v12, v13, v14, v15 } = compress(SIGMA, bufPos, buf, 7, s[0], s[1], s[2], s[3], s[4], s[5], s[6], s[7], B2S_IV[0], B2S_IV[1], B2S_IV[2], B2S_IV[3], h, l, pos, flags);
        s[0] = v0 ^ v8;
        s[1] = v1 ^ v9;
        s[2] = v2 ^ v10;
        s[3] = v3 ^ v11;
        s[4] = v4 ^ v12;
        s[5] = v5 ^ v13;
        s[6] = v6 ^ v14;
        s[7] = v7 ^ v15;
    }
    compress(buf, bufPos = 0, isLast = false) {
        // Compress last block
        let flags = this.flags;
        if (!this.chunkPos)
            flags |= B3_Flags.CHUNK_START;
        if (this.chunkPos === 15 || isLast)
            flags |= B3_Flags.CHUNK_END;
        if (!isLast)
            this.pos = this.blockLen;
        this.b2Compress(this.chunksDone, flags, buf, bufPos);
        this.chunkPos += 1;
        // If current block is last in chunk (16 blocks), then compress chunks
        if (this.chunkPos === 16 || isLast) {
            let chunk = this.state;
            this.state = this.IV.slice();
            // If not the last one, compress only when there are trailing zeros in chunk counter
            // chunks used as binary tree where current stack is path. Zero means current leaf is finished and can be compressed.
            // 1 (001) - leaf not finished (just push current chunk to stack)
            // 2 (010) - leaf finished at depth=1 (merge with last elm on stack and push back)
            // 3 (011) - last leaf not finished
            // 4 (100) - leafs finished at depth=1 and depth=2
            for (let last, chunks = this.chunksDone + 1; isLast || !(chunks & 1); chunks >>= 1) {
                if (!(last = this.stack.pop()))
                    break;
                this.buffer32.set(last, 0);
                this.buffer32.set(chunk, 8);
                this.pos = this.blockLen;
                this.b2Compress(0, this.flags | B3_Flags.PARENT, this.buffer32, 0);
                chunk = this.state;
                this.state = this.IV.slice();
            }
            this.chunksDone++;
            this.chunkPos = 0;
            this.stack.push(chunk);
        }
        this.pos = 0;
    }
    _cloneInto(to) {
        to = super._cloneInto(to);
        const { IV, flags, state, chunkPos, posOut, chunkOut, stack, chunksDone } = this;
        to.state.set(state.slice());
        to.stack = stack.map((i) => Uint32Array.from(i));
        to.IV.set(IV);
        to.flags = flags;
        to.chunkPos = chunkPos;
        to.chunksDone = chunksDone;
        to.posOut = posOut;
        to.chunkOut = chunkOut;
        to.enableXOF = this.enableXOF;
        to.bufferOut32.set(this.bufferOut32);
        return to;
    }
    destroy() {
        this.destroyed = true;
        this.state.fill(0);
        this.buffer32.fill(0);
        this.IV.fill(0);
        this.bufferOut32.fill(0);
        for (let i of this.stack)
            i.fill(0);
    }
    // Same as b2Compress, but doesn't modify state and returns 16 u32 array (instead of 8)
    b2CompressOut() {
        const { state: s, pos, flags, buffer32, bufferOut32: out32 } = this;
        const { h, l } = fromBig(BigInt(this.chunkOut++));
        if (!isLE)
            byteSwap32(buffer32);
        // prettier-ignore
        const { v0, v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11, v12, v13, v14, v15 } = compress(SIGMA, 0, buffer32, 7, s[0], s[1], s[2], s[3], s[4], s[5], s[6], s[7], B2S_IV[0], B2S_IV[1], B2S_IV[2], B2S_IV[3], l, h, pos, flags);
        out32[0] = v0 ^ v8;
        out32[1] = v1 ^ v9;
        out32[2] = v2 ^ v10;
        out32[3] = v3 ^ v11;
        out32[4] = v4 ^ v12;
        out32[5] = v5 ^ v13;
        out32[6] = v6 ^ v14;
        out32[7] = v7 ^ v15;
        out32[8] = s[0] ^ v8;
        out32[9] = s[1] ^ v9;
        out32[10] = s[2] ^ v10;
        out32[11] = s[3] ^ v11;
        out32[12] = s[4] ^ v12;
        out32[13] = s[5] ^ v13;
        out32[14] = s[6] ^ v14;
        out32[15] = s[7] ^ v15;
        if (!isLE) {
            byteSwap32(buffer32);
            byteSwap32(out32);
        }
        this.posOut = 0;
    }
    finish() {
        if (this.finished)
            return;
        this.finished = true;
        // Padding
        this.buffer.fill(0, this.pos);
        // Process last chunk
        let flags = this.flags | B3_Flags.ROOT;
        if (this.stack.length) {
            flags |= B3_Flags.PARENT;
            if (!isLE)
                byteSwap32(this.buffer32);
            this.compress(this.buffer32, 0, true);
            if (!isLE)
                byteSwap32(this.buffer32);
            this.chunksDone = 0;
            this.pos = this.blockLen;
        }
        else {
            flags |= (!this.chunkPos ? B3_Flags.CHUNK_START : 0) | B3_Flags.CHUNK_END;
        }
        this.flags = flags;
        this.b2CompressOut();
    }
    writeInto(out) {
        aexists(this, false);
        abytes(out);
        this.finish();
        const { blockLen, bufferOut } = this;
        for (let pos = 0, len = out.length; pos < len;) {
            if (this.posOut >= blockLen)
                this.b2CompressOut();
            const take = Math.min(blockLen - this.posOut, len - pos);
            out.set(bufferOut.subarray(this.posOut, this.posOut + take), pos);
            this.posOut += take;
            pos += take;
        }
        return out;
    }
    xofInto(out) {
        if (!this.enableXOF)
            throw new Error('XOF is not possible after digest call');
        return this.writeInto(out);
    }
    xof(bytes) {
        anumber(bytes);
        return this.xofInto(new Uint8Array(bytes));
    }
    digestInto(out) {
        aoutput(out, this);
        if (this.finished)
            throw new Error('digest() was already called');
        this.enableXOF = false;
        this.writeInto(out);
        this.destroy();
        return out;
    }
    digest() {
        return this.digestInto(new Uint8Array(this.outputLen));
    }
}
/**
 * BLAKE3 hash function. Can be used as MAC and KDF.
 * @param msg - message that would be hashed
 * @param opts - `dkLen` for output length, `key` for MAC mode, `context` for KDF mode
 * @example
 * const data = new Uint8Array(32);
 * const hash = blake3(data);
 * const mac = blake3(data, { key: new Uint8Array(32) });
 * const kdf = blake3(data, { context: 'application name' });
 */
export const blake3 = /* @__PURE__ */ wrapXOFConstructorWithOpts((opts) => new BLAKE3(opts));
//# sourceMappingURL=blake3.js.map