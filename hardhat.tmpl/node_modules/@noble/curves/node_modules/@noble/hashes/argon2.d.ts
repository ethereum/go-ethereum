import { type Input } from './utils.ts';
/**
 * Argon2 options.
 * * t: time cost, m: mem cost in kb, p: parallelization.
 * * key: optional key. personalization: arbitrary extra data.
 * * dkLen: desired number of output bytes.
 */
export type ArgonOpts = {
    t: number;
    m: number;
    p: number;
    version?: number;
    key?: Input;
    personalization?: Input;
    dkLen?: number;
    asyncTick?: number;
    maxmem?: number;
    onProgress?: (progress: number) => void;
};
/** argon2d GPU-resistant version. */
export declare const argon2d: (password: Input, salt: Input, opts: ArgonOpts) => Uint8Array;
/** argon2i side-channel-resistant version. */
export declare const argon2i: (password: Input, salt: Input, opts: ArgonOpts) => Uint8Array;
/** argon2id, combining i+d, the most popular version from RFC 9106 */
export declare const argon2id: (password: Input, salt: Input, opts: ArgonOpts) => Uint8Array;
/** argon2d async GPU-resistant version. */
export declare const argon2dAsync: (password: Input, salt: Input, opts: ArgonOpts) => Promise<Uint8Array>;
/** argon2i async side-channel-resistant version. */
export declare const argon2iAsync: (password: Input, salt: Input, opts: ArgonOpts) => Promise<Uint8Array>;
/** argon2id async, combining i+d, the most popular version from RFC 9106 */
export declare const argon2idAsync: (password: Input, salt: Input, opts: ArgonOpts) => Promise<Uint8Array>;
//# sourceMappingURL=argon2.d.ts.map