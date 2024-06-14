"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.decode = exports.encode = void 0;
//See: https://github.com/ethereum/wiki/wiki/RLP
var bytes_1 = require("@ethersproject/bytes");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
function arrayifyInteger(value) {
    var result = [];
    while (value) {
        result.unshift(value & 0xff);
        value >>= 8;
    }
    return result;
}
function unarrayifyInteger(data, offset, length) {
    var result = 0;
    for (var i = 0; i < length; i++) {
        result = (result * 256) + data[offset + i];
    }
    return result;
}
function _encode(object) {
    if (Array.isArray(object)) {
        var payload_1 = [];
        object.forEach(function (child) {
            payload_1 = payload_1.concat(_encode(child));
        });
        if (payload_1.length <= 55) {
            payload_1.unshift(0xc0 + payload_1.length);
            return payload_1;
        }
        var length_1 = arrayifyInteger(payload_1.length);
        length_1.unshift(0xf7 + length_1.length);
        return length_1.concat(payload_1);
    }
    if (!(0, bytes_1.isBytesLike)(object)) {
        logger.throwArgumentError("RLP object must be BytesLike", "object", object);
    }
    var data = Array.prototype.slice.call((0, bytes_1.arrayify)(object));
    if (data.length === 1 && data[0] <= 0x7f) {
        return data;
    }
    else if (data.length <= 55) {
        data.unshift(0x80 + data.length);
        return data;
    }
    var length = arrayifyInteger(data.length);
    length.unshift(0xb7 + length.length);
    return length.concat(data);
}
function encode(object) {
    return (0, bytes_1.hexlify)(_encode(object));
}
exports.encode = encode;
function _decodeChildren(data, offset, childOffset, length) {
    var result = [];
    while (childOffset < offset + 1 + length) {
        var decoded = _decode(data, childOffset);
        result.push(decoded.result);
        childOffset += decoded.consumed;
        if (childOffset > offset + 1 + length) {
            logger.throwError("child data too short", logger_1.Logger.errors.BUFFER_OVERRUN, {});
        }
    }
    return { consumed: (1 + length), result: result };
}
// returns { consumed: number, result: Object }
function _decode(data, offset) {
    if (data.length === 0) {
        logger.throwError("data too short", logger_1.Logger.errors.BUFFER_OVERRUN, {});
    }
    // Array with extra length prefix
    if (data[offset] >= 0xf8) {
        var lengthLength = data[offset] - 0xf7;
        if (offset + 1 + lengthLength > data.length) {
            logger.throwError("data short segment too short", logger_1.Logger.errors.BUFFER_OVERRUN, {});
        }
        var length_2 = unarrayifyInteger(data, offset + 1, lengthLength);
        if (offset + 1 + lengthLength + length_2 > data.length) {
            logger.throwError("data long segment too short", logger_1.Logger.errors.BUFFER_OVERRUN, {});
        }
        return _decodeChildren(data, offset, offset + 1 + lengthLength, lengthLength + length_2);
    }
    else if (data[offset] >= 0xc0) {
        var length_3 = data[offset] - 0xc0;
        if (offset + 1 + length_3 > data.length) {
            logger.throwError("data array too short", logger_1.Logger.errors.BUFFER_OVERRUN, {});
        }
        return _decodeChildren(data, offset, offset + 1, length_3);
    }
    else if (data[offset] >= 0xb8) {
        var lengthLength = data[offset] - 0xb7;
        if (offset + 1 + lengthLength > data.length) {
            logger.throwError("data array too short", logger_1.Logger.errors.BUFFER_OVERRUN, {});
        }
        var length_4 = unarrayifyInteger(data, offset + 1, lengthLength);
        if (offset + 1 + lengthLength + length_4 > data.length) {
            logger.throwError("data array too short", logger_1.Logger.errors.BUFFER_OVERRUN, {});
        }
        var result = (0, bytes_1.hexlify)(data.slice(offset + 1 + lengthLength, offset + 1 + lengthLength + length_4));
        return { consumed: (1 + lengthLength + length_4), result: result };
    }
    else if (data[offset] >= 0x80) {
        var length_5 = data[offset] - 0x80;
        if (offset + 1 + length_5 > data.length) {
            logger.throwError("data too short", logger_1.Logger.errors.BUFFER_OVERRUN, {});
        }
        var result = (0, bytes_1.hexlify)(data.slice(offset + 1, offset + 1 + length_5));
        return { consumed: (1 + length_5), result: result };
    }
    return { consumed: 1, result: (0, bytes_1.hexlify)(data[offset]) };
}
function decode(data) {
    var bytes = (0, bytes_1.arrayify)(data);
    var decoded = _decode(bytes, 0);
    if (decoded.consumed !== bytes.length) {
        logger.throwArgumentError("invalid rlp data", "data", data);
    }
    return decoded.result;
}
exports.decode = decode;
//# sourceMappingURL=index.js.map