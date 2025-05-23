/**
 * Blake2s hash function. Focuses on 8-bit to 32-bit platforms. blake2b for 64-bit, but in JS it is slower.
 * @module
 */
import { BLAKE, type BlakeOpts, SIGMA } from './_blake.ts';
import { fromBig } from './_u64.ts';
import { byteSwapIfBE, type CHashO, rotr, toBytes, u32, wrapConstructorWithOpts } from './utils.ts';

/**
 * Initial state: same as SHA256. First 32 bits of the fractional parts of the square roots
 * of the first 8 primes 2..19.
 */
// prettier-ignore
export const B2S_IV: Uint32Array = /* @__PURE__ */ new Uint32Array([
  0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19
]);

// prettier-ignore
type Num4 = { a: number; b: number; c: number; d: number; };
// prettier-ignore
type Num16 = {
  v0: number; v1: number; v2: number; v3: number;
  v4: number; v5: number; v6: number; v7: number;
  v8: number; v9: number; v10: number; v11: number;
  v12: number; v13: number; v14: number; v15: number;
};

// Mixing function G splitted in two halfs
export function G1s(a: number, b: number, c: number, d: number, x: number): Num4 {
  a = (a + b + x) | 0;
  d = rotr(d ^ a, 16);
  c = (c + d) | 0;
  b = rotr(b ^ c, 12);
  return { a, b, c, d };
}

export function G2s(a: number, b: number, c: number, d: number, x: number): Num4 {
  a = (a + b + x) | 0;
  d = rotr(d ^ a, 8);
  c = (c + d) | 0;
  b = rotr(b ^ c, 7);
  return { a, b, c, d };
}

// prettier-ignore
export function compress(s: Uint8Array, offset: number, msg: Uint32Array, rounds: number,
  v0: number, v1: number, v2: number, v3: number, v4: number, v5: number, v6: number, v7: number,
  v8: number, v9: number, v10: number, v11: number, v12: number, v13: number, v14: number, v15: number,
): Num16 {
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

export class BLAKE2s extends BLAKE<BLAKE2s> {
  // Internal state, same as SHA-256
  private v0 = B2S_IV[0] | 0;
  private v1 = B2S_IV[1] | 0;
  private v2 = B2S_IV[2] | 0;
  private v3 = B2S_IV[3] | 0;
  private v4 = B2S_IV[4] | 0;
  private v5 = B2S_IV[5] | 0;
  private v6 = B2S_IV[6] | 0;
  private v7 = B2S_IV[7] | 0;

  constructor(opts: BlakeOpts = {}) {
    super(64, opts.dkLen === undefined ? 32 : opts.dkLen, opts, 32, 8, 8);
    const keyLength = opts.key ? opts.key.length : 0;
    this.v0 ^= this.outputLen | (keyLength << 8) | (0x01 << 16) | (0x01 << 24);
    if (opts.salt) {
      const salt = u32(toBytes(opts.salt));
      this.v4 ^= byteSwapIfBE(salt[0]);
      this.v5 ^= byteSwapIfBE(salt[1]);
    }
    if (opts.personalization) {
      const pers = u32(toBytes(opts.personalization));
      this.v6 ^= byteSwapIfBE(pers[0]);
      this.v7 ^= byteSwapIfBE(pers[1]);
    }
    if (opts.key) {
      // Pad to blockLen and update
      const tmp = new Uint8Array(this.blockLen);
      tmp.set(toBytes(opts.key));
      this.update(tmp);
    }
  }
  protected get(): [number, number, number, number, number, number, number, number] {
    const { v0, v1, v2, v3, v4, v5, v6, v7 } = this;
    return [v0, v1, v2, v3, v4, v5, v6, v7];
  }
  // prettier-ignore
  protected set(
    v0: number, v1: number, v2: number, v3: number, v4: number, v5: number, v6: number, v7: number
  ): void {
    this.v0 = v0 | 0;
    this.v1 = v1 | 0;
    this.v2 = v2 | 0;
    this.v3 = v3 | 0;
    this.v4 = v4 | 0;
    this.v5 = v5 | 0;
    this.v6 = v6 | 0;
    this.v7 = v7 | 0;
  }
  protected compress(msg: Uint32Array, offset: number, isLast: boolean): void {
    const { h, l } = fromBig(BigInt(this.length));
    // prettier-ignore
    const { v0, v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11, v12, v13, v14, v15 } =
      compress(
        SIGMA, offset, msg, 10,
        this.v0, this.v1, this.v2, this.v3, this.v4, this.v5, this.v6, this.v7,
        B2S_IV[0], B2S_IV[1], B2S_IV[2], B2S_IV[3], l ^ B2S_IV[4], h ^ B2S_IV[5], isLast ? ~B2S_IV[6] : B2S_IV[6], B2S_IV[7]
      );
    this.v0 ^= v0 ^ v8;
    this.v1 ^= v1 ^ v9;
    this.v2 ^= v2 ^ v10;
    this.v3 ^= v3 ^ v11;
    this.v4 ^= v4 ^ v12;
    this.v5 ^= v5 ^ v13;
    this.v6 ^= v6 ^ v14;
    this.v7 ^= v7 ^ v15;
  }
  destroy(): void {
    this.destroyed = true;
    this.buffer32.fill(0);
    this.set(0, 0, 0, 0, 0, 0, 0, 0);
  }
}

/**
 * Blake2s hash function. Focuses on 8-bit to 32-bit platforms. blake2b for 64-bit, but in JS it is slower.
 * @param msg - message that would be hashed
 * @param opts - dkLen output length, key for MAC mode, salt, personalization
 */
export const blake2s: CHashO = /* @__PURE__ */ wrapConstructorWithOpts<BLAKE2s, BlakeOpts>(
  (opts) => new BLAKE2s(opts)
);
