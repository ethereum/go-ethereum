import {
  generateAddress,
  generateAddress2,
  isValidAddress,
  privateToAddress,
  pubToAddress,
} from './account.js'
import {
  bigIntToBytes,
  bytesToBigInt,
  bytesToHex,
  equalsBytes,
  hexToBytes,
  zeros,
} from './bytes.js'
import { BIGINT_0 } from './constants.js'

import type { PrefixedHexString } from './types.js'

/**
 * Handling and generating Ethereum addresses
 */
export class Address {
  public readonly bytes: Uint8Array

  constructor(bytes: Uint8Array) {
    if (bytes.length !== 20) {
      throw new Error('Invalid address length')
    }
    this.bytes = bytes
  }

  /**
   * Returns the zero address.
   */
  static zero(): Address {
    return new Address(zeros(20))
  }

  /**
   * Returns an Address object from a hex-encoded string.
   * @param str - Hex-encoded address
   */
  static fromString(str: string): Address {
    if (!isValidAddress(str)) {
      throw new Error(`Invalid address input=${str}`)
    }
    return new Address(hexToBytes(str))
  }

  /**
   * Returns an address for a given public key.
   * @param pubKey The two points of an uncompressed key
   */
  static fromPublicKey(pubKey: Uint8Array): Address {
    if (!(pubKey instanceof Uint8Array)) {
      throw new Error('Public key should be Uint8Array')
    }
    const bytes = pubToAddress(pubKey)
    return new Address(bytes)
  }

  /**
   * Returns an address for a given private key.
   * @param privateKey A private key must be 256 bits wide
   */
  static fromPrivateKey(privateKey: Uint8Array): Address {
    if (!(privateKey instanceof Uint8Array)) {
      throw new Error('Private key should be Uint8Array')
    }
    const bytes = privateToAddress(privateKey)
    return new Address(bytes)
  }

  /**
   * Generates an address for a newly created contract.
   * @param from The address which is creating this new address
   * @param nonce The nonce of the from account
   */
  static generate(from: Address, nonce: bigint): Address {
    if (typeof nonce !== 'bigint') {
      throw new Error('Expected nonce to be a bigint')
    }
    return new Address(generateAddress(from.bytes, bigIntToBytes(nonce)))
  }

  /**
   * Generates an address for a contract created using CREATE2.
   * @param from The address which is creating this new address
   * @param salt A salt
   * @param initCode The init code of the contract being created
   */
  static generate2(from: Address, salt: Uint8Array, initCode: Uint8Array): Address {
    if (!(salt instanceof Uint8Array)) {
      throw new Error('Expected salt to be a Uint8Array')
    }
    if (!(initCode instanceof Uint8Array)) {
      throw new Error('Expected initCode to be a Uint8Array')
    }
    return new Address(generateAddress2(from.bytes, salt, initCode))
  }

  /**
   * Is address equal to another.
   */
  equals(address: Address): boolean {
    return equalsBytes(this.bytes, address.bytes)
  }

  /**
   * Is address zero.
   */
  isZero(): boolean {
    return this.equals(Address.zero())
  }

  /**
   * True if address is in the address range defined
   * by EIP-1352
   */
  isPrecompileOrSystemAddress(): boolean {
    const address = bytesToBigInt(this.bytes)
    const rangeMin = BIGINT_0
    const rangeMax = BigInt('0xffff')
    return address >= rangeMin && address <= rangeMax
  }

  /**
   * Returns hex encoding of address.
   */
  toString(): PrefixedHexString {
    return bytesToHex(this.bytes)
  }

  /**
   * Returns a new Uint8Array representation of address.
   */
  toBytes(): Uint8Array {
    return new Uint8Array(this.bytes)
  }
}
