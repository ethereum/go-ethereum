export interface EthEncryptedData {
    version: string;
    nonce: string;
    ephemPublicKey: string;
    ciphertext: string;
}
/**
 * Encrypt a message.
 *
 * @param options - The encryption options.
 * @param options.publicKey - The public key of the message recipient.
 * @param options.data - The message data.
 * @param options.version - The type of encryption to use.
 * @returns The encrypted data.
 */
export declare function encrypt({ publicKey, data, version, }: {
    publicKey: string;
    data: unknown;
    version: string;
}): EthEncryptedData;
/**
 * Encrypt a message in a way that obscures the message length.
 *
 * The message is padded to a multiple of 2048 before being encrypted so that the length of the
 * resulting encrypted message can't be used to guess the exact length of the original message.
 *
 * @param options - The encryption options.
 * @param options.publicKey - The public key of the message recipient.
 * @param options.data - The message data.
 * @param options.version - The type of encryption to use.
 * @returns The encrypted data.
 */
export declare function encryptSafely({ publicKey, data, version, }: {
    publicKey: string;
    data: unknown;
    version: string;
}): EthEncryptedData;
/**
 * Decrypt a message.
 *
 * @param options - The decryption options.
 * @param options.encryptedData - The encrypted data.
 * @param options.privateKey - The private key to decrypt with.
 * @returns The decrypted message.
 */
export declare function decrypt({ encryptedData, privateKey, }: {
    encryptedData: EthEncryptedData;
    privateKey: string;
}): string;
/**
 * Decrypt a message that has been encrypted using `encryptSafely`.
 *
 * @param options - The decryption options.
 * @param options.encryptedData - The encrypted data.
 * @param options.privateKey - The private key to decrypt with.
 * @returns The decrypted message.
 */
export declare function decryptSafely({ encryptedData, privateKey, }: {
    encryptedData: EthEncryptedData;
    privateKey: string;
}): string;
/**
 * Get the encryption public key for the given key.
 *
 * @param privateKey - The private key to generate the encryption public key with.
 * @returns The encryption public key.
 */
export declare function getEncryptionPublicKey(privateKey: string): string;
