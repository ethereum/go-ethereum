import type { PrefixedHexString } from './types.js';
/**
 * Handling and generating Ethereum addresses
 */
export declare class Address {
    readonly bytes: Uint8Array;
    constructor(bytes: Uint8Array);
    /**
     * Returns the zero address.
     */
    static zero(): Address;
    /**
     * Returns an Address object from a hex-encoded string.
     * @param str - Hex-encoded address
     */
    static fromString(str: string): Address;
    /**
     * Returns an address for a given public key.
     * @param pubKey The two points of an uncompressed key
     */
    static fromPublicKey(pubKey: Uint8Array): Address;
    /**
     * Returns an address for a given private key.
     * @param privateKey A private key must be 256 bits wide
     */
    static fromPrivateKey(privateKey: Uint8Array): Address;
    /**
     * Generates an address for a newly created contract.
     * @param from The address which is creating this new address
     * @param nonce The nonce of the from account
     */
    static generate(from: Address, nonce: bigint): Address;
    /**
     * Generates an address for a contract created using CREATE2.
     * @param from The address which is creating this new address
     * @param salt A salt
     * @param initCode The init code of the contract being created
     */
    static generate2(from: Address, salt: Uint8Array, initCode: Uint8Array): Address;
    /**
     * Is address equal to another.
     */
    equals(address: Address): boolean;
    /**
     * Is address zero.
     */
    isZero(): boolean;
    /**
     * True if address is in the address range defined
     * by EIP-1352
     */
    isPrecompileOrSystemAddress(): boolean;
    /**
     * Returns hex encoding of address.
     */
    toString(): PrefixedHexString;
    /**
     * Returns a new Uint8Array representation of address.
     */
    toBytes(): Uint8Array;
}
//# sourceMappingURL=address.d.ts.map