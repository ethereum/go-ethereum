/*
The MIT License

Copyright (c) 2016 Nick Dodson. nickdodson.com

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE
 */

import { bytesToUnprefixedHex, utf8ToBytes } from './bytes.js'

/**
 * Returns a `Boolean` on whether or not the a `String` starts with '0x'
 * @param str the string input value
 * @return a boolean if it is or is not hex prefixed
 * @throws if the str input is not a string
 */
export function isHexPrefixed(str: string): boolean {
  if (typeof str !== 'string') {
    throw new Error(`[isHexPrefixed] input must be type 'string', received type ${typeof str}`)
  }

  return str[0] === '0' && str[1] === 'x'
}

/**
 * Removes '0x' from a given `String` if present
 * @param str the string value
 * @returns the string without 0x prefix
 */
export const stripHexPrefix = (str: string): string => {
  if (typeof str !== 'string')
    throw new Error(`[stripHexPrefix] input must be type 'string', received ${typeof str}`)

  return isHexPrefixed(str) ? str.slice(2) : str
}

/**
 * Pads a `String` to have an even length
 * @param value
 * @return output
 */
export function padToEven(value: string): string {
  let a = value

  if (typeof a !== 'string') {
    throw new Error(`[padToEven] value must be type 'string', received ${typeof a}`)
  }

  if (a.length % 2) a = `0${a}`

  return a
}

/**
 * Get the binary size of a string
 * @param str
 * @returns the number of bytes contained within the string
 */
export function getBinarySize(str: string) {
  if (typeof str !== 'string') {
    throw new Error(`[getBinarySize] method requires input type 'string', received ${typeof str}`)
  }

  return utf8ToBytes(str).byteLength
}

/**
 * Returns TRUE if the first specified array contains all elements
 * from the second one. FALSE otherwise.
 *
 * @param superset
 * @param subset
 *
 */
export function arrayContainsArray(
  superset: unknown[],
  subset: unknown[],
  some?: boolean
): boolean {
  if (Array.isArray(superset) !== true) {
    throw new Error(
      `[arrayContainsArray] method requires input 'superset' to be an array, got type '${typeof superset}'`
    )
  }
  if (Array.isArray(subset) !== true) {
    throw new Error(
      `[arrayContainsArray] method requires input 'subset' to be an array, got type '${typeof subset}'`
    )
  }

  return subset[some === true ? 'some' : 'every']((value) => superset.indexOf(value) >= 0)
}

/**
 * Should be called to get ascii from its hex representation
 *
 * @param string in hex
 * @returns ascii string representation of hex value
 */
export function toAscii(hex: string): string {
  let str = ''
  let i = 0
  const l = hex.length

  if (hex.substring(0, 2) === '0x') i = 2

  for (; i < l; i += 2) {
    const code = parseInt(hex.substr(i, 2), 16)
    str += String.fromCharCode(code)
  }

  return str
}

/**
 * Should be called to get hex representation (prefixed by 0x) of utf8 string.
 * Strips leading and trailing 0's.
 *
 * @param string
 * @param optional padding
 * @returns hex representation of input string
 */
export function fromUtf8(stringValue: string) {
  const str = utf8ToBytes(stringValue)

  return `0x${padToEven(bytesToUnprefixedHex(str)).replace(/^0+|0+$/g, '')}`
}

/**
 * Should be called to get hex representation (prefixed by 0x) of ascii string
 *
 * @param  string
 * @param  optional padding
 * @returns  hex representation of input string
 */
export function fromAscii(stringValue: string) {
  let hex = ''
  for (let i = 0; i < stringValue.length; i++) {
    const code = stringValue.charCodeAt(i)
    const n = code.toString(16)
    hex += n.length < 2 ? `0${n}` : n
  }

  return `0x${hex}`
}

/**
 * Returns the keys from an array of objects.
 * @example
 * ```js
 * getKeys([{a: '1', b: '2'}, {a: '3', b: '4'}], 'a') => ['1', '3']
 *````
 * @param  params
 * @param  key
 * @param  allowEmpty
 * @returns output just a simple array of output keys
 */
export function getKeys(params: Record<string, string>[], key: string, allowEmpty?: boolean) {
  if (!Array.isArray(params)) {
    throw new Error(`[getKeys] method expects input 'params' to be an array, got ${typeof params}`)
  }
  if (typeof key !== 'string') {
    throw new Error(
      `[getKeys] method expects input 'key' to be type 'string', got ${typeof params}`
    )
  }

  const result = []

  for (let i = 0; i < params.length; i++) {
    let value = params[i][key]
    if (allowEmpty === true && !value) {
      value = ''
    } else if (typeof value !== 'string') {
      throw new Error(`invalid abi - expected type 'string', received ${typeof value}`)
    }
    result.push(value)
  }

  return result
}

/**
 * Is the string a hex string.
 *
 * @param  value
 * @param  length
 * @returns  output the string is a hex string
 */
export function isHexString(value: string, length?: number): boolean {
  if (typeof value !== 'string' || !value.match(/^0x[0-9A-Fa-f]*$/)) return false

  if (typeof length !== 'undefined' && length > 0 && value.length !== 2 + 2 * length) return false

  return true
}
