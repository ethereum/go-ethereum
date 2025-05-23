"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.scrypt = scrypt;
exports.scryptSync = scryptSync;
const scrypt_1 = require("@noble/hashes/scrypt");
const utils_js_1 = require("./utils.js");
async function scrypt(password, salt, n, p, r, dkLen, onProgress) {
    (0, utils_js_1.assertBytes)(password);
    (0, utils_js_1.assertBytes)(salt);
    return (0, scrypt_1.scryptAsync)(password, salt, { N: n, r, p, dkLen, onProgress });
}
function scryptSync(password, salt, n, p, r, dkLen, onProgress) {
    (0, utils_js_1.assertBytes)(password);
    (0, utils_js_1.assertBytes)(salt);
    return (0, scrypt_1.scrypt)(password, salt, { N: n, r, p, dkLen, onProgress });
}
