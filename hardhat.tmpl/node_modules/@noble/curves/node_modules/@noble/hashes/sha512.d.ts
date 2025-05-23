/**
 * SHA2-512 a.k.a. sha512 and sha384. It is slower than sha256 in js because u64 operations are slow.
 *
 * Check out [RFC 4634](https://datatracker.ietf.org/doc/html/rfc4634) and
 * [the paper on truncated SHA512/256](https://eprint.iacr.org/2010/548.pdf).
 * @module
 */
import { HashMD } from './_md.ts';
import { type CHash } from './utils.ts';
export declare class SHA512 extends HashMD<SHA512> {
    protected Ah: number;
    protected Al: number;
    protected Bh: number;
    protected Bl: number;
    protected Ch: number;
    protected Cl: number;
    protected Dh: number;
    protected Dl: number;
    protected Eh: number;
    protected El: number;
    protected Fh: number;
    protected Fl: number;
    protected Gh: number;
    protected Gl: number;
    protected Hh: number;
    protected Hl: number;
    constructor(outputLen?: number);
    protected get(): [
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number,
        number
    ];
    protected set(Ah: number, Al: number, Bh: number, Bl: number, Ch: number, Cl: number, Dh: number, Dl: number, Eh: number, El: number, Fh: number, Fl: number, Gh: number, Gl: number, Hh: number, Hl: number): void;
    protected process(view: DataView, offset: number): void;
    protected roundClean(): void;
    destroy(): void;
}
export declare class SHA512_224 extends SHA512 {
    protected Ah: number;
    protected Al: number;
    protected Bh: number;
    protected Bl: number;
    protected Ch: number;
    protected Cl: number;
    protected Dh: number;
    protected Dl: number;
    protected Eh: number;
    protected El: number;
    protected Fh: number;
    protected Fl: number;
    protected Gh: number;
    protected Gl: number;
    protected Hh: number;
    protected Hl: number;
    constructor();
}
export declare class SHA512_256 extends SHA512 {
    protected Ah: number;
    protected Al: number;
    protected Bh: number;
    protected Bl: number;
    protected Ch: number;
    protected Cl: number;
    protected Dh: number;
    protected Dl: number;
    protected Eh: number;
    protected El: number;
    protected Fh: number;
    protected Fl: number;
    protected Gh: number;
    protected Gl: number;
    protected Hh: number;
    protected Hl: number;
    constructor();
}
export declare class SHA384 extends SHA512 {
    protected Ah: number;
    protected Al: number;
    protected Bh: number;
    protected Bl: number;
    protected Ch: number;
    protected Cl: number;
    protected Dh: number;
    protected Dl: number;
    protected Eh: number;
    protected El: number;
    protected Fh: number;
    protected Fl: number;
    protected Gh: number;
    protected Gl: number;
    protected Hh: number;
    protected Hl: number;
    constructor();
}
/** SHA2-512 hash function. */
export declare const sha512: CHash;
/** SHA2-512/224 "truncated" hash function, with improved resistance to length extension attacks. */
export declare const sha512_224: CHash;
/** SHA2-512/256 "truncated" hash function, with improved resistance to length extension attacks. */
export declare const sha512_256: CHash;
/** SHA2-384 hash function. */
export declare const sha384: CHash;
//# sourceMappingURL=sha512.d.ts.map