/**
 * Is the string a valid ethereum address?
 */
export declare function isAddress(address: any): address is string;
/**
 * Returns a normalized and checksumed address for the given address.
 *
 * @param address - the address to reformat
 * @returns checksumed address
 */
export declare function toChecksumFormat(address: string): string;
/**
 * Determine if two addresses are equal ignoring case (which is a consideration
 * because of checksumming).
 */
export declare function equalAddresses(leftAddress: string, rightAddress: string): boolean;
//# sourceMappingURL=address.d.ts.map