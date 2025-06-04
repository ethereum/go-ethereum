import { isHexString } from './internal'

/**
 * Throws if a string is not hex prefixed
 * @param {string} input string to check hex prefix of
 */
export const assertIsHexString = function (input: string): void {
  if (!isHexString(input)) {
    const msg = `This method only supports 0x-prefixed hex strings but input was: ${input}`
    throw new Error(msg)
  }
}

/**
 * Throws if input is not a buffer
 * @param {Buffer} input value to check
 */
export const assertIsBuffer = function (input: Buffer): void {
  if (!Buffer.isBuffer(input)) {
    const msg = `This method only supports Buffer but input was: ${input}`
    throw new Error(msg)
  }
}

/**
 * Throws if input is not an array
 * @param {number[]} input value to check
 */
export const assertIsArray = function (input: number[]): void {
  if (!Array.isArray(input)) {
    const msg = `This method only supports number arrays but input was: ${input}`
    throw new Error(msg)
  }
}

/**
 * Throws if input is not a string
 * @param {string} input value to check
 */
export const assertIsString = function (input: string): void {
  if (typeof input !== 'string') {
    const msg = `This method only supports strings but input was: ${input}`
    throw new Error(msg)
  }
}
