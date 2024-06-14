/// <reference types="node" />
export interface ECDSASignature {
    v: bigint;
    r: Buffer;
    s: Buffer;
}
/**
 * Returns the ECDSA signature of a message hash.
 *
 * If `chainId` is provided assume an EIP-155-style signature and calculate the `v` value
 * accordingly, otherwise return a "static" `v` just derived from the `recovery` bit
 */
export declare function ecsign(msgHash: Buffer, privateKey: Buffer, chainId?: bigint): ECDSASignature;
/**
 * ECDSA public key recovery from signature.
 * NOTE: Accepts `v === 0 | v === 1` for EIP1559 transactions
 * @returns Recovered public key
 */
export declare const ecrecover: (msgHash: Buffer, v: bigint, r: Buffer, s: Buffer, chainId?: bigint) => Buffer;
/**
 * Convert signature parameters into the format of `eth_sign` RPC method.
 * NOTE: Accepts `v === 0 | v === 1` for EIP1559 transactions
 * @returns Signature
 */
export declare const toRpcSig: (v: bigint, r: Buffer, s: Buffer, chainId?: bigint) => string;
/**
 * Convert signature parameters into the format of Compact Signature Representation (EIP-2098).
 * NOTE: Accepts `v === 0 | v === 1` for EIP1559 transactions
 * @returns Signature
 */
export declare const toCompactSig: (v: bigint, r: Buffer, s: Buffer, chainId?: bigint) => string;
/**
 * Convert signature format of the `eth_sign` RPC method to signature parameters
 *
 * NOTE: For an extracted `v` value < 27 (see Geth bug https://github.com/ethereum/go-ethereum/issues/2053)
 * `v + 27` is returned for the `v` value
 * NOTE: After EIP1559, `v` could be `0` or `1` but this function assumes
 * it's a signed message (EIP-191 or EIP-712) adding `27` at the end. Remove if needed.
 */
export declare const fromRpcSig: (sig: string) => ECDSASignature;
/**
 * Validate a ECDSA signature.
 * NOTE: Accepts `v === 0 | v === 1` for EIP1559 transactions
 * @param homesteadOrLater Indicates whether this is being used on either the homestead hardfork or a later one
 */
export declare const isValidSignature: (v: bigint, r: Buffer, s: Buffer, homesteadOrLater?: boolean, chainId?: bigint) => boolean;
/**
 * Returns the keccak-256 hash of `message`, prefixed with the header used by the `eth_sign` RPC call.
 * The output of this function can be fed into `ecsign` to produce the same signature as the `eth_sign`
 * call for a given `message`, or fed to `ecrecover` along with a signature to recover the public key
 * used to produce the signature.
 */
export declare const hashPersonalMessage: (message: Buffer) => Buffer;
//# sourceMappingURL=signature.d.ts.map