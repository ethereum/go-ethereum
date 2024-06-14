"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.rlphash = exports.ripemd160 = exports.sha256 = exports.keccak256 = exports.keccak = void 0;
var _a = require('ethereum-cryptography/keccak'), keccak224 = _a.keccak224, keccak384 = _a.keccak384, k256 = _a.keccak256, keccak512 = _a.keccak512;
var createHash = require('create-hash');
var ethjsUtil = require('ethjs-util');
var rlp = require("rlp");
var bytes_1 = require("./bytes");
/**
 * Creates Keccak hash of the input
 * @param a The input data (Buffer|Array|String|Number) If the string is a 0x-prefixed hex value
 * it's interpreted as hexadecimal, otherwise as utf8.
 * @param bits The Keccak width
 */
exports.keccak = function (a, bits) {
    if (bits === void 0) { bits = 256; }
    if (typeof a === 'string' && !ethjsUtil.isHexString(a)) {
        a = Buffer.from(a, 'utf8');
    }
    else {
        a = bytes_1.toBuffer(a);
    }
    if (!bits)
        bits = 256;
    switch (bits) {
        case 224: {
            return keccak224(a);
        }
        case 256: {
            return k256(a);
        }
        case 384: {
            return keccak384(a);
        }
        case 512: {
            return keccak512(a);
        }
        default: {
            throw new Error("Invald algorithm: keccak" + bits);
        }
    }
};
/**
 * Creates Keccak-256 hash of the input, alias for keccak(a, 256).
 * @param a The input data (Buffer|Array|String|Number)
 */
exports.keccak256 = function (a) {
    return exports.keccak(a);
};
/**
 * Creates SHA256 hash of the input.
 * @param a The input data (Buffer|Array|String|Number)
 */
exports.sha256 = function (a) {
    a = bytes_1.toBuffer(a);
    return createHash('sha256')
        .update(a)
        .digest();
};
/**
 * Creates RIPEMD160 hash of the input.
 * @param a The input data (Buffer|Array|String|Number)
 * @param padded Whether it should be padded to 256 bits or not
 */
exports.ripemd160 = function (a, padded) {
    a = bytes_1.toBuffer(a);
    var hash = createHash('rmd160')
        .update(a)
        .digest();
    if (padded === true) {
        return bytes_1.setLength(hash, 32);
    }
    else {
        return hash;
    }
};
/**
 * Creates SHA-3 hash of the RLP encoded version of the input.
 * @param a The input data
 */
exports.rlphash = function (a) {
    return exports.keccak(rlp.encode(a));
};
//# sourceMappingURL=hash.js.map