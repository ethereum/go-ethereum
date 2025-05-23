import { ValidInputTypes } from '../types.js';
/**
 * Checks if a given value is a valid big int
 */
export declare const isBigInt: (value: ValidInputTypes) => boolean;
/** @internal */
export declare const bigintPower: (base: bigint, expo: bigint) => bigint;
export declare const isUInt: (value: ValidInputTypes, options?: {
    abiType: string;
    bitSize?: never;
} | {
    bitSize: number;
    abiType?: never;
}) => boolean;
export declare const isInt: (value: ValidInputTypes, options?: {
    abiType: string;
    bitSize?: never;
} | {
    bitSize: number;
    abiType?: never;
}) => boolean;
export declare const isNumber: (value: ValidInputTypes) => boolean;
//# sourceMappingURL=numbers.d.ts.map