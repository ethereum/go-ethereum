"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.computeHmac = exports.sha512 = exports.sha256 = exports.ripemd160 = void 0;
var hash_js_1 = __importDefault(require("hash.js"));
//const _ripemd160 = _hash.ripemd160;
var bytes_1 = require("@ethersproject/bytes");
var types_1 = require("./types");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
function ripemd160(data) {
    return "0x" + (hash_js_1.default.ripemd160().update((0, bytes_1.arrayify)(data)).digest("hex"));
}
exports.ripemd160 = ripemd160;
function sha256(data) {
    return "0x" + (hash_js_1.default.sha256().update((0, bytes_1.arrayify)(data)).digest("hex"));
}
exports.sha256 = sha256;
function sha512(data) {
    return "0x" + (hash_js_1.default.sha512().update((0, bytes_1.arrayify)(data)).digest("hex"));
}
exports.sha512 = sha512;
function computeHmac(algorithm, key, data) {
    if (!types_1.SupportedAlgorithm[algorithm]) {
        logger.throwError("unsupported algorithm " + algorithm, logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
            operation: "hmac",
            algorithm: algorithm
        });
    }
    return "0x" + hash_js_1.default.hmac(hash_js_1.default[algorithm], (0, bytes_1.arrayify)(key)).update((0, bytes_1.arrayify)(data)).digest("hex");
}
exports.computeHmac = computeHmac;
//# sourceMappingURL=browser-sha2.js.map