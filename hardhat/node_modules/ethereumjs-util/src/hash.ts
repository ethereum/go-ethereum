import { keccak224, keccak384, keccak256 as k256, keccak512 } from 'ethereum-cryptography/keccak'
const createHash = require('create-hash')
import { rlp } from './externals'
import { toBuffer, setLengthLeft } from './bytes'
import { assertIsString, assertIsBuffer, assertIsArray, assertIsHexString } from './helpers'

/**
 * Creates Keccak hash of a Buffer input
 * @param a The input data (Buffer)
 * @param bits (number = 256) The Keccak width
 */
export const keccak = function (a: Buffer, bits: number = 256): Buffer {
  assertIsBuffer(a)
  switch (bits) {
    case 224: {
      return keccak224(a)
    }
    case 256: {
      return k256(a)
    }
    case 384: {
      return keccak384(a)
    }
    case 512: {
      return keccak512(a)
    }
    default: {
      throw new Error(`Invald algorithm: keccak${bits}`)
    }
  }
}

/**
 * Creates Keccak-256 hash of the input, alias for keccak(a, 256).
 * @param a The input data (Buffer)
 */
export const keccak256 = function (a: Buffer): Buffer {
  return keccak(a)
}

/**
 * Creates Keccak hash of a utf-8 string input
 * @param a The input data (String)
 * @param bits (number = 256) The Keccak width
 */
export const keccakFromString = function (a: string, bits: number = 256) {
  assertIsString(a)
  const buf = Buffer.from(a, 'utf8')
  return keccak(buf, bits)
}

/**
 * Creates Keccak hash of an 0x-prefixed string input
 * @param a The input data (String)
 * @param bits (number = 256) The Keccak width
 */
export const keccakFromHexString = function (a: string, bits: number = 256) {
  assertIsHexString(a)
  return keccak(toBuffer(a), bits)
}

/**
 * Creates Keccak hash of a number array input
 * @param a The input data (number[])
 * @param bits (number = 256) The Keccak width
 */
export const keccakFromArray = function (a: number[], bits: number = 256) {
  assertIsArray(a)
  return keccak(toBuffer(a), bits)
}

/**
 * Creates SHA256 hash of an input.
 * @param  a The input data (Buffer|Array|String)
 */
const _sha256 = function (a: any): Buffer {
  a = toBuffer(a)
  return createHash('sha256').update(a).digest()
}

/**
 * Creates SHA256 hash of a Buffer input.
 * @param a The input data (Buffer)
 */
export const sha256 = function (a: Buffer): Buffer {
  assertIsBuffer(a)
  return _sha256(a)
}

/**
 * Creates SHA256 hash of a string input.
 * @param a The input data (string)
 */
export const sha256FromString = function (a: string): Buffer {
  assertIsString(a)
  return _sha256(a)
}

/**
 * Creates SHA256 hash of a number[] input.
 * @param a The input data (number[])
 */
export const sha256FromArray = function (a: number[]): Buffer {
  assertIsArray(a)
  return _sha256(a)
}

/**
 * Creates RIPEMD160 hash of the input.
 * @param a The input data (Buffer|Array|String|Number)
 * @param padded Whether it should be padded to 256 bits or not
 */
const _ripemd160 = function (a: any, padded: boolean): Buffer {
  a = toBuffer(a)
  const hash = createHash('rmd160').update(a).digest()
  if (padded === true) {
    return setLengthLeft(hash, 32)
  } else {
    return hash
  }
}

/**
 * Creates RIPEMD160 hash of a Buffer input.
 * @param a The input data (Buffer)
 * @param padded Whether it should be padded to 256 bits or not
 */
export const ripemd160 = function (a: Buffer, padded: boolean): Buffer {
  assertIsBuffer(a)
  return _ripemd160(a, padded)
}

/**
 * Creates RIPEMD160 hash of a string input.
 * @param a The input data (String)
 * @param padded Whether it should be padded to 256 bits or not
 */
export const ripemd160FromString = function (a: string, padded: boolean): Buffer {
  assertIsString(a)
  return _ripemd160(a, padded)
}

/**
 * Creates RIPEMD160 hash of a number[] input.
 * @param a The input data (number[])
 * @param padded Whether it should be padded to 256 bits or not
 */
export const ripemd160FromArray = function (a: number[], padded: boolean): Buffer {
  assertIsArray(a)
  return _ripemd160(a, padded)
}

/**
 * Creates SHA-3 hash of the RLP encoded version of the input.
 * @param a The input data
 */
export const rlphash = function (a: rlp.Input): Buffer {
  return keccak(rlp.encode(a))
}
