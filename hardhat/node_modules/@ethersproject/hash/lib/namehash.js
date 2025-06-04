"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.dnsEncode = exports.namehash = exports.isValidName = exports.ensNormalize = void 0;
var bytes_1 = require("@ethersproject/bytes");
var strings_1 = require("@ethersproject/strings");
var keccak256_1 = require("@ethersproject/keccak256");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var lib_1 = require("./ens-normalize/lib");
var Zeros = new Uint8Array(32);
Zeros.fill(0);
function checkComponent(comp) {
    if (comp.length === 0) {
        throw new Error("invalid ENS name; empty component");
    }
    return comp;
}
function ensNameSplit(name) {
    var bytes = (0, strings_1.toUtf8Bytes)((0, lib_1.ens_normalize)(name));
    var comps = [];
    if (name.length === 0) {
        return comps;
    }
    var last = 0;
    for (var i = 0; i < bytes.length; i++) {
        var d = bytes[i];
        // A separator (i.e. "."); copy this component
        if (d === 0x2e) {
            comps.push(checkComponent(bytes.slice(last, i)));
            last = i + 1;
        }
    }
    // There was a stray separator at the end of the name
    if (last >= bytes.length) {
        throw new Error("invalid ENS name; empty component");
    }
    comps.push(checkComponent(bytes.slice(last)));
    return comps;
}
function ensNormalize(name) {
    return ensNameSplit(name).map(function (comp) { return (0, strings_1.toUtf8String)(comp); }).join(".");
}
exports.ensNormalize = ensNormalize;
function isValidName(name) {
    try {
        return (ensNameSplit(name).length !== 0);
    }
    catch (error) { }
    return false;
}
exports.isValidName = isValidName;
function namehash(name) {
    /* istanbul ignore if */
    if (typeof (name) !== "string") {
        logger.throwArgumentError("invalid ENS name; not a string", "name", name);
    }
    var result = Zeros;
    var comps = ensNameSplit(name);
    while (comps.length) {
        result = (0, keccak256_1.keccak256)((0, bytes_1.concat)([result, (0, keccak256_1.keccak256)(comps.pop())]));
    }
    return (0, bytes_1.hexlify)(result);
}
exports.namehash = namehash;
function dnsEncode(name) {
    return (0, bytes_1.hexlify)((0, bytes_1.concat)(ensNameSplit(name).map(function (comp) {
        // DNS does not allow components over 63 bytes in length
        if (comp.length > 63) {
            throw new Error("invalid DNS encoded entry; length exceeds 63 bytes");
        }
        var bytes = new Uint8Array(comp.length + 1);
        bytes.set(comp, 1);
        bytes[0] = bytes.length - 1;
        return bytes;
    }))) + "00";
}
exports.dnsEncode = dnsEncode;
//# sourceMappingURL=namehash.js.map