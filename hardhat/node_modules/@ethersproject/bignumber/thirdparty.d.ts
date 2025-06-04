declare module "bn.js" {
    export class BN {
        constructor(value: string | number, radix?: number);

        add(other: BN): BN;
        sub(other: BN): BN;
        div(other: BN): BN;
        mod(other: BN): BN;
        mul(other: BN): BN;

        pow(other: BN): BN;
        umod(other: BN): BN;

        eq(other: BN): boolean;
        lt(other: BN): boolean;
        lte(other: BN): boolean;
        gt(other: BN): boolean;
        gte(other: BN): boolean;

        isNeg(): boolean;
        isZero(): boolean;

        toTwos(other: number): BN;
        fromTwos(other: number): BN;

        or(other: BN): BN;
        and(other: BN): BN;
        xor(other: BN): BN;
        shln(other: number): BN;
        shrn(other: number): BN;
        maskn(other: number): BN;

        toString(radix: number): string;
        toNumber(): number;
        toArray(endian: string, width: number): Uint8Array;
        encode(encoding: string, compact: boolean): Uint8Array;
    }
}
