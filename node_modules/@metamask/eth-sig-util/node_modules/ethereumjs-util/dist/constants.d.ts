/// <reference types="node" />
import BN = require('bn.js');
/**
 * The max integer that this VM can handle
 */
export declare const MAX_INTEGER: BN;
/**
 * 2^256
 */
export declare const TWO_POW256: BN;
/**
 * Keccak-256 hash of null
 */
export declare const KECCAK256_NULL_S: string;
/**
 * Keccak-256 hash of null
 */
export declare const KECCAK256_NULL: Buffer;
/**
 * Keccak-256 of an RLP of an empty array
 */
export declare const KECCAK256_RLP_ARRAY_S: string;
/**
 * Keccak-256 of an RLP of an empty array
 */
export declare const KECCAK256_RLP_ARRAY: Buffer;
/**
 * Keccak-256 hash of the RLP of null
 */
export declare const KECCAK256_RLP_S: string;
/**
 * Keccak-256 hash of the RLP of null
 */
export declare const KECCAK256_RLP: Buffer;
