"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.pbkdf2 = pbkdf2;
exports.pbkdf2Sync = pbkdf2Sync;
const pbkdf2_1 = require("@noble/hashes/pbkdf2");
const sha256_1 = require("@noble/hashes/sha256");
const sha512_1 = require("@noble/hashes/sha512");
const utils_js_1 = require("./utils.js");
async function pbkdf2(password, salt, iterations, keylen, digest) {
    if (!["sha256", "sha512"].includes(digest)) {
        throw new Error("Only sha256 and sha512 are supported");
    }
    (0, utils_js_1.assertBytes)(password);
    (0, utils_js_1.assertBytes)(salt);
    return (0, pbkdf2_1.pbkdf2Async)(digest === "sha256" ? sha256_1.sha256 : sha512_1.sha512, password, salt, {
        c: iterations,
        dkLen: keylen
    });
}
function pbkdf2Sync(password, salt, iterations, keylen, digest) {
    if (!["sha256", "sha512"].includes(digest)) {
        throw new Error("Only sha256 and sha512 are supported");
    }
    (0, utils_js_1.assertBytes)(password);
    (0, utils_js_1.assertBytes)(salt);
    return (0, pbkdf2_1.pbkdf2)(digest === "sha256" ? sha256_1.sha256 : sha512_1.sha512, password, salt, {
        c: iterations,
        dkLen: keylen
    });
}
