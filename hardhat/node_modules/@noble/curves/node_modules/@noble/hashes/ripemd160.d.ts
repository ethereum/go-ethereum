/**
 * RIPEMD-160 legacy hash function.
 * https://homes.esat.kuleuven.be/~bosselae/ripemd160.html
 * https://homes.esat.kuleuven.be/~bosselae/ripemd160/pdf/AB-9601/AB-9601.pdf
 * @module
 */
import { HashMD } from './_md.ts';
import { type CHash } from './utils.ts';
export declare class RIPEMD160 extends HashMD<RIPEMD160> {
    private h0;
    private h1;
    private h2;
    private h3;
    private h4;
    constructor();
    protected get(): [number, number, number, number, number];
    protected set(h0: number, h1: number, h2: number, h3: number, h4: number): void;
    protected process(view: DataView, offset: number): void;
    protected roundClean(): void;
    destroy(): void;
}
/** RIPEMD-160 - a legacy hash function from 1990s. */
export declare const ripemd160: CHash;
//# sourceMappingURL=ripemd160.d.ts.map