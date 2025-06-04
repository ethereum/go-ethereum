/**
 * Keccak256 hash
 * @param data The data
 */
export declare function keccak256(data: string | ArrayLike<number>): string;
/**
 * Adding padding to string on the left
 * @param value The value
 * @param chars The chars
 */
export declare const padLeft: (value: string, chars: number) => string;
/**
 * Convert bytes to hex
 * @param bytes The bytes
 */
export declare function bytesToHex(bytes: Uint8Array): string;
/**
 * To byte array
 * @param value The value
 */
export declare function toByteArray(value: string | ArrayLike<number>): Uint8Array;
