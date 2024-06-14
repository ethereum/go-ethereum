/// <reference types="bn.js" />
/// <reference types="node" />
import { Buffer } from 'buffer';
import { BN } from './externals';
/**
 * 2^64-1
 */
export declare const MAX_UINT64: BN;
/**
 * The max integer that the evm can handle (2^256-1)
 */
export declare const MAX_INTEGER: BN;
/**
 * 2^256
 */
export declare const TWO_POW256: BN;
/**
 * Keccak-256 hash of null
 */
export declare const KECCAK256_NULL_S = "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470";
/**
 * Keccak-256 hash of null
 */
export declare const KECCAK256_NULL: Buffer;
/**
 * Keccak-256 of an RLP of an empty array
 */
export declare const KECCAK256_RLP_ARRAY_S = "1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347";
/**
 * Keccak-256 of an RLP of an empty array
 */
export declare const KECCAK256_RLP_ARRAY: Buffer;
/**
 * Keccak-256 hash of the RLP of null
 */
export declare const KECCAK256_RLP_S = "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421";
/**
 * Keccak-256 hash of the RLP of null
 */
export declare const KECCAK256_RLP: Buffer;
