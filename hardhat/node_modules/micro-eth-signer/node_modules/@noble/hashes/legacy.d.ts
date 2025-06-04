/**
SHA1 (RFC 3174) and MD5 (RFC 1321) legacy, broken hash functions.
Don't use them in a new protocol. What "broken" means:

- Collisions can be made with 2^18 effort in MD5, 2^60 in SHA1.
- No practical pre-image attacks (only theoretical, 2^123.4)
- HMAC seems kinda ok: https://datatracker.ietf.org/doc/html/rfc6151

MD5 architecture is similar to SHA1, with some differences:

- reduced output length: 16 bytes (128 bit) instead of 20
- 64 rounds, instead of 80
- little-endian: could be faster, but will require more code
- non-linear index selection: huge speed-up for unroll
- per round constants: more memory accesses, additional speed-up for unroll
 * @module
 */
import { HashMD } from './_md.ts';
import { type CHash } from './utils.ts';
/** SHA1 legacy hash class. */
export declare class SHA1 extends HashMD<SHA1> {
    private A;
    private B;
    private C;
    private D;
    private E;
    constructor();
    protected get(): [number, number, number, number, number];
    protected set(A: number, B: number, C: number, D: number, E: number): void;
    protected process(view: DataView, offset: number): void;
    protected roundClean(): void;
    destroy(): void;
}
/** SHA1 (RFC 3174) legacy hash function. It was cryptographically broken. */
export declare const sha1: CHash;
/** MD5 legacy hash class. */
export declare class MD5 extends HashMD<MD5> {
    private A;
    private B;
    private C;
    private D;
    constructor();
    protected get(): [number, number, number, number];
    protected set(A: number, B: number, C: number, D: number): void;
    protected process(view: DataView, offset: number): void;
    protected roundClean(): void;
    destroy(): void;
}
/** MD5 (RFC 1321) legacy hash function. It was cryptographically broken. */
export declare const md5: CHash;
//# sourceMappingURL=legacy.d.ts.map