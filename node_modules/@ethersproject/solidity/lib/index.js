"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.sha256 = exports.keccak256 = exports.pack = void 0;
var bignumber_1 = require("@ethersproject/bignumber");
var bytes_1 = require("@ethersproject/bytes");
var keccak256_1 = require("@ethersproject/keccak256");
var sha2_1 = require("@ethersproject/sha2");
var strings_1 = require("@ethersproject/strings");
var regexBytes = new RegExp("^bytes([0-9]+)$");
var regexNumber = new RegExp("^(u?int)([0-9]*)$");
var regexArray = new RegExp("^(.*)\\[([0-9]*)\\]$");
var Zeros = "0000000000000000000000000000000000000000000000000000000000000000";
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
function _pack(type, value, isArray) {
    switch (type) {
        case "address":
            if (isArray) {
                return (0, bytes_1.zeroPad)(value, 32);
            }
            return (0, bytes_1.arrayify)(value);
        case "string":
            return (0, strings_1.toUtf8Bytes)(value);
        case "bytes":
            return (0, bytes_1.arrayify)(value);
        case "bool":
            value = (value ? "0x01" : "0x00");
            if (isArray) {
                return (0, bytes_1.zeroPad)(value, 32);
            }
            return (0, bytes_1.arrayify)(value);
    }
    var match = type.match(regexNumber);
    if (match) {
        //let signed = (match[1] === "int")
        var size = parseInt(match[2] || "256");
        if ((match[2] && String(size) !== match[2]) || (size % 8 !== 0) || size === 0 || size > 256) {
            logger.throwArgumentError("invalid number type", "type", type);
        }
        if (isArray) {
            size = 256;
        }
        value = bignumber_1.BigNumber.from(value).toTwos(size);
        return (0, bytes_1.zeroPad)(value, size / 8);
    }
    match = type.match(regexBytes);
    if (match) {
        var size = parseInt(match[1]);
        if (String(size) !== match[1] || size === 0 || size > 32) {
            logger.throwArgumentError("invalid bytes type", "type", type);
        }
        if ((0, bytes_1.arrayify)(value).byteLength !== size) {
            logger.throwArgumentError("invalid value for " + type, "value", value);
        }
        if (isArray) {
            return (0, bytes_1.arrayify)((value + Zeros).substring(0, 66));
        }
        return value;
    }
    match = type.match(regexArray);
    if (match && Array.isArray(value)) {
        var baseType_1 = match[1];
        var count = parseInt(match[2] || String(value.length));
        if (count != value.length) {
            logger.throwArgumentError("invalid array length for " + type, "value", value);
        }
        var result_1 = [];
        value.forEach(function (value) {
            result_1.push(_pack(baseType_1, value, true));
        });
        return (0, bytes_1.concat)(result_1);
    }
    return logger.throwArgumentError("invalid type", "type", type);
}
// @TODO: Array Enum
function pack(types, values) {
    if (types.length != values.length) {
        logger.throwArgumentError("wrong number of values; expected ${ types.length }", "values", values);
    }
    var tight = [];
    types.forEach(function (type, index) {
        tight.push(_pack(type, values[index]));
    });
    return (0, bytes_1.hexlify)((0, bytes_1.concat)(tight));
}
exports.pack = pack;
function keccak256(types, values) {
    return (0, keccak256_1.keccak256)(pack(types, values));
}
exports.keccak256 = keccak256;
function sha256(types, values) {
    return (0, sha2_1.sha256)(pack(types, values));
}
exports.sha256 = sha256;
//# sourceMappingURL=index.js.map