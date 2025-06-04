"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.MAX_WITHDRAWALS_PER_PAYLOAD = exports.RLP_EMPTY_STRING = exports.KECCAK256_RLP = exports.KECCAK256_RLP_S = exports.KECCAK256_RLP_ARRAY = exports.KECCAK256_RLP_ARRAY_S = exports.KECCAK256_NULL = exports.KECCAK256_NULL_S = exports.TWO_POW256 = exports.SECP256K1_ORDER_DIV_2 = exports.SECP256K1_ORDER = exports.MAX_INTEGER_BIGINT = exports.MAX_INTEGER = exports.MAX_UINT64 = void 0;
const buffer_1 = require("buffer");
const secp256k1_1 = require("ethereum-cryptography/secp256k1");
/**
 * 2^64-1
 */
exports.MAX_UINT64 = BigInt('0xffffffffffffffff');
/**
 * The max integer that the evm can handle (2^256-1)
 */
exports.MAX_INTEGER = BigInt('0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff');
/**
 * The max integer that the evm can handle (2^256-1) as a bigint
 * 2^256-1 equals to 340282366920938463463374607431768211455
 * We use literal value instead of calculated value for compatibility issue.
 */
exports.MAX_INTEGER_BIGINT = BigInt('115792089237316195423570985008687907853269984665640564039457584007913129639935');
exports.SECP256K1_ORDER = secp256k1_1.secp256k1.CURVE.n;
exports.SECP256K1_ORDER_DIV_2 = secp256k1_1.secp256k1.CURVE.n / BigInt(2);
/**
 * 2^256
 */
exports.TWO_POW256 = BigInt('0x10000000000000000000000000000000000000000000000000000000000000000');
/**
 * Keccak-256 hash of null
 */
exports.KECCAK256_NULL_S = 'c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470';
/**
 * Keccak-256 hash of null
 */
exports.KECCAK256_NULL = buffer_1.Buffer.from(exports.KECCAK256_NULL_S, 'hex');
/**
 * Keccak-256 of an RLP of an empty array
 */
exports.KECCAK256_RLP_ARRAY_S = '1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347';
/**
 * Keccak-256 of an RLP of an empty array
 */
exports.KECCAK256_RLP_ARRAY = buffer_1.Buffer.from(exports.KECCAK256_RLP_ARRAY_S, 'hex');
/**
 * Keccak-256 hash of the RLP of null
 */
exports.KECCAK256_RLP_S = '56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421';
/**
 * Keccak-256 hash of the RLP of null
 */
exports.KECCAK256_RLP = buffer_1.Buffer.from(exports.KECCAK256_RLP_S, 'hex');
/**
 *  RLP encoded empty string
 */
exports.RLP_EMPTY_STRING = buffer_1.Buffer.from([0x80]);
exports.MAX_WITHDRAWALS_PER_PAYLOAD = 16;
//# sourceMappingURL=constants.js.map