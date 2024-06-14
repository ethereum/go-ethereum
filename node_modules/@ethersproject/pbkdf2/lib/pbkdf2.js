"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.pbkdf2 = void 0;
var crypto_1 = require("crypto");
var bytes_1 = require("@ethersproject/bytes");
function bufferify(value) {
    return Buffer.from((0, bytes_1.arrayify)(value));
}
function pbkdf2(password, salt, iterations, keylen, hashAlgorithm) {
    return (0, bytes_1.hexlify)((0, crypto_1.pbkdf2Sync)(bufferify(password), bufferify(salt), iterations, keylen, hashAlgorithm));
}
exports.pbkdf2 = pbkdf2;
//# sourceMappingURL=pbkdf2.js.map