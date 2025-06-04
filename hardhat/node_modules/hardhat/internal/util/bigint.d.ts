export declare function min(x: bigint, y: bigint): bigint;
export declare function max(x: bigint, y: bigint): bigint;
export declare function isBigInt(x: unknown): x is bigint;
export declare function divUp(x: bigint, y: bigint): bigint;
export declare function cmp(a: bigint, b: bigint): number;
/**
 * Converts the number to a hexadecimal string with a length of 32
 * bytes. This hex string is NOT 0x-prefixed.
 */
export declare function toEvmWord(x: bigint | number): string;
export declare function fromBigIntLike(x: string | number | bigint | Uint8Array | undefined): bigint | undefined;
export declare function toHex(x: number | bigint): string;
//# sourceMappingURL=bigint.d.ts.map