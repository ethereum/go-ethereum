import { bytesToBigInt, bytesToHex, toBytes } from './bytes.js'
import { isHexString } from './internal.js'

import type { Address } from './address.js'
import type { ToBytesInputTypes } from './bytes.js'

/*
 * A type that represents an input that can be converted to a BigInt.
 */
export type BigIntLike = bigint | PrefixedHexString | number | Uint8Array

/*
 * A type that represents an input that can be converted to a Uint8Array.
 */
export type BytesLike =
  | Uint8Array
  | number[]
  | number
  | bigint
  | TransformabletoBytes
  | PrefixedHexString

/*
 * A type that represents a `0x`-prefixed hex string.
 */
export type PrefixedHexString = `0x${string}`

/**
 * A type that represents an input that can be converted to an Address.
 */
export type AddressLike = Address | Uint8Array | PrefixedHexString

export interface TransformabletoBytes {
  toBytes?(): Uint8Array
}

export type NestedUint8Array = Array<Uint8Array | NestedUint8Array>

export function isNestedUint8Array(value: unknown): value is NestedUint8Array {
  if (!Array.isArray(value)) {
    return false
  }
  for (const item of value) {
    if (Array.isArray(item)) {
      if (!isNestedUint8Array(item)) {
        return false
      }
    } else if (!(item instanceof Uint8Array)) {
      return false
    }
  }
  return true
}

/**
 * Type output options
 */
export enum TypeOutput {
  Number,
  BigInt,
  Uint8Array,
  PrefixedHexString,
}

export type TypeOutputReturnType = {
  [TypeOutput.Number]: number
  [TypeOutput.BigInt]: bigint
  [TypeOutput.Uint8Array]: Uint8Array
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
  input: ToBytesInputTypes,
  outputType: T
): TypeOutputReturnType[T]
export function toType<T extends TypeOutput>(
  input: ToBytesInputTypes,
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

  const output = toBytes(input)

  switch (outputType) {
    case TypeOutput.Uint8Array:
      return output as TypeOutputReturnType[T]
    case TypeOutput.BigInt:
      return bytesToBigInt(output) as TypeOutputReturnType[T]
    case TypeOutput.Number: {
      const bigInt = bytesToBigInt(output)
      if (bigInt > BigInt(Number.MAX_SAFE_INTEGER)) {
        throw new Error(
          'The provided number is greater than MAX_SAFE_INTEGER (please use an alternative output type)'
        )
      }
      return Number(bigInt) as TypeOutputReturnType[T]
    }
    case TypeOutput.PrefixedHexString:
      return bytesToHex(output) as TypeOutputReturnType[T]
    default:
      throw new Error('unknown outputType')
  }
}
