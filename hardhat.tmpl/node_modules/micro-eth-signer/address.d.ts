export declare const addr: {
    RE: RegExp;
    parse: (address: string, allowEmpty?: boolean) => {
        hasPrefix: boolean;
        data: string;
    };
    /**
     * Address checksum is calculated by hashing with keccak_256.
     * It hashes *string*, not a bytearray: keccak('beef') not keccak([0xbe, 0xef])
     * @param nonChecksummedAddress
     * @param allowEmpty - allows '0x'
     * @returns checksummed address
     */
    addChecksum: (nonChecksummedAddress: string, allowEmpty?: boolean) => string;
    /**
     * Creates address from secp256k1 public key.
     */
    fromPublicKey: (key: string | Uint8Array) => string;
    /**
     * Creates address from ETH private key in hex or ui8a format.
     */
    fromPrivateKey: (key: string | Uint8Array) => string;
    /**
     * Generates hex string with new random private key and address. Uses CSPRNG internally.
     */
    random(): {
        privateKey: string;
        address: string;
    };
    /**
     * Verifies checksum if the address is checksummed.
     * Always returns true when the address is not checksummed.
     * @param allowEmpty - allows '0x'
     */
    isValid: (checksummedAddress: string, allowEmpty?: boolean) => boolean;
};
//# sourceMappingURL=address.d.ts.map