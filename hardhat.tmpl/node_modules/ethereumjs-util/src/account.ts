import assert from 'assert'
import { BN, rlp } from './externals'
import {
  privateKeyVerify,
  publicKeyCreate,
  publicKeyVerify,
  publicKeyConvert,
} from 'ethereum-cryptography/secp256k1'
import { stripHexPrefix } from './internal'
import { KECCAK256_RLP, KECCAK256_NULL } from './constants'
import { zeros, bufferToHex, toBuffer } from './bytes'
import { keccak, keccak256, keccakFromString, rlphash } from './hash'
import { assertIsString, assertIsHexString, assertIsBuffer } from './helpers'
import { BNLike, BufferLike, bnToUnpaddedBuffer, toType, TypeOutput } from './types'

export interface AccountData {
  nonce?: BNLike
  balance?: BNLike
  stateRoot?: BufferLike
  codeHash?: BufferLike
}

export class Account {
  nonce: BN
  balance: BN
  stateRoot: Buffer
  codeHash: Buffer

  static fromAccountData(accountData: AccountData) {
    const { nonce, balance, stateRoot, codeHash } = accountData

    return new Account(
      nonce ? new BN(toBuffer(nonce)) : undefined,
      balance ? new BN(toBuffer(balance)) : undefined,
      stateRoot ? toBuffer(stateRoot) : undefined,
      codeHash ? toBuffer(codeHash) : undefined
    )
  }

  public static fromRlpSerializedAccount(serialized: Buffer) {
    const values = rlp.decode(serialized)

    if (!Array.isArray(values)) {
      throw new Error('Invalid serialized account input. Must be array')
    }

    return this.fromValuesArray(values)
  }

  public static fromValuesArray(values: Buffer[]) {
    const [nonce, balance, stateRoot, codeHash] = values

    return new Account(new BN(nonce), new BN(balance), stateRoot, codeHash)
  }

  /**
   * This constructor assigns and validates the values.
   * Use the static factory methods to assist in creating an Account from varying data types.
   */
  constructor(
    nonce = new BN(0),
    balance = new BN(0),
    stateRoot = KECCAK256_RLP,
    codeHash = KECCAK256_NULL
  ) {
    this.nonce = nonce
    this.balance = balance
    this.stateRoot = stateRoot
    this.codeHash = codeHash

    this._validate()
  }

  private _validate() {
    if (this.nonce.lt(new BN(0))) {
      throw new Error('nonce must be greater than zero')
    }
    if (this.balance.lt(new BN(0))) {
      throw new Error('balance must be greater than zero')
    }
    if (this.stateRoot.length !== 32) {
      throw new Error('stateRoot must have a length of 32')
    }
    if (this.codeHash.length !== 32) {
      throw new Error('codeHash must have a length of 32')
    }
  }

  /**
   * Returns a Buffer Array of the raw Buffers for the account, in order.
   */
  raw(): Buffer[] {
    return [
      bnToUnpaddedBuffer(this.nonce),
      bnToUnpaddedBuffer(this.balance),
      this.stateRoot,
      this.codeHash,
    ]
  }

  /**
   * Returns the RLP serialization of the account as a `Buffer`.
   */
  serialize(): Buffer {
    return rlp.encode(this.raw())
  }

  /**
   * Returns a `Boolean` determining if the account is a contract.
   */
  isContract(): boolean {
    return !this.codeHash.equals(KECCAK256_NULL)
  }

  /**
   * Returns a `Boolean` determining if the account is empty complying to the definition of
   * account emptiness in [EIP-161](https://eips.ethereum.org/EIPS/eip-161):
   * "An account is considered empty when it has no code and zero nonce and zero balance."
   */
  isEmpty(): boolean {
    return this.balance.isZero() && this.nonce.isZero() && this.codeHash.equals(KECCAK256_NULL)
  }
}

/**
 * Checks if the address is a valid. Accepts checksummed addresses too.
 */
export const isValidAddress = function (hexAddress: string): boolean {
  try {
    assertIsString(hexAddress)
  } catch (e: any) {
    return false
  }

  return /^0x[0-9a-fA-F]{40}$/.test(hexAddress)
}

/**
 * Returns a checksummed address.
 *
 * If an eip1191ChainId is provided, the chainId will be included in the checksum calculation. This
 * has the effect of checksummed addresses for one chain having invalid checksums for others.
 * For more details see [EIP-1191](https://eips.ethereum.org/EIPS/eip-1191).
 *
 * WARNING: Checksums with and without the chainId will differ and the EIP-1191 checksum is not
 * backwards compatible to the original widely adopted checksum format standard introduced in
 * [EIP-55](https://eips.ethereum.org/EIPS/eip-55), so this will break in existing applications.
 * Usage of this EIP is therefore discouraged unless you have a very targeted use case.
 */
export const toChecksumAddress = function (hexAddress: string, eip1191ChainId?: BNLike): string {
  assertIsHexString(hexAddress)
  const address = stripHexPrefix(hexAddress).toLowerCase()

  let prefix = ''
  if (eip1191ChainId) {
    const chainId = toType(eip1191ChainId, TypeOutput.BN)
    prefix = chainId.toString() + '0x'
  }

  const hash = keccakFromString(prefix + address).toString('hex')
  let ret = '0x'

  for (let i = 0; i < address.length; i++) {
    if (parseInt(hash[i], 16) >= 8) {
      ret += address[i].toUpperCase()
    } else {
      ret += address[i]
    }
  }

  return ret
}

/**
 * Checks if the address is a valid checksummed address.
 *
 * See toChecksumAddress' documentation for details about the eip1191ChainId parameter.
 */
export const isValidChecksumAddress = function (
  hexAddress: string,
  eip1191ChainId?: BNLike
): boolean {
  return isValidAddress(hexAddress) && toChecksumAddress(hexAddress, eip1191ChainId) === hexAddress
}

/**
 * Generates an address of a newly created contract.
 * @param from The address which is creating this new address
 * @param nonce The nonce of the from account
 */
export const generateAddress = function (from: Buffer, nonce: Buffer): Buffer {
  assertIsBuffer(from)
  assertIsBuffer(nonce)
  const nonceBN = new BN(nonce)

  if (nonceBN.isZero()) {
    // in RLP we want to encode null in the case of zero nonce
    // read the RLP documentation for an answer if you dare
    return rlphash([from, null]).slice(-20)
  }

  // Only take the lower 160bits of the hash
  return rlphash([from, Buffer.from(nonceBN.toArray())]).slice(-20)
}

/**
 * Generates an address for a contract created using CREATE2.
 * @param from The address which is creating this new address
 * @param salt A salt
 * @param initCode The init code of the contract being created
 */
export const generateAddress2 = function (from: Buffer, salt: Buffer, initCode: Buffer): Buffer {
  assertIsBuffer(from)
  assertIsBuffer(salt)
  assertIsBuffer(initCode)

  assert(from.length === 20)
  assert(salt.length === 32)

  const address = keccak256(
    Buffer.concat([Buffer.from('ff', 'hex'), from, salt, keccak256(initCode)])
  )

  return address.slice(-20)
}

/**
 * Checks if the private key satisfies the rules of the curve secp256k1.
 */
export const isValidPrivate = function (privateKey: Buffer): boolean {
  return privateKeyVerify(privateKey)
}

/**
 * Checks if the public key satisfies the rules of the curve secp256k1
 * and the requirements of Ethereum.
 * @param publicKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
export const isValidPublic = function (publicKey: Buffer, sanitize: boolean = false): boolean {
  assertIsBuffer(publicKey)
  if (publicKey.length === 64) {
    // Convert to SEC1 for secp256k1
    return publicKeyVerify(Buffer.concat([Buffer.from([4]), publicKey]))
  }

  if (!sanitize) {
    return false
  }

  return publicKeyVerify(publicKey)
}

/**
 * Returns the ethereum address of a given public key.
 * Accepts "Ethereum public keys" and SEC1 encoded keys.
 * @param pubKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
export const pubToAddress = function (pubKey: Buffer, sanitize: boolean = false): Buffer {
  assertIsBuffer(pubKey)
  if (sanitize && pubKey.length !== 64) {
    pubKey = Buffer.from(publicKeyConvert(pubKey, false).slice(1))
  }
  assert(pubKey.length === 64)
  // Only take the lower 160bits of the hash
  return keccak(pubKey).slice(-20)
}
export const publicToAddress = pubToAddress

/**
 * Returns the ethereum public key of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
export const privateToPublic = function (privateKey: Buffer): Buffer {
  assertIsBuffer(privateKey)
  // skip the type flag and use the X, Y points
  return Buffer.from(publicKeyCreate(privateKey, false)).slice(1)
}

/**
 * Returns the ethereum address of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
export const privateToAddress = function (privateKey: Buffer): Buffer {
  return publicToAddress(privateToPublic(privateKey))
}

/**
 * Converts a public key to the Ethereum format.
 */
export const importPublic = function (publicKey: Buffer): Buffer {
  assertIsBuffer(publicKey)
  if (publicKey.length !== 64) {
    publicKey = Buffer.from(publicKeyConvert(publicKey, false).slice(1))
  }
  return publicKey
}

/**
 * Returns the zero address.
 */
export const zeroAddress = function (): string {
  const addressLength = 20
  const addr = zeros(addressLength)
  return bufferToHex(addr)
}

/**
 * Checks if a given address is the zero address.
 */
export const isZeroAddress = function (hexAddress: string): boolean {
  try {
    assertIsString(hexAddress)
  } catch (e: any) {
    return false
  }

  const zeroAddr = zeroAddress()
  return zeroAddr === hexAddress
}
