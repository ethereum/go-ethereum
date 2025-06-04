"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.uuidV4 = exports.searchPath = exports.getPassword = exports.zpad = exports.looseArrayify = void 0;
var bytes_1 = require("@ethersproject/bytes");
var strings_1 = require("@ethersproject/strings");
function looseArrayify(hexString) {
    if (typeof (hexString) === 'string' && hexString.substring(0, 2) !== '0x') {
        hexString = '0x' + hexString;
    }
    return (0, bytes_1.arrayify)(hexString);
}
exports.looseArrayify = looseArrayify;
function zpad(value, length) {
    value = String(value);
    while (value.length < length) {
        value = '0' + value;
    }
    return value;
}
exports.zpad = zpad;
function getPassword(password) {
    if (typeof (password) === 'string') {
        return (0, strings_1.toUtf8Bytes)(password, strings_1.UnicodeNormalizationForm.NFKC);
    }
    return (0, bytes_1.arrayify)(password);
}
exports.getPassword = getPassword;
function searchPath(object, path) {
    var currentChild = object;
    var comps = path.toLowerCase().split('/');
    for (var i = 0; i < comps.length; i++) {
        // Search for a child object with a case-insensitive matching key
        var matchingChild = null;
        for (var key in currentChild) {
            if (key.toLowerCase() === comps[i]) {
                matchingChild = currentChild[key];
                break;
            }
        }
        // Didn't find one. :'(
        if (matchingChild === null) {
            return null;
        }
        // Now check this child...
        currentChild = matchingChild;
    }
    return currentChild;
}
exports.searchPath = searchPath;
// See: https://www.ietf.org/rfc/rfc4122.txt (Section 4.4)
function uuidV4(randomBytes) {
    var bytes = (0, bytes_1.arrayify)(randomBytes);
    // Section: 4.1.3:
    // - time_hi_and_version[12:16] = 0b0100
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    // Section 4.4
    // - clock_seq_hi_and_reserved[6] = 0b0
    // - clock_seq_hi_and_reserved[7] = 0b1
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    var value = (0, bytes_1.hexlify)(bytes);
    return [
        value.substring(2, 10),
        value.substring(10, 14),
        value.substring(14, 18),
        value.substring(18, 22),
        value.substring(22, 34),
    ].join("-");
}
exports.uuidV4 = uuidV4;
//# sourceMappingURL=utils.js.map