import {
  generateAddress,
  generateAddress2,
  isValidAddress,
  privateToAddress,
  pubToAddress,
} from './account'
import { bigIntToBuffer, bufferToBigInt, toBuffer, zeros } from './bytes'

/**
 * Handling and generating Ethereum addresses
 */
export class Address {
  public readonly buf: Buffer

  constructor(buf: Buffer) {
    if (buf.length !== 20) {
      throw new Error('Invalid address length')
    }
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
    if (!isValidAddress(str)) {
      throw new Error('Invalid address')
    }
    return new Address(toBuffer(str))
  }

  /**
   * Returns an address for a given public key.
   * @param pubKey The two points of an uncompressed key
   */
  static fromPublicKey(pubKey: Buffer): Address {
    if (!Buffer.isBuffer(pubKey)) {
      throw new Error('Public key should be Buffer')
    }
    const buf = pubToAddress(pubKey)
    return new Address(buf)
  }

  /**
   * Returns an address for a given private key.
   * @param privateKey A private key must be 256 bits wide
   */
  static fromPrivateKey(privateKey: Buffer): Address {
    if (!Buffer.isBuffer(privateKey)) {
      throw new Error('Private key should be Buffer')
    }
    const buf = privateToAddress(privateKey)
    return new Address(buf)
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
    return new Address(generateAddress(from.buf, bigIntToBuffer(nonce)))
  }

  /**
   * Generates an address for a contract created using CREATE2.
   * @param from The address which is creating this new address
   * @param salt A salt
   * @param initCode The init code of the contract being created
   */
  static generate2(from: Address, salt: Buffer, initCode: Buffer): Address {
    if (!Buffer.isBuffer(salt)) {
      throw new Error('Expected salt to be a Buffer')
    }
    if (!Buffer.isBuffer(initCode)) {
      throw new Error('Expected initCode to be a Buffer')
    }
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
    const address = bufferToBigInt(this.buf)
    const rangeMin = BigInt(0)
    const rangeMax = BigInt('0xffff')
    return address >= rangeMin && address <= rangeMax
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
