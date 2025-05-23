"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getCreate2Address = exports.getContractAddress = exports.getIcapAddress = exports.isAddress = exports.getAddress = void 0;
var bytes_1 = require("@ethersproject/bytes");
var bignumber_1 = require("@ethersproject/bignumber");
var keccak256_1 = require("@ethersproject/keccak256");
var rlp_1 = require("@ethersproject/rlp");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
function getChecksumAddress(address) {
    if (!(0, bytes_1.isHexString)(address, 20)) {
        logger.throwArgumentError("invalid address", "address", address);
    }
    address = address.toLowerCase();
    var chars = address.substring(2).split("");
    var expanded = new Uint8Array(40);
    for (var i = 0; i < 40; i++) {
        expanded[i] = chars[i].charCodeAt(0);
    }
    var hashed = (0, bytes_1.arrayify)((0, keccak256_1.keccak256)(expanded));
    for (var i = 0; i < 40; i += 2) {
        if ((hashed[i >> 1] >> 4) >= 8) {
            chars[i] = chars[i].toUpperCase();
        }
        if ((hashed[i >> 1] & 0x0f) >= 8) {
            chars[i + 1] = chars[i + 1].toUpperCase();
        }
    }
    return "0x" + chars.join("");
}
// Shims for environments that are missing some required constants and functions
var MAX_SAFE_INTEGER = 0x1fffffffffffff;
function log10(x) {
    if (Math.log10) {
        return Math.log10(x);
    }
    return Math.log(x) / Math.LN10;
}
// See: https://en.wikipedia.org/wiki/International_Bank_Account_Number
// Create lookup table
var ibanLookup = {};
for (var i = 0; i < 10; i++) {
    ibanLookup[String(i)] = String(i);
}
for (var i = 0; i < 26; i++) {
    ibanLookup[String.fromCharCode(65 + i)] = String(10 + i);
}
// How many decimal digits can we process? (for 64-bit float, this is 15)
var safeDigits = Math.floor(log10(MAX_SAFE_INTEGER));
function ibanChecksum(address) {
    address = address.toUpperCase();
    address = address.substring(4) + address.substring(0, 2) + "00";
    var expanded = address.split("").map(function (c) { return ibanLookup[c]; }).join("");
    // Javascript can handle integers safely up to 15 (decimal) digits
    while (expanded.length >= safeDigits) {
        var block = expanded.substring(0, safeDigits);
        expanded = parseInt(block, 10) % 97 + expanded.substring(block.length);
    }
    var checksum = String(98 - (parseInt(expanded, 10) % 97));
    while (checksum.length < 2) {
        checksum = "0" + checksum;
    }
    return checksum;
}
;
function getAddress(address) {
    var result = null;
    if (typeof (address) !== "string") {
        logger.throwArgumentError("invalid address", "address", address);
    }
    if (address.match(/^(0x)?[0-9a-fA-F]{40}$/)) {
        // Missing the 0x prefix
        if (address.substring(0, 2) !== "0x") {
            address = "0x" + address;
        }
        result = getChecksumAddress(address);
        // It is a checksummed address with a bad checksum
        if (address.match(/([A-F].*[a-f])|([a-f].*[A-F])/) && result !== address) {
            logger.throwArgumentError("bad address checksum", "address", address);
        }
        // Maybe ICAP? (we only support direct mode)
    }
    else if (address.match(/^XE[0-9]{2}[0-9A-Za-z]{30,31}$/)) {
        // It is an ICAP address with a bad checksum
        if (address.substring(2, 4) !== ibanChecksum(address)) {
            logger.throwArgumentError("bad icap checksum", "address", address);
        }
        result = (0, bignumber_1._base36To16)(address.substring(4));
        while (result.length < 40) {
            result = "0" + result;
        }
        result = getChecksumAddress("0x" + result);
    }
    else {
        logger.throwArgumentError("invalid address", "address", address);
    }
    return result;
}
exports.getAddress = getAddress;
function isAddress(address) {
    try {
        getAddress(address);
        return true;
    }
    catch (error) { }
    return false;
}
exports.isAddress = isAddress;
function getIcapAddress(address) {
    var base36 = (0, bignumber_1._base16To36)(getAddress(address).substring(2)).toUpperCase();
    while (base36.length < 30) {
        base36 = "0" + base36;
    }
    return "XE" + ibanChecksum("XE00" + base36) + base36;
}
exports.getIcapAddress = getIcapAddress;
// http://ethereum.stackexchange.com/questions/760/how-is-the-address-of-an-ethereum-contract-computed
function getContractAddress(transaction) {
    var from = null;
    try {
        from = getAddress(transaction.from);
    }
    catch (error) {
        logger.throwArgumentError("missing from address", "transaction", transaction);
    }
    var nonce = (0, bytes_1.stripZeros)((0, bytes_1.arrayify)(bignumber_1.BigNumber.from(transaction.nonce).toHexString()));
    return getAddress((0, bytes_1.hexDataSlice)((0, keccak256_1.keccak256)((0, rlp_1.encode)([from, nonce])), 12));
}
exports.getContractAddress = getContractAddress;
function getCreate2Address(from, salt, initCodeHash) {
    if ((0, bytes_1.hexDataLength)(salt) !== 32) {
        logger.throwArgumentError("salt must be 32 bytes", "salt", salt);
    }
    if ((0, bytes_1.hexDataLength)(initCodeHash) !== 32) {
        logger.throwArgumentError("initCodeHash must be 32 bytes", "initCodeHash", initCodeHash);
    }
    return getAddress((0, bytes_1.hexDataSlice)((0, keccak256_1.keccak256)((0, bytes_1.concat)(["0xff", getAddress(from), salt, initCodeHash])), 12));
}
exports.getCreate2Address = getCreate2Address;
//# sourceMappingURL=index.js.map