import { getRandomBytesSync } from 'ethereum-cryptography/random.js'
// eslint-disable-next-line no-restricted-imports
import { bytesToHex as _bytesToUnprefixedHex } from 'ethereum-cryptography/utils.js'

import { assertIsArray, assertIsBytes, assertIsHexString } from './helpers.js'
import { isHexString, padToEven, stripHexPrefix } from './internal.js'

import type { PrefixedHexString, TransformabletoBytes } from './types.js'

const BIGINT_0 = BigInt(0)

/**
 * @deprecated
 */
export const bytesToUnprefixedHex = _bytesToUnprefixedHex

// hexToBytes cache
const hexToBytesMapFirstKey: { [key: string]: number } = {}
const hexToBytesMapSecondKey: { [key: string]: number } = {}

for (let i = 0; i < 16; i++) {
  const vSecondKey = i
  const vFirstKey = i * 16
  const key = i.toString(16).toLowerCase()
  hexToBytesMapSecondKey[key] = vSecondKey
  hexToBytesMapSecondKey[key.toUpperCase()] = vSecondKey
  hexToBytesMapFirstKey[key] = vFirstKey
  hexToBytesMapFirstKey[key.toUpperCase()] = vFirstKey
}

/**
 * NOTE: only use this function if the string is even, and only consists of hex characters
 * If this is not the case, this function could return weird results
 * @deprecated
 */
function _unprefixedHexToBytes(hex: string): Uint8Array {
  const byteLen = hex.length
  const bytes = new Uint8Array(byteLen / 2)
  for (let i = 0; i < byteLen; i += 2) {
    bytes[i / 2] = hexToBytesMapFirstKey[hex[i]] + hexToBytesMapSecondKey[hex[i + 1]]
  }
  return bytes
}

/**
 * @deprecated
 */
export const unprefixedHexToBytes = (inp: string) => {
  if (inp.slice(0, 2) === '0x') {
    throw new Error('hex string is prefixed with 0x, should be unprefixed')
  } else {
    return _unprefixedHexToBytes(padToEven(inp))
  }
}

/****************  Borrowed from @chainsafe/ssz */
// Caching this info costs about ~1000 bytes and speeds up toHexString() by x6
const hexByByte = Array.from({ length: 256 }, (v, i) => i.toString(16).padStart(2, '0'))

export const bytesToHex = (bytes: Uint8Array): PrefixedHexString => {
  let hex: PrefixedHexString = `0x`
  if (bytes === undefined || bytes.length === 0) return hex
  for (const byte of bytes) {
    hex = `${hex}${hexByByte[byte]}`
  }
  return hex
}

// BigInt cache for the numbers 0 - 256*256-1 (two-byte bytes)
const BIGINT_CACHE: bigint[] = []
for (let i = 0; i <= 256 * 256 - 1; i++) {
  BIGINT_CACHE[i] = BigInt(i)
}

/**
 * Converts a {@link Uint8Array} to a {@link bigint}
 * @param {Uint8Array} bytes the bytes to convert
 * @returns {bigint}
 */
export const bytesToBigInt = (bytes: Uint8Array, littleEndian = false): bigint => {
  if (littleEndian) {
    bytes.reverse()
  }
  const hex = bytesToHex(bytes)
  if (hex === '0x') {
    return BIGINT_0
  }
  if (hex.length === 4) {
    // If the byte length is 1 (this is faster than checking `bytes.length === 1`)
    return BIGINT_CACHE[bytes[0]]
  }
  if (hex.length === 6) {
    return BIGINT_CACHE[bytes[0] * 256 + bytes[1]]
  }
  return BigInt(hex)
}

/**
 * Converts a {@link Uint8Array} to a {@link number}.
 * @param {Uint8Array} bytes the bytes to convert
 * @return  {number}
 * @throws If the input number exceeds 53 bits.
 */
export const bytesToInt = (bytes: Uint8Array): number => {
  const res = Number(bytesToBigInt(bytes))
  if (!Number.isSafeInteger(res)) throw new Error('Number exceeds 53 bits')
  return res
}

// TODO: Restrict the input type to only PrefixedHexString
/**
 * Converts a {@link PrefixedHexString} to a {@link Uint8Array}
 * @param {PrefixedHexString | string} hex The 0x-prefixed hex string to convert
 * @returns {Uint8Array} The converted bytes
 * @throws If the input is not a valid 0x-prefixed hex string
 */
export const hexToBytes = (hex: PrefixedHexString | string): Uint8Array => {
  if (typeof hex !== 'string') {
    throw new Error(`hex argument type ${typeof hex} must be of type string`)
  }

  if (!/^0x[0-9a-fA-F]*$/.test(hex)) {
    throw new Error(`Input must be a 0x-prefixed hexadecimal string, got ${hex}`)
  }

  const unprefixedHex = hex.slice(2)

  return _unprefixedHexToBytes(
    unprefixedHex.length % 2 === 0 ? unprefixedHex : padToEven(unprefixedHex)
  )
}

/******************************************/

/**
 * Converts a {@link number} into a {@link PrefixedHexString}
 * @param {number} i
 * @return {PrefixedHexString}
 */
export const intToHex = (i: number): PrefixedHexString => {
  if (!Number.isSafeInteger(i) || i < 0) {
    throw new Error(`Received an invalid integer type: ${i}`)
  }
  return `0x${i.toString(16)}`
}

/**
 * Converts an {@link number} to a {@link Uint8Array}
 * @param {Number} i
 * @return {Uint8Array}
 */
export const intToBytes = (i: number): Uint8Array => {
  const hex = intToHex(i)
  return hexToBytes(hex)
}

/**
 * Converts a {@link bigint} to a {@link Uint8Array}
 *  * @param {bigint} num the bigint to convert
 * @returns {Uint8Array}
 */
export const bigIntToBytes = (num: bigint, littleEndian = false): Uint8Array => {
  // eslint-disable-next-line @typescript-eslint/no-use-before-define
  const bytes = toBytes(`0x${padToEven(num.toString(16))}`)

  return littleEndian ? bytes.reverse() : bytes
}

/**
 * Returns a Uint8Array filled with 0s.
 * @param {number} bytes the number of bytes of the Uint8Array
 * @return {Uint8Array}
 */
export const zeros = (bytes: number): Uint8Array => {
  return new Uint8Array(bytes)
}

/**
 * Pads a `Uint8Array` with zeros till it has `length` bytes.
 * Truncates the beginning or end of input if its length exceeds `length`.
 * @param {Uint8Array} msg the value to pad
 * @param {number} length the number of bytes the output should be
 * @param {boolean} right whether to start padding form the left or right
 * @return {Uint8Array}
 */
const setLength = (msg: Uint8Array, length: number, right: boolean): Uint8Array => {
  if (right) {
    if (msg.length < length) {
      return new Uint8Array([...msg, ...zeros(length - msg.length)])
    }
    return msg.subarray(0, length)
  } else {
    if (msg.length < length) {
      return new Uint8Array([...zeros(length - msg.length), ...msg])
    }
    return msg.subarray(-length)
  }
}

/**
 * Left Pads a `Uint8Array` with leading zeros till it has `length` bytes.
 * Or it truncates the beginning if it exceeds.
 * @param {Uint8Array} msg the value to pad
 * @param {number} length the number of bytes the output should be
 * @return {Uint8Array}
 */
export const setLengthLeft = (msg: Uint8Array, length: number): Uint8Array => {
  assertIsBytes(msg)
  return setLength(msg, length, false)
}

/**
 * Right Pads a `Uint8Array` with trailing zeros till it has `length` bytes.
 * it truncates the end if it exceeds.
 * @param {Uint8Array} msg the value to pad
 * @param {number} length the number of bytes the output should be
 * @return {Uint8Array}
 */
export const setLengthRight = (msg: Uint8Array, length: number): Uint8Array => {
  assertIsBytes(msg)
  return setLength(msg, length, true)
}

/**
 * Trims leading zeros from a `Uint8Array`, `number[]` or `string`.
 * @param {Uint8Array|number[]|string} a
 * @return {Uint8Array|number[]|string}
 */
const stripZeros = <T extends Uint8Array | number[] | string = Uint8Array | number[] | string>(
  a: T
): T => {
  let first = a[0]
  while (a.length > 0 && first.toString() === '0') {
    a = a.slice(1) as T
    first = a[0]
  }
  return a
}

/**
 * Trims leading zeros from a `Uint8Array`.
 * @param {Uint8Array} a
 * @return {Uint8Array}
 */
export const unpadBytes = (a: Uint8Array): Uint8Array => {
  assertIsBytes(a)
  return stripZeros(a)
}

/**
 * Trims leading zeros from an `Array` (of numbers).
 * @param  {number[]} a
 * @return {number[]}
 */
export const unpadArray = (a: number[]): number[] => {
  assertIsArray(a)
  return stripZeros(a)
}

// TODO: Restrict the input type to only PrefixedHexString
/**
 * Trims leading zeros from a `PrefixedHexString`.
 * @param {PrefixedHexString | string} a
 * @return {PrefixedHexString}
 */
export const unpadHex = (a: PrefixedHexString | string): PrefixedHexString => {
  assertIsHexString(a)
  return `0x${stripZeros(stripHexPrefix(a))}`
}

// TODO: remove the string type from this function (only keep PrefixedHexString)
export type ToBytesInputTypes =
  | PrefixedHexString
  | string
  | number
  | bigint
  | Uint8Array
  | number[]
  | TransformabletoBytes
  | null
  | undefined

/**
 * Attempts to turn a value into a `Uint8Array`.
 * Inputs supported: `Buffer`, `Uint8Array`, `String` (hex-prefixed), `Number`, null/undefined, `BigInt` and other objects
 * with a `toArray()` or `toBytes()` method.
 * @param {ToBytesInputTypes} v the value
 * @return {Uint8Array}
 */

export const toBytes = (v: ToBytesInputTypes): Uint8Array => {
  if (v === null || v === undefined) {
    return new Uint8Array()
  }

  if (Array.isArray(v) || v instanceof Uint8Array) {
    return Uint8Array.from(v)
  }

  if (typeof v === 'string') {
    if (!isHexString(v)) {
      throw new Error(
        `Cannot convert string to Uint8Array. toBytes only supports 0x-prefixed hex strings and this string was given: ${v}`
      )
    }
    return hexToBytes(v)
  }

  if (typeof v === 'number') {
    return intToBytes(v)
  }

  if (typeof v === 'bigint') {
    if (v < BIGINT_0) {
      throw new Error(`Cannot convert negative bigint to Uint8Array. Given: ${v}`)
    }
    let n = v.toString(16)
    if (n.length % 2) n = '0' + n
    return unprefixedHexToBytes(n)
  }

  if (v.toBytes !== undefined) {
    // converts a `TransformableToBytes` object to a Uint8Array
    return v.toBytes()
  }

  throw new Error('invalid type')
}

/**
 * Interprets a `Uint8Array` as a signed integer and returns a `BigInt`. Assumes 256-bit numbers.
 * @param {Uint8Array} num Signed integer value
 * @returns {bigint}
 */
export const fromSigned = (num: Uint8Array): bigint => {
  return BigInt.asIntN(256, bytesToBigInt(num))
}

/**
 * Converts a `BigInt` to an unsigned integer and returns it as a `Uint8Array`. Assumes 256-bit numbers.
 * @param {bigint} num
 * @returns {Uint8Array}
 */
export const toUnsigned = (num: bigint): Uint8Array => {
  return bigIntToBytes(BigInt.asUintN(256, num))
}

/**
 * Adds "0x" to a given `string` if it does not already start with "0x".
 * @param {string} str
 * @return {PrefixedHexString}
 */
export const addHexPrefix = (str: string): PrefixedHexString => {
  if (typeof str !== 'string') {
    return str
  }

  return isHexString(str) ? str : `0x${str}`
}

/**
 * Shortens a string  or Uint8Array's hex string representation to maxLength (default 50).
 *
 * Examples:
 *
 * Input:  '657468657265756d000000000000000000000000000000000000000000000000'
 * Output: '657468657265756d0000000000000000000000000000000000…'
 * @param {Uint8Array | string} bytes
 * @param {number} maxLength
 * @return {string}
 */
export const short = (bytes: Uint8Array | string, maxLength: number = 50): string => {
  const byteStr = bytes instanceof Uint8Array ? bytesToHex(bytes) : bytes
  const len = byteStr.slice(0, 2) === '0x' ? maxLength + 2 : maxLength
  if (byteStr.length <= len) {
    return byteStr
  }
  return byteStr.slice(0, len) + '…'
}

/**
 * Checks provided Uint8Array for leading zeroes and throws if found.
 *
 * Examples:
 *
 * Valid values: 0x1, 0x, 0x01, 0x1234
 * Invalid values: 0x0, 0x00, 0x001, 0x0001
 *
 * Note: This method is useful for validating that RLP encoded integers comply with the rule that all
 * integer values encoded to RLP must be in the most compact form and contain no leading zero bytes
 * @param values An object containing string keys and Uint8Array values
 * @throws if any provided value is found to have leading zero bytes
 */
export const validateNoLeadingZeroes = (values: { [key: string]: Uint8Array | undefined }) => {
  for (const [k, v] of Object.entries(values)) {
    if (v !== undefined && v.length > 0 && v[0] === 0) {
      throw new Error(`${k} cannot have leading zeroes, received: ${bytesToHex(v)}`)
    }
  }
}

/**
 * Converts a {@link bigint} to a `0x` prefixed hex string
 * @param {bigint} num the bigint to convert
 * @returns {PrefixedHexString}
 */
export const bigIntToHex = (num: bigint): PrefixedHexString => {
  return `0x${num.toString(16)}`
}

/**
 * Calculates max bigint from an array of bigints
 * @param args array of bigints
 */
export const bigIntMax = (...args: bigint[]) => args.reduce((m, e) => (e > m ? e : m))

/**
 * Calculates min BigInt from an array of BigInts
 * @param args array of bigints
 */
export const bigIntMin = (...args: bigint[]) => args.reduce((m, e) => (e < m ? e : m))

/**
 * Convert value from bigint to an unpadded Uint8Array
 * (useful for RLP transport)
 * @param {bigint} value the bigint to convert
 * @returns {Uint8Array}
 */
export const bigIntToUnpaddedBytes = (value: bigint): Uint8Array => {
  return unpadBytes(bigIntToBytes(value))
}

export const bigIntToAddressBytes = (value: bigint, strict: boolean = true): Uint8Array => {
  const addressBytes = bigIntToBytes(value)
  if (strict && addressBytes.length > 20) {
    throw Error(`Invalid address bytes length=${addressBytes.length} strict=${strict}`)
  }

  // setLength already slices if more than requisite length
  return setLengthLeft(addressBytes, 20)
}

/**
 * Convert value from number to an unpadded Uint8Array
 * (useful for RLP transport)
 * @param {number} value the bigint to convert
 * @returns {Uint8Array}
 */
export const intToUnpaddedBytes = (value: number): Uint8Array => {
  return unpadBytes(intToBytes(value))
}

/**
 * Compares two Uint8Arrays and returns a number indicating their order in a sorted array.
 *
 * @param {Uint8Array} value1 - The first Uint8Array to compare.
 * @param {Uint8Array} value2 - The second Uint8Array to compare.
 * @returns {number} A positive number if value1 is larger than value2,
 *                   A negative number if value1 is smaller than value2,
 *                   or 0 if value1 and value2 are equal.
 */
export const compareBytes = (value1: Uint8Array, value2: Uint8Array): number => {
  const bigIntValue1 = bytesToBigInt(value1)
  const bigIntValue2 = bytesToBigInt(value2)
  return bigIntValue1 > bigIntValue2 ? 1 : bigIntValue1 < bigIntValue2 ? -1 : 0
}

/**
 * Generates a Uint8Array of random bytes of specified length.
 *
 * @param {number} length - The length of the Uint8Array.
 * @returns {Uint8Array} A Uint8Array of random bytes of specified length.
 */
export const randomBytes = (length: number): Uint8Array => {
  return getRandomBytesSync(length)
}

/**
 * This mirrors the functionality of the `ethereum-cryptography` export except
 * it skips the check to validate that every element of `arrays` is indead a `uint8Array`
 * Can give small performance gains on large arrays
 * @param {Uint8Array[]} arrays an array of Uint8Arrays
 * @returns {Uint8Array} one Uint8Array with all the elements of the original set
 * works like `Buffer.concat`
 */
export const concatBytes = (...arrays: Uint8Array[]): Uint8Array => {
  if (arrays.length === 1) return arrays[0]
  const length = arrays.reduce((a, arr) => a + arr.length, 0)
  const result = new Uint8Array(length)
  for (let i = 0, pad = 0; i < arrays.length; i++) {
    const arr = arrays[i]
    result.set(arr, pad)
    pad += arr.length
  }
  return result
}

/**
 * @notice Convert a Uint8Array to a 32-bit integer
 * @param {Uint8Array} bytes The input Uint8Array from which to read the 32-bit integer.
 * @param {boolean} littleEndian True for little-endian, undefined or false for big-endian.
 * @return {number} The 32-bit integer read from the input Uint8Array.
 */
export function bytesToInt32(bytes: Uint8Array, littleEndian: boolean = false): number {
  if (bytes.length < 4) {
    bytes = setLength(bytes, 4, littleEndian)
  }
  const dataView = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
  return dataView.getUint32(0, littleEndian)
}

/**
 * @notice Convert a Uint8Array to a 64-bit bigint
 * @param {Uint8Array} bytes The input Uint8Array from which to read the 64-bit bigint.
 * @param {boolean} littleEndian True for little-endian, undefined or false for big-endian.
 * @return {bigint} The 64-bit bigint read from the input Uint8Array.
 */
export function bytesToBigInt64(bytes: Uint8Array, littleEndian: boolean = false): bigint {
  if (bytes.length < 8) {
    bytes = setLength(bytes, 8, littleEndian)
  }
  const dataView = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
  return dataView.getBigUint64(0, littleEndian)
}

/**
 * @notice Convert a 32-bit integer to a Uint8Array.
 * @param {number} value The 32-bit integer to convert.
 * @param {boolean} littleEndian True for little-endian, undefined or false for big-endian.
 * @return {Uint8Array} A Uint8Array of length 4 containing the integer.
 */
export function int32ToBytes(value: number, littleEndian: boolean = false): Uint8Array {
  const buffer = new ArrayBuffer(4)
  const dataView = new DataView(buffer)
  dataView.setUint32(0, value, littleEndian)
  return new Uint8Array(buffer)
}

/**
 * @notice Convert a 64-bit bigint to a Uint8Array.
 * @param {bigint} value The 64-bit bigint to convert.
 * @param {boolean} littleEndian True for little-endian, undefined or false for big-endian.
 * @return {Uint8Array} A Uint8Array of length 8 containing the bigint.
 */
export function bigInt64ToBytes(value: bigint, littleEndian: boolean = false): Uint8Array {
  const buffer = new ArrayBuffer(8)
  const dataView = new DataView(buffer)
  dataView.setBigUint64(0, value, littleEndian)
  return new Uint8Array(buffer)
}

// eslint-disable-next-line no-restricted-imports
export { bytesToUtf8, equalsBytes, utf8ToBytes } from 'ethereum-cryptography/utils.js'

// TODO: Restrict the input type to only PrefixedHexString
export function hexToBigInt(input: PrefixedHexString | string): bigint {
  return bytesToBigInt(hexToBytes(isHexString(input) ? input : `0x${input}`))
}
