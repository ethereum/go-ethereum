/// <reference types="node" />
export interface SignOptions {
    data?: Buffer;
    noncefn?: (message: Buffer, privateKey: Buffer, algo: Buffer | null, data: Buffer | null, attempt: number) => Buffer;
}
export interface SignOptionsV4 {
    data?: Uint8Array;
    noncefn?: (message: Uint8Array, privateKey: Uint8Array, algo: Uint8Array | null, data: Uint8Array | null, attempt: number) => Uint8Array;
}
/**
 * Verify an ECDSA privateKey
 * @method privateKeyVerify
 * @param {Buffer} privateKey
 * @return {boolean}
 */
export declare const privateKeyVerify: (privateKey: Buffer) => boolean;
/**
 * Export a privateKey in DER format
 * @method privateKeyExport
 * @param {Buffer} privateKey
 * @param {boolean} compressed
 * @return {boolean}
 */
export declare const privateKeyExport: (privateKey: Buffer, compressed?: boolean | undefined) => Buffer;
/**
 * Import a privateKey in DER format
 * @method privateKeyImport
 * @param {Buffer} privateKey
 * @return {Buffer}
 */
export declare const privateKeyImport: (privateKey: Buffer) => Buffer;
/**
 * Negate a privateKey by subtracting it from the order of the curve's base point
 * @method privateKeyNegate
 * @param {Buffer} privateKey
 * @return {Buffer}
 */
export declare const privateKeyNegate: (privateKey: Buffer) => Buffer;
/**
 * Compute the inverse of a privateKey (modulo the order of the curve's base point).
 * @method privateKeyModInverse
 * @param {Buffer} privateKey
 * @return {Buffer}
 */
export declare const privateKeyModInverse: (privateKey: Buffer) => Buffer;
/**
 * Tweak a privateKey by adding tweak to it.
 * @method privateKeyTweakAdd
 * @param {Buffer} privateKey
 * @param {Buffer} tweak
 * @return {Buffer}
 */
export declare const privateKeyTweakAdd: (privateKey: Buffer, tweak: Buffer) => Buffer;
/**
 * Tweak a privateKey by multiplying it by a tweak.
 * @method privateKeyTweakMul
 * @param {Buffer} privateKey
 * @param {Buffer} tweak
 * @return {Buffer}
 */
export declare const privateKeyTweakMul: (privateKey: Buffer, tweak: Buffer) => Buffer;
/**
 * Compute the public key for a privateKey.
 * @method publicKeyCreate
 * @param {Buffer} privateKey
 * @param {boolean} compressed
 * @return {Buffer}
 */
export declare const publicKeyCreate: (privateKey: Buffer, compressed?: boolean | undefined) => Buffer;
/**
 * Convert a publicKey to compressed or uncompressed form.
 * @method publicKeyConvert
 * @param {Buffer} publicKey
 * @param {boolean} compressed
 * @return {Buffer}
 */
export declare const publicKeyConvert: (publicKey: Buffer, compressed?: boolean | undefined) => Buffer;
/**
 * Verify an ECDSA publicKey.
 * @method publicKeyVerify
 * @param {Buffer} publicKey
 * @return {boolean}
 */
export declare const publicKeyVerify: (publicKey: Buffer) => boolean;
/**
 * Tweak a publicKey by adding tweak times the generator to it.
 * @method publicKeyTweakAdd
 * @param {Buffer} publicKey
 * @param {Buffer} tweak
 * @param {boolean} compressed
 * @return {Buffer}
 */
export declare const publicKeyTweakAdd: (publicKey: Buffer, tweak: Buffer, compressed?: boolean | undefined) => Buffer;
/**
 * Tweak a publicKey by multiplying it by a tweak value
 * @method publicKeyTweakMul
 * @param {Buffer} publicKey
 * @param {Buffer} tweak
 * @param {boolean} compressed
 * @return {Buffer}
 */
export declare const publicKeyTweakMul: (publicKey: Buffer, tweak: Buffer, compressed?: boolean | undefined) => Buffer;
/**
 * Add a given publicKeys together.
 * @method publicKeyCombine
 * @param {Array<Buffer>} publicKeys
 * @param {boolean} compressed
 * @return {Buffer}
 */
export declare const publicKeyCombine: (publicKeys: Buffer[], compressed?: boolean | undefined) => Buffer;
/**
 * Convert a signature to a normalized lower-S form.
 * @method signatureNormalize
 * @param {Buffer} signature
 * @return {Buffer}
 */
export declare const signatureNormalize: (signature: Buffer) => Buffer;
/**
 * Serialize an ECDSA signature in DER format.
 * @method signatureExport
 * @param {Buffer} signature
 * @return {Buffer}
 */
export declare const signatureExport: (signature: Buffer) => Buffer;
/**
 * Parse a DER ECDSA signature (follow by [BIP66](https://github.com/bitcoin/bips/blob/master/bip-0066.mediawiki)).
 * @method signatureImport
 * @param {Buffer} signature
 * @return {Buffer}
 */
export declare const signatureImport: (signature: Buffer) => Buffer;
/**
 * Parse a DER ECDSA signature (not follow by [BIP66](https://github.com/bitcoin/bips/blob/master/bip-0066.mediawiki)).
 * @method signatureImportLax
 * @param {Buffer} signature
 * @return {Buffer}
 */
export declare const signatureImportLax: (signature: Buffer) => Buffer;
/**
 * Create an ECDSA signature. Always return low-S signature.
 * @method sign
 * @param {Buffer} message
 * @param {Buffer} privateKey
 * @param {Object} options
 * @return {Buffer}
 */
export declare const sign: (message: Buffer, privateKey: Buffer, options?: SignOptions | undefined) => {
    signature: Buffer;
    recovery: number;
};
/**
 * Verify an ECDSA signature.
 * @method verify
 * @param {Buffer} message
 * @param {Buffer} signature
 * @param {Buffer} publicKey
 * @return {boolean}
 */
export declare const verify: (message: Buffer, signature: Buffer, publicKey: Buffer) => boolean;
/**
 * Recover an ECDSA public key from a signature.
 * @method recover
 * @param {Buffer} message
 * @param {Buffer} signature
 * @param {Number} recid
 * @param {boolean} compressed
 * @return {Buffer}
 */
export declare const recover: (message: Buffer, signature: Buffer, recid: number, compressed?: boolean | undefined) => Buffer;
/**
 * Compute an EC Diffie-Hellman secret and applied sha256 to compressed public key.
 * @method ecdh
 * @param {Buffer} publicKey
 * @param {Buffer} privateKey
 * @return {Buffer}
 */
export declare const ecdh: (publicKey: Buffer, privateKey: Buffer) => Buffer;
export declare const ecdhUnsafe: (publicKey: Buffer, privateKey: Buffer, compressed?: boolean | undefined) => Buffer;
