"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.BIGINT_2EXP256 = exports.BIGINT_2EXP224 = exports.BIGINT_2EXP160 = exports.BIGINT_2EXP96 = exports.BIGINT_224 = exports.BIGINT_160 = exports.BIGINT_100 = exports.BIGINT_96 = exports.BIGINT_256 = exports.BIGINT_255 = exports.BIGINT_128 = exports.BIGINT_64 = exports.BIGINT_32 = exports.BIGINT_31 = exports.BIGINT_28 = exports.BIGINT_27 = exports.BIGINT_8 = exports.BIGINT_7 = exports.BIGINT_3 = exports.BIGINT_2 = exports.BIGINT_1 = exports.BIGINT_0 = exports.BIGINT_NEG1 = exports.RIPEMD160_ADDRESS_STRING = exports.MAX_WITHDRAWALS_PER_PAYLOAD = exports.RLP_EMPTY_STRING = exports.KECCAK256_RLP = exports.KECCAK256_RLP_S = exports.KECCAK256_RLP_ARRAY = exports.KECCAK256_RLP_ARRAY_S = exports.KECCAK256_NULL = exports.KECCAK256_NULL_S = exports.TWO_POW256 = exports.SECP256K1_ORDER_DIV_2 = exports.SECP256K1_ORDER = exports.MAX_INTEGER_BIGINT = exports.MAX_INTEGER = exports.MAX_UINT64 = void 0;
const bytes_js_1 = require("./bytes.js");
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
exports.SECP256K1_ORDER = BigInt('0xfffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141');
exports.SECP256K1_ORDER_DIV_2 = exports.SECP256K1_ORDER / BigInt(2);
/**
 * 2^256
 */
exports.TWO_POW256 = BigInt('0x10000000000000000000000000000000000000000000000000000000000000000');
/**
 * Keccak-256 hash of null
 */
exports.KECCAK256_NULL_S = '0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470';
/**
 * Keccak-256 hash of null
 */
exports.KECCAK256_NULL = (0, bytes_js_1.hexToBytes)(exports.KECCAK256_NULL_S);
/**
 * Keccak-256 of an RLP of an empty array
 */
exports.KECCAK256_RLP_ARRAY_S = '0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347';
/**
 * Keccak-256 of an RLP of an empty array
 */
exports.KECCAK256_RLP_ARRAY = (0, bytes_js_1.hexToBytes)(exports.KECCAK256_RLP_ARRAY_S);
/**
 * Keccak-256 hash of the RLP of null
 */
exports.KECCAK256_RLP_S = '0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421';
/**
 * Keccak-256 hash of the RLP of null
 */
exports.KECCAK256_RLP = (0, bytes_js_1.hexToBytes)(exports.KECCAK256_RLP_S);
/**
 *  RLP encoded empty string
 */
exports.RLP_EMPTY_STRING = Uint8Array.from([0x80]);
exports.MAX_WITHDRAWALS_PER_PAYLOAD = 16;
exports.RIPEMD160_ADDRESS_STRING = '0000000000000000000000000000000000000003';
/**
 * BigInt constants
 */
exports.BIGINT_NEG1 = BigInt(-1);
exports.BIGINT_0 = BigInt(0);
exports.BIGINT_1 = BigInt(1);
exports.BIGINT_2 = BigInt(2);
exports.BIGINT_3 = BigInt(3);
exports.BIGINT_7 = BigInt(7);
exports.BIGINT_8 = BigInt(8);
exports.BIGINT_27 = BigInt(27);
exports.BIGINT_28 = BigInt(28);
exports.BIGINT_31 = BigInt(31);
exports.BIGINT_32 = BigInt(32);
exports.BIGINT_64 = BigInt(64);
exports.BIGINT_128 = BigInt(128);
exports.BIGINT_255 = BigInt(255);
exports.BIGINT_256 = BigInt(256);
exports.BIGINT_96 = BigInt(96);
exports.BIGINT_100 = BigInt(100);
exports.BIGINT_160 = BigInt(160);
exports.BIGINT_224 = BigInt(224);
exports.BIGINT_2EXP96 = BigInt(79228162514264337593543950336);
exports.BIGINT_2EXP160 = BigInt(1461501637330902918203684832716283019655932542976);
exports.BIGINT_2EXP224 = BigInt(26959946667150639794667015087019630673637144422540572481103610249216);
exports.BIGINT_2EXP256 = exports.BIGINT_2 ** exports.BIGINT_256;
//# sourceMappingURL=constants.js.map