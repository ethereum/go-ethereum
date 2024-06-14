"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.scryptSync = exports.scrypt = void 0;
const scrypt_1 = require("@noble/hashes/scrypt");
const utils_1 = require("./utils");
async function scrypt(password, salt, n, p, r, dkLen, onProgress) {
    (0, utils_1.assertBytes)(password);
    (0, utils_1.assertBytes)(salt);
    return (0, scrypt_1.scryptAsync)(password, salt, { N: n, r, p, dkLen, onProgress });
}
exports.scrypt = scrypt;
function scryptSync(password, salt, n, p, r, dkLen, onProgress) {
    (0, utils_1.assertBytes)(password);
    (0, utils_1.assertBytes)(salt);
    return (0, scrypt_1.scrypt)(password, salt, { N: n, r, p, dkLen, onProgress });
}
exports.scryptSync = scryptSync;
