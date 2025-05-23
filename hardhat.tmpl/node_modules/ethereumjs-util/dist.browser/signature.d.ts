/// <reference types="node" />
import { BNLike } from './types';
export interface ECDSASignature {
    v: number;
    r: Buffer;
    s: Buffer;
}
export interface ECDSASignatureBuffer {
    v: Buffer;
    r: Buffer;
    s: Buffer;
}
/**
 * Returns the ECDSA signature of a message hash.
 */
export declare function ecsign(msgHash: Buffer, privateKey: Buffer, chainId?: number): ECDSASignature;
export declare function ecsign(msgHash: Buffer, privateKey: Buffer, chainId: BNLike): ECDSASignatureBuffer;
/**
 * ECDSA public key recovery from signature.
 * NOTE: Accepts `v == 0 | v == 1` for EIP1559 transactions
 * @returns Recovered public key
 */
export declare const ecrecover: (msgHash: Buffer, v: BNLike, r: Buffer, s: Buffer, chainId?: BNLike) => Buffer;
/**
 * Convert signature parameters into the format of `eth_sign` RPC method.
 * NOTE: Accepts `v == 0 | v == 1` for EIP1559 transactions
 * @returns Signature
 */
export declare const toRpcSig: (v: BNLike, r: Buffer, s: Buffer, chainId?: BNLike) => string;
/**
 * Convert signature parameters into the format of Compact Signature Representation (EIP-2098).
 * NOTE: Accepts `v == 0 | v == 1` for EIP1559 transactions
 * @returns Signature
 */
export declare const toCompactSig: (v: BNLike, r: Buffer, s: Buffer, chainId?: BNLike) => string;
/**
 * Convert signature format of the `eth_sign` RPC method to signature parameters
 * NOTE: all because of a bug in geth: https://github.com/ethereum/go-ethereum/issues/2053
 * NOTE: After EIP1559, `v` could be `0` or `1` but this function assumes
 * it's a signed message (EIP-191 or EIP-712) adding `27` at the end. Remove if needed.
 */
export declare const fromRpcSig: (sig: string) => ECDSASignature;
/**
 * Validate a ECDSA signature.
 * NOTE: Accepts `v == 0 | v == 1` for EIP1559 transactions
 * @param homesteadOrLater Indicates whether this is being used on either the homestead hardfork or a later one
 */
export declare const isValidSignature: (v: BNLike, r: Buffer, s: Buffer, homesteadOrLater?: boolean, chainId?: BNLike) => boolean;
/**
 * Returns the keccak-256 hash of `message`, prefixed with the header used by the `eth_sign` RPC call.
 * The output of this function can be fed into `ecsign` to produce the same signature as the `eth_sign`
 * call for a given `message`, or fed to `ecrecover` along with a signature to recover the public key
 * used to produce the signature.
 */
export declare const hashPersonalMessage: (message: Buffer) => Buffer;
