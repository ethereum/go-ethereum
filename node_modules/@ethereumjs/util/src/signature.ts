import { keccak256 } from 'ethereum-cryptography/keccak'
import { secp256k1 } from 'ethereum-cryptography/secp256k1'

import { bufferToBigInt, bufferToHex, bufferToInt, setLengthLeft, toBuffer } from './bytes'
import { SECP256K1_ORDER, SECP256K1_ORDER_DIV_2 } from './constants'
import { assertIsBuffer } from './helpers'

export interface ECDSASignature {
  v: bigint
  r: Buffer
  s: Buffer
}

/**
 * Returns the ECDSA signature of a message hash.
 *
 * If `chainId` is provided assume an EIP-155-style signature and calculate the `v` value
 * accordingly, otherwise return a "static" `v` just derived from the `recovery` bit
 */
export function ecsign(msgHash: Buffer, privateKey: Buffer, chainId?: bigint): ECDSASignature {
  const sig = secp256k1.sign(msgHash, privateKey)
  const buf = sig.toCompactRawBytes()
  const r = Buffer.from(buf.slice(0, 32))
  const s = Buffer.from(buf.slice(32, 64))

  const v =
    chainId === undefined
      ? BigInt(sig.recovery! + 27)
      : BigInt(sig.recovery! + 35) + BigInt(chainId) * BigInt(2)

  return { r, s, v }
}

function calculateSigRecovery(v: bigint, chainId?: bigint): bigint {
  if (v === BigInt(0) || v === BigInt(1)) return v

  if (chainId === undefined) {
    return v - BigInt(27)
  }
  return v - (chainId * BigInt(2) + BigInt(35))
}

function isValidSigRecovery(recovery: bigint): boolean {
  return recovery === BigInt(0) || recovery === BigInt(1)
}

/**
 * ECDSA public key recovery from signature.
 * NOTE: Accepts `v === 0 | v === 1` for EIP1559 transactions
 * @returns Recovered public key
 */
export const ecrecover = function (
  msgHash: Buffer,
  v: bigint,
  r: Buffer,
  s: Buffer,
  chainId?: bigint
): Buffer {
  const signature = Buffer.concat([setLengthLeft(r, 32), setLengthLeft(s, 32)], 64)
  const recovery = calculateSigRecovery(v, chainId)
  if (!isValidSigRecovery(recovery)) {
    throw new Error('Invalid signature v value')
  }

  const sig = secp256k1.Signature.fromCompact(signature).addRecoveryBit(Number(recovery))
  const senderPubKey = sig.recoverPublicKey(msgHash)
  return Buffer.from(senderPubKey.toRawBytes(false).slice(1))
}

/**
 * Convert signature parameters into the format of `eth_sign` RPC method.
 * NOTE: Accepts `v === 0 | v === 1` for EIP1559 transactions
 * @returns Signature
 */
export const toRpcSig = function (v: bigint, r: Buffer, s: Buffer, chainId?: bigint): string {
  const recovery = calculateSigRecovery(v, chainId)
  if (!isValidSigRecovery(recovery)) {
    throw new Error('Invalid signature v value')
  }

  // geth (and the RPC eth_sign method) uses the 65 byte format used by Bitcoin
  return bufferToHex(Buffer.concat([setLengthLeft(r, 32), setLengthLeft(s, 32), toBuffer(v)]))
}

/**
 * Convert signature parameters into the format of Compact Signature Representation (EIP-2098).
 * NOTE: Accepts `v === 0 | v === 1` for EIP1559 transactions
 * @returns Signature
 */
export const toCompactSig = function (v: bigint, r: Buffer, s: Buffer, chainId?: bigint): string {
  const recovery = calculateSigRecovery(v, chainId)
  if (!isValidSigRecovery(recovery)) {
    throw new Error('Invalid signature v value')
  }

  let ss = s
  if ((v > BigInt(28) && v % BigInt(2) === BigInt(1)) || v === BigInt(1) || v === BigInt(28)) {
    ss = Buffer.from(s)
    ss[0] |= 0x80
  }

  return bufferToHex(Buffer.concat([setLengthLeft(r, 32), setLengthLeft(ss, 32)]))
}

/**
 * Convert signature format of the `eth_sign` RPC method to signature parameters
 *
 * NOTE: For an extracted `v` value < 27 (see Geth bug https://github.com/ethereum/go-ethereum/issues/2053)
 * `v + 27` is returned for the `v` value
 * NOTE: After EIP1559, `v` could be `0` or `1` but this function assumes
 * it's a signed message (EIP-191 or EIP-712) adding `27` at the end. Remove if needed.
 */
export const fromRpcSig = function (sig: string): ECDSASignature {
  const buf: Buffer = toBuffer(sig)

  let r: Buffer
  let s: Buffer
  let v: bigint
  if (buf.length >= 65) {
    r = buf.slice(0, 32)
    s = buf.slice(32, 64)
    v = bufferToBigInt(buf.slice(64))
  } else if (buf.length === 64) {
    // Compact Signature Representation (https://eips.ethereum.org/EIPS/eip-2098)
    r = buf.slice(0, 32)
    s = buf.slice(32, 64)
    v = BigInt(bufferToInt(buf.slice(32, 33)) >> 7)
    s[0] &= 0x7f
  } else {
    throw new Error('Invalid signature length')
  }

  // support both versions of `eth_sign` responses
  if (v < 27) {
    v = v + BigInt(27)
  }

  return {
    v,
    r,
    s,
  }
}

/**
 * Validate a ECDSA signature.
 * NOTE: Accepts `v === 0 | v === 1` for EIP1559 transactions
 * @param homesteadOrLater Indicates whether this is being used on either the homestead hardfork or a later one
 */
export const isValidSignature = function (
  v: bigint,
  r: Buffer,
  s: Buffer,
  homesteadOrLater: boolean = true,
  chainId?: bigint
): boolean {
  if (r.length !== 32 || s.length !== 32) {
    return false
  }

  if (!isValidSigRecovery(calculateSigRecovery(v, chainId))) {
    return false
  }

  const rBigInt = bufferToBigInt(r)
  const sBigInt = bufferToBigInt(s)

  if (
    rBigInt === BigInt(0) ||
    rBigInt >= SECP256K1_ORDER ||
    sBigInt === BigInt(0) ||
    sBigInt >= SECP256K1_ORDER
  ) {
    return false
  }

  if (homesteadOrLater && sBigInt >= SECP256K1_ORDER_DIV_2) {
    return false
  }

  return true
}

/**
 * Returns the keccak-256 hash of `message`, prefixed with the header used by the `eth_sign` RPC call.
 * The output of this function can be fed into `ecsign` to produce the same signature as the `eth_sign`
 * call for a given `message`, or fed to `ecrecover` along with a signature to recover the public key
 * used to produce the signature.
 */
export const hashPersonalMessage = function (message: Buffer): Buffer {
  assertIsBuffer(message)
  const prefix = Buffer.from(`\u0019Ethereum Signed Message:\n${message.length}`, 'utf-8')
  return Buffer.from(keccak256(Buffer.concat([prefix, message])))
}
