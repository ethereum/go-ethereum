"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.rlphash = exports.ripemd160FromArray = exports.ripemd160FromString = exports.ripemd160 = exports.sha256FromArray = exports.sha256FromString = exports.sha256 = exports.keccakFromArray = exports.keccakFromHexString = exports.keccakFromString = exports.keccak256 = exports.keccak = void 0;
var keccak_1 = require("ethereum-cryptography/keccak");
var createHash = require('create-hash');
var externals_1 = require("./externals");
var bytes_1 = require("./bytes");
var helpers_1 = require("./helpers");
/**
 * Creates Keccak hash of a Buffer input
 * @param a The input data (Buffer)
 * @param bits (number = 256) The Keccak width
 */
var keccak = function (a, bits) {
    if (bits === void 0) { bits = 256; }
    (0, helpers_1.assertIsBuffer)(a);
    switch (bits) {
        case 224: {
            return (0, keccak_1.keccak224)(a);
        }
        case 256: {
            return (0, keccak_1.keccak256)(a);
        }
        case 384: {
            return (0, keccak_1.keccak384)(a);
        }
        case 512: {
            return (0, keccak_1.keccak512)(a);
        }
        default: {
            throw new Error("Invald algorithm: keccak".concat(bits));
        }
    }
};
exports.keccak = keccak;
/**
 * Creates Keccak-256 hash of the input, alias for keccak(a, 256).
 * @param a The input data (Buffer)
 */
var keccak256 = function (a) {
    return (0, exports.keccak)(a);
};
exports.keccak256 = keccak256;
/**
 * Creates Keccak hash of a utf-8 string input
 * @param a The input data (String)
 * @param bits (number = 256) The Keccak width
 */
var keccakFromString = function (a, bits) {
    if (bits === void 0) { bits = 256; }
    (0, helpers_1.assertIsString)(a);
    var buf = Buffer.from(a, 'utf8');
    return (0, exports.keccak)(buf, bits);
};
exports.keccakFromString = keccakFromString;
/**
 * Creates Keccak hash of an 0x-prefixed string input
 * @param a The input data (String)
 * @param bits (number = 256) The Keccak width
 */
var keccakFromHexString = function (a, bits) {
    if (bits === void 0) { bits = 256; }
    (0, helpers_1.assertIsHexString)(a);
    return (0, exports.keccak)((0, bytes_1.toBuffer)(a), bits);
};
exports.keccakFromHexString = keccakFromHexString;
/**
 * Creates Keccak hash of a number array input
 * @param a The input data (number[])
 * @param bits (number = 256) The Keccak width
 */
var keccakFromArray = function (a, bits) {
    if (bits === void 0) { bits = 256; }
    (0, helpers_1.assertIsArray)(a);
    return (0, exports.keccak)((0, bytes_1.toBuffer)(a), bits);
};
exports.keccakFromArray = keccakFromArray;
/**
 * Creates SHA256 hash of an input.
 * @param  a The input data (Buffer|Array|String)
 */
var _sha256 = function (a) {
    a = (0, bytes_1.toBuffer)(a);
    return createHash('sha256').update(a).digest();
};
/**
 * Creates SHA256 hash of a Buffer input.
 * @param a The input data (Buffer)
 */
var sha256 = function (a) {
    (0, helpers_1.assertIsBuffer)(a);
    return _sha256(a);
};
exports.sha256 = sha256;
/**
 * Creates SHA256 hash of a string input.
 * @param a The input data (string)
 */
var sha256FromString = function (a) {
    (0, helpers_1.assertIsString)(a);
    return _sha256(a);
};
exports.sha256FromString = sha256FromString;
/**
 * Creates SHA256 hash of a number[] input.
 * @param a The input data (number[])
 */
var sha256FromArray = function (a) {
    (0, helpers_1.assertIsArray)(a);
    return _sha256(a);
};
exports.sha256FromArray = sha256FromArray;
/**
 * Creates RIPEMD160 hash of the input.
 * @param a The input data (Buffer|Array|String|Number)
 * @param padded Whether it should be padded to 256 bits or not
 */
var _ripemd160 = function (a, padded) {
    a = (0, bytes_1.toBuffer)(a);
    var hash = createHash('rmd160').update(a).digest();
    if (padded === true) {
        return (0, bytes_1.setLengthLeft)(hash, 32);
    }
    else {
        return hash;
    }
};
/**
 * Creates RIPEMD160 hash of a Buffer input.
 * @param a The input data (Buffer)
 * @param padded Whether it should be padded to 256 bits or not
 */
var ripemd160 = function (a, padded) {
    (0, helpers_1.assertIsBuffer)(a);
    return _ripemd160(a, padded);
};
exports.ripemd160 = ripemd160;
/**
 * Creates RIPEMD160 hash of a string input.
 * @param a The input data (String)
 * @param padded Whether it should be padded to 256 bits or not
 */
var ripemd160FromString = function (a, padded) {
    (0, helpers_1.assertIsString)(a);
    return _ripemd160(a, padded);
};
exports.ripemd160FromString = ripemd160FromString;
/**
 * Creates RIPEMD160 hash of a number[] input.
 * @param a The input data (number[])
 * @param padded Whether it should be padded to 256 bits or not
 */
var ripemd160FromArray = function (a, padded) {
    (0, helpers_1.assertIsArray)(a);
    return _ripemd160(a, padded);
};
exports.ripemd160FromArray = ripemd160FromArray;
/**
 * Creates SHA-3 hash of the RLP encoded version of the input.
 * @param a The input data
 */
var rlphash = function (a) {
    return (0, exports.keccak)(externals_1.rlp.encode(a));
};
exports.rlphash = rlphash;
//# sourceMappingURL=hash.js.map