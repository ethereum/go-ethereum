import { BN } from './externals'
import { isHexString } from './internal'
import { Address } from './address'
import { unpadBuffer, toBuffer, ToBufferInputTypes } from './bytes'

/*
 * A type that represents a BNLike input that can be converted to a BN.
 */
export type BNLike = BN | PrefixedHexString | number | Buffer

/*
 * A type that represents a BufferLike input that can be converted to a Buffer.
 */
export type BufferLike =
  | Buffer
  | Uint8Array
  | number[]
  | number
  | BN
  | TransformableToBuffer
  | PrefixedHexString

/*
 * A type that represents a `0x`-prefixed hex string.
 */
export type PrefixedHexString = string

/**
 * A type that represents an Address-like value.
 * To convert to address, use `new Address(toBuffer(value))`
 */
export type AddressLike = Address | Buffer | PrefixedHexString

/*
 * A type that represents an object that has a `toArray()` method.
 */
export interface TransformableToArray {
  toArray(): Uint8Array
  toBuffer?(): Buffer
}

/*
 * A type that represents an object that has a `toBuffer()` method.
 */
export interface TransformableToBuffer {
  toBuffer(): Buffer
  toArray?(): Uint8Array
}

export type NestedUint8Array = Array<Uint8Array | NestedUint8Array>
export type NestedBufferArray = Array<Buffer | NestedBufferArray>

/**
 * Convert BN to 0x-prefixed hex string.
 */
export function bnToHex(value: BN): PrefixedHexString {
  return `0x${value.toString(16)}`
}

/**
 * Convert value from BN to an unpadded Buffer
 * (useful for RLP transport)
 * @param value value to convert
 */
export function bnToUnpaddedBuffer(value: BN): Buffer {
  // Using `bn.toArrayLike(Buffer)` instead of `bn.toBuffer()`
  // for compatibility with browserify and similar tools
  return unpadBuffer(value.toArrayLike(Buffer))
}

/**
 * Deprecated alias for {@link bnToUnpaddedBuffer}
 * @deprecated
 */
export function bnToRlp(value: BN): Buffer {
  return bnToUnpaddedBuffer(value)
}

/**
 * Type output options
 */
export enum TypeOutput {
  Number,
  BN,
  Buffer,
  PrefixedHexString,
}

export type TypeOutputReturnType = {
  [TypeOutput.Number]: number
  [TypeOutput.BN]: BN
  [TypeOutput.Buffer]: Buffer
  [TypeOutput.PrefixedHexString]: PrefixedHexString
}

/**
 * Convert an input to a specified type.
 * Input of null/undefined returns null/undefined regardless of the output type.
 * @param input value to convert
 * @param outputType type to output
 */
export function toType<T extends TypeOutput>(input: null, outputType: T): null
export function toType<T extends TypeOutput>(input: undefined, outputType: T): undefined
export function toType<T extends TypeOutput>(
  input: ToBufferInputTypes,
  outputType: T
): TypeOutputReturnType[T]
export function toType<T extends TypeOutput>(
  input: ToBufferInputTypes,
  outputType: T
): TypeOutputReturnType[T] | undefined | null {
  if (input === null) {
    return null
  }
  if (input === undefined) {
    return undefined
  }

  if (typeof input === 'string' && !isHexString(input)) {
    throw new Error(`A string must be provided with a 0x-prefix, given: ${input}`)
  } else if (typeof input === 'number' && !Number.isSafeInteger(input)) {
    throw new Error(
      'The provided number is greater than MAX_SAFE_INTEGER (please use an alternative input type)'
    )
  }

  const output = toBuffer(input)

  if (outputType === TypeOutput.Buffer) {
    return output as TypeOutputReturnType[T]
  } else if (outputType === TypeOutput.BN) {
    return new BN(output) as TypeOutputReturnType[T]
  } else if (outputType === TypeOutput.Number) {
    const bn = new BN(output)
    const max = new BN(Number.MAX_SAFE_INTEGER.toString())
    if (bn.gt(max)) {
      throw new Error(
        'The provided number is greater than MAX_SAFE_INTEGER (please use an alternative output type)'
      )
    }
    return bn.toNumber() as TypeOutputReturnType[T]
  } else {
    // outputType === TypeOutput.PrefixedHexString
    return `0x${output.toString('hex')}` as TypeOutputReturnType[T]
  }
}
