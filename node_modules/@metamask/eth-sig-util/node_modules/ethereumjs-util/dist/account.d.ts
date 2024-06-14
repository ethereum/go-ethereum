/// <reference types="node" />
/**
 * Returns a zero address.
 */
export declare const zeroAddress: () => string;
/**
 * Checks if the address is a valid. Accepts checksummed addresses too.
 */
export declare const isValidAddress: (address: string) => boolean;
/**
 * Checks if a given address is a zero address.
 */
export declare const isZeroAddress: (address: string) => boolean;
/**
 * Returns a checksummed address.
 *
 * If a eip1191ChainId is provided, the chainId will be included in the checksum calculation. This
 * has the effect of checksummed addresses for one chain having invalid checksums for others.
 * For more details, consult EIP-1191.
 *
 * WARNING: Checksums with and without the chainId will differ. As of 2019-06-26, the most commonly
 * used variation in Ethereum was without the chainId. This may change in the future.
 */
export declare const toChecksumAddress: (address: string, eip1191ChainId?: number | undefined) => string;
/**
 * Checks if the address is a valid checksummed address.
 *
 * See toChecksumAddress' documentation for details about the eip1191ChainId parameter.
 */
export declare const isValidChecksumAddress: (address: string, eip1191ChainId?: number | undefined) => boolean;
/**
 * Generates an address of a newly created contract.
 * @param from The address which is creating this new address
 * @param nonce The nonce of the from account
 */
export declare const generateAddress: (from: Buffer, nonce: Buffer) => Buffer;
/**
 * Generates an address for a contract created using CREATE2.
 * @param from The address which is creating this new address
 * @param salt A salt
 * @param initCode The init code of the contract being created
 */
export declare const generateAddress2: (from: Buffer | string, salt: Buffer | string, initCode: Buffer | string) => Buffer;
/**
 * Returns true if the supplied address belongs to a precompiled account (Byzantium).
 */
export declare const isPrecompiled: (address: Buffer | string) => boolean;
/**
 * Checks if the private key satisfies the rules of the curve secp256k1.
 */
export declare const isValidPrivate: (privateKey: Buffer) => boolean;
/**
 * Checks if the public key satisfies the rules of the curve secp256k1
 * and the requirements of Ethereum.
 * @param publicKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
export declare const isValidPublic: (publicKey: Buffer, sanitize?: boolean) => boolean;
/**
 * Returns the ethereum address of a given public key.
 * Accepts "Ethereum public keys" and SEC1 encoded keys.
 * @param pubKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
export declare const pubToAddress: (pubKey: Buffer, sanitize?: boolean) => Buffer;
export declare const publicToAddress: (pubKey: Buffer, sanitize?: boolean) => Buffer;
/**
 * Returns the ethereum address of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
export declare const privateToAddress: (privateKey: Buffer) => Buffer;
/**
 * Returns the ethereum public key of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
export declare const privateToPublic: (privateKey: Buffer) => Buffer;
/**
 * Converts a public key to the Ethereum format.
 */
export declare const importPublic: (publicKey: Buffer) => Buffer;
