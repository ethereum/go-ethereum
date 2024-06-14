import type { LegacyTxInterface } from '../types.js';
export declare function errorMsg(tx: LegacyTxInterface, msg: string): string;
export declare function isSigned(tx: LegacyTxInterface): boolean;
/**
 * The amount of gas paid for the data in this tx
 */
export declare function getDataFee(tx: LegacyTxInterface, extraCost?: bigint): bigint;
export declare function hash(tx: LegacyTxInterface): Uint8Array;
/**
 * EIP-2: All transaction signatures whose s-value is greater than secp256k1n/2are considered invalid.
 * Reasoning: https://ethereum.stackexchange.com/a/55728
 */
export declare function validateHighS(tx: LegacyTxInterface): void;
export declare function getSenderPublicKey(tx: LegacyTxInterface): Uint8Array;
//# sourceMappingURL=legacy.d.ts.map