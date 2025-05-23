export declare class Address {
    readonly buf: Uint8Array;
    constructor(buf: Uint8Array);
    /**
     * Returns the zero address.
     */
    static zero(): Address;
    /**
     * Is address equal to another.
     */
    equals(address: Address): boolean;
    /**
     * Is address zero.
     */
    isZero(): boolean;
    /**
     * Returns hex encoding of address.
     */
    toString(): string;
    /**
     * Returns Uint8Array representation of address.
     */
    toArray(): Uint8Array;
    /**
     * Returns the ethereum address of a given public key.
     * Accepts "Ethereum public keys" and SEC1 encoded keys.
     * @param pubKey The two points of an uncompressed key, unless sanitize is enabled
     * @param sanitize Accept public keys in other formats
     */
    static publicToAddress(_pubKey: Uint8Array, sanitize?: boolean): Uint8Array;
}
//# sourceMappingURL=address.d.ts.map