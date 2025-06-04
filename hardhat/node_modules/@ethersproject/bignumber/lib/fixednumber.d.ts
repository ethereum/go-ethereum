import { BytesLike } from "@ethersproject/bytes";
import { BigNumber, BigNumberish } from "./bignumber";
export declare function formatFixed(value: BigNumberish, decimals?: string | BigNumberish): string;
export declare function parseFixed(value: string, decimals?: BigNumberish): BigNumber;
export declare class FixedFormat {
    readonly signed: boolean;
    readonly width: number;
    readonly decimals: number;
    readonly name: string;
    readonly _multiplier: string;
    constructor(constructorGuard: any, signed: boolean, width: number, decimals: number);
    static from(value: any): FixedFormat;
}
export declare class FixedNumber {
    readonly format: FixedFormat;
    readonly _hex: string;
    readonly _value: string;
    readonly _isFixedNumber: boolean;
    constructor(constructorGuard: any, hex: string, value: string, format?: FixedFormat);
    _checkFormat(other: FixedNumber): void;
    addUnsafe(other: FixedNumber): FixedNumber;
    subUnsafe(other: FixedNumber): FixedNumber;
    mulUnsafe(other: FixedNumber): FixedNumber;
    divUnsafe(other: FixedNumber): FixedNumber;
    floor(): FixedNumber;
    ceiling(): FixedNumber;
    round(decimals?: number): FixedNumber;
    isZero(): boolean;
    isNegative(): boolean;
    toString(): string;
    toHexString(width?: number): string;
    toUnsafeFloat(): number;
    toFormat(format: FixedFormat | string): FixedNumber;
    static fromValue(value: BigNumber, decimals?: BigNumberish, format?: FixedFormat | string | number): FixedNumber;
    static fromString(value: string, format?: FixedFormat | string | number): FixedNumber;
    static fromBytes(value: BytesLike, format?: FixedFormat | string | number): FixedNumber;
    static from(value: any, format?: FixedFormat | string | number): FixedNumber;
    static isFixedNumber(value: any): value is FixedNumber;
}
//# sourceMappingURL=fixednumber.d.ts.map