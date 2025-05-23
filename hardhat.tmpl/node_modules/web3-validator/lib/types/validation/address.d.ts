import { ValidInputTypes } from '../types.js';
/**
 * Checks the checksum of a given address. Will also return false on non-checksum addresses.
 */
export declare const checkAddressCheckSum: (data: string) => boolean;
/**
 * Checks if a given string is a valid Ethereum address. It will also check the checksum, if the address has upper and lowercase letters.
 */
export declare const isAddress: (value: ValidInputTypes, checkChecksum?: boolean) => boolean;
//# sourceMappingURL=address.d.ts.map