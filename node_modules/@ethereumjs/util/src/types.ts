import { bufferToBigInt, bufferToHex, toBuffer } from './bytes'
import { isHexString } from './internal'

import type { Address } from './address'
import type { ToBufferInputTypes } from './bytes'

/*
 * A type that represents an input that can be converted to a BigInt.
 */
export type BigIntLike = bigint | PrefixedHexString | number | Buffer

/*
 * A type that represents an input that can be converted to a Buffer.
 */
export type BufferLike =
  | Buffer
  | Uint8Array
  | number[]
  | number
  | bigint
  | TransformableToBuffer
  | PrefixedHexString

/*
 * A type that represents a `0x`-prefixed hex string.
 */
export type PrefixedHexString = string

/**
 * A type that represents an input that can be converted to an Address.
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
 * Type output options
 */
export enum TypeOutput {
  Number,
  BigInt,
  Buffer,
  PrefixedHexString,
}

export type TypeOutputReturnType = {
  [TypeOutput.Number]: number
  [TypeOutput.BigInt]: bigint
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

  switch (outputType) {
    case TypeOutput.Buffer:
      return output as TypeOutputReturnType[T]
    case TypeOutput.BigInt:
      return bufferToBigInt(output) as TypeOutputReturnType[T]
    case TypeOutput.Number: {
      const bigInt = bufferToBigInt(output)
      if (bigInt > BigInt(Number.MAX_SAFE_INTEGER)) {
        throw new Error(
          'The provided number is greater than MAX_SAFE_INTEGER (please use an alternative output type)'
        )
      }
      return Number(bigInt) as TypeOutputReturnType[T]
    }
    case TypeOutput.PrefixedHexString:
      return bufferToHex(output) as TypeOutputReturnType[T]
    default:
      throw new Error('unknown outputType')
  }
}
