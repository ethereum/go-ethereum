import assert from 'assert'
import { BN } from './externals'
import { toBuffer, zeros } from './bytes'
import {
  isValidAddress,
  pubToAddress,
  privateToAddress,
  generateAddress,
  generateAddress2,
} from './account'

export class Address {
  public readonly buf: Buffer

  constructor(buf: Buffer) {
    assert(buf.length === 20, 'Invalid address length')
    this.buf = buf
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
    assert(isValidAddress(str), 'Invalid address')
    return new Address(toBuffer(str))
  }

  /**
   * Returns an address for a given public key.
   * @param pubKey The two points of an uncompressed key
   */
  static fromPublicKey(pubKey: Buffer): Address {
    assert(Buffer.isBuffer(pubKey), 'Public key should be Buffer')
    const buf = pubToAddress(pubKey)
    return new Address(buf)
  }

  /**
   * Returns an address for a given private key.
   * @param privateKey A private key must be 256 bits wide
   */
  static fromPrivateKey(privateKey: Buffer): Address {
    assert(Buffer.isBuffer(privateKey), 'Private key should be Buffer')
    const buf = privateToAddress(privateKey)
    return new Address(buf)
  }

  /**
   * Generates an address for a newly created contract.
   * @param from The address which is creating this new address
   * @param nonce The nonce of the from account
   */
  static generate(from: Address, nonce: BN): Address {
    assert(BN.isBN(nonce))
    return new Address(generateAddress(from.buf, nonce.toArrayLike(Buffer)))
  }

  /**
   * Generates an address for a contract created using CREATE2.
   * @param from The address which is creating this new address
   * @param salt A salt
   * @param initCode The init code of the contract being created
   */
  static generate2(from: Address, salt: Buffer, initCode: Buffer): Address {
    assert(Buffer.isBuffer(salt))
    assert(Buffer.isBuffer(initCode))
    return new Address(generateAddress2(from.buf, salt, initCode))
  }

  /**
   * Is address equal to another.
   */
  equals(address: Address): boolean {
    return this.buf.equals(address.buf)
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
    const addressBN = new BN(this.buf)
    const rangeMin = new BN(0)
    const rangeMax = new BN('ffff', 'hex')

    return addressBN.gte(rangeMin) && addressBN.lte(rangeMax)
  }

  /**
   * Returns hex encoding of address.
   */
  toString(): string {
    return '0x' + this.buf.toString('hex')
  }

  /**
   * Returns Buffer representation of address.
   */
  toBuffer(): Buffer {
    return Buffer.from(this.buf)
  }
}
