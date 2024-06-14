"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var browserifyAes = require("browserify-aes");
var SUPPORTED_MODES = ["aes-128-ctr", "aes-128-cbc", "aes-256-cbc"];
function ensureAesMode(mode) {
    if (!mode.startsWith("aes-")) {
        throw new Error("AES submodule doesn't support mode " + mode);
    }
}
function warnIfUnsuportedMode(mode) {
    if (!SUPPORTED_MODES.includes(mode)) {
        // tslint:disable-next-line no-console
        console.warn("Using an unsupported AES mode. Consider using aes-128-ctr.");
    }
}
function encrypt(msg, key, iv, mode, pkcs7PaddingEnabled) {
    if (mode === void 0) { mode = "aes-128-ctr"; }
    if (pkcs7PaddingEnabled === void 0) { pkcs7PaddingEnabled = true; }
    ensureAesMode(mode);
    var cipher = browserifyAes.createCipheriv(mode, key, iv);
    cipher.setAutoPadding(pkcs7PaddingEnabled);
    var encrypted = cipher.update(msg);
    var final = cipher.final();
    return Buffer.concat([encrypted, final]);
}
exports.encrypt = encrypt;
function decrypt(cypherText, key, iv, mode, pkcs7PaddingEnabled) {
    if (mode === void 0) { mode = "aes-128-ctr"; }
    if (pkcs7PaddingEnabled === void 0) { pkcs7PaddingEnabled = true; }
    ensureAesMode(mode);
    var decipher = browserifyAes.createDecipheriv(mode, key, iv);
    decipher.setAutoPadding(pkcs7PaddingEnabled);
    var encrypted = decipher.update(cypherText);
    var final = decipher.final();
    return Buffer.concat([encrypted, final]);
}
exports.decrypt = decrypt;
//# sourceMappingURL=aes.js.map