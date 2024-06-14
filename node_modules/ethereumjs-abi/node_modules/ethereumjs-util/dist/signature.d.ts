/// <reference types="node" />
export interface ECDSASignature {
    v: number;
    r: Buffer;
    s: Buffer;
}
/**
 * Returns the ECDSA signature of a message hash.
 */
export declare const ecsign: (msgHash: Buffer, privateKey: Buffer, chainId?: number | undefined) => ECDSASignature;
/**
 * ECDSA public key recovery from signature.
 * @returns Recovered public key
 */
export declare const ecrecover: (msgHash: Buffer, v: number, r: Buffer, s: Buffer, chainId?: number | undefined) => Buffer;
/**
 * Convert signature parameters into the format of `eth_sign` RPC method.
 * @returns Signature
 */
export declare const toRpcSig: (v: number, r: Buffer, s: Buffer, chainId?: number | undefined) => string;
/**
 * Convert signature format of the `eth_sign` RPC method to signature parameters
 * NOTE: all because of a bug in geth: https://github.com/ethereum/go-ethereum/issues/2053
 */
export declare const fromRpcSig: (sig: string) => ECDSASignature;
/**
 * Validate a ECDSA signature.
 * @param homesteadOrLater Indicates whether this is being used on either the homestead hardfork or a later one
 */
export declare const isValidSignature: (v: number, r: Buffer, s: Buffer, homesteadOrLater?: boolean, chainId?: number | undefined) => boolean;
/**
 * Returns the keccak-256 hash of `message`, prefixed with the header used by the `eth_sign` RPC call.
 * The output of this function can be fed into `ecsign` to produce the same signature as the `eth_sign`
 * call for a given `message`, or fed to `ecrecover` along with a signature to recover the public key
 * used to produce the signature.
 */
export declare const hashPersonalMessage: (message: Buffer) => Buffer;
