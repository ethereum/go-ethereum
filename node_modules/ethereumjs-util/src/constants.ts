import { Buffer } from 'buffer'
import { BN } from './externals'

/**
 * 2^64-1
 */
export const MAX_UINT64 = new BN('ffffffffffffffff', 16)

/**
 * The max integer that the evm can handle (2^256-1)
 */
export const MAX_INTEGER = new BN(
  'ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff',
  16
)

/**
 * 2^256
 */
export const TWO_POW256 = new BN(
  '10000000000000000000000000000000000000000000000000000000000000000',
  16
)

/**
 * Keccak-256 hash of null
 */
export const KECCAK256_NULL_S = 'c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470'

/**
 * Keccak-256 hash of null
 */
export const KECCAK256_NULL = Buffer.from(KECCAK256_NULL_S, 'hex')

/**
 * Keccak-256 of an RLP of an empty array
 */
export const KECCAK256_RLP_ARRAY_S =
  '1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347'

/**
 * Keccak-256 of an RLP of an empty array
 */
export const KECCAK256_RLP_ARRAY = Buffer.from(KECCAK256_RLP_ARRAY_S, 'hex')

/**
 * Keccak-256 hash of the RLP of null
 */
export const KECCAK256_RLP_S = '56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421'

/**
 * Keccak-256 hash of the RLP of null
 */
export const KECCAK256_RLP = Buffer.from(KECCAK256_RLP_S, 'hex')
