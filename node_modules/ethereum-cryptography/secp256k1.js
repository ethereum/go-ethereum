"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.schnorr = exports.Signature = exports.Point = exports.CURVE = exports.utils = exports.getSharedSecret = exports.recoverPublicKey = exports.verify = exports.signSync = exports.sign = exports.getPublicKey = void 0;
const hmac_1 = require("@noble/hashes/hmac");
const sha256_1 = require("@noble/hashes/sha256");
const secp256k1_1 = require("@noble/secp256k1");
var secp256k1_2 = require("@noble/secp256k1");
Object.defineProperty(exports, "getPublicKey", { enumerable: true, get: function () { return secp256k1_2.getPublicKey; } });
Object.defineProperty(exports, "sign", { enumerable: true, get: function () { return secp256k1_2.sign; } });
Object.defineProperty(exports, "signSync", { enumerable: true, get: function () { return secp256k1_2.signSync; } });
Object.defineProperty(exports, "verify", { enumerable: true, get: function () { return secp256k1_2.verify; } });
Object.defineProperty(exports, "recoverPublicKey", { enumerable: true, get: function () { return secp256k1_2.recoverPublicKey; } });
Object.defineProperty(exports, "getSharedSecret", { enumerable: true, get: function () { return secp256k1_2.getSharedSecret; } });
Object.defineProperty(exports, "utils", { enumerable: true, get: function () { return secp256k1_2.utils; } });
Object.defineProperty(exports, "CURVE", { enumerable: true, get: function () { return secp256k1_2.CURVE; } });
Object.defineProperty(exports, "Point", { enumerable: true, get: function () { return secp256k1_2.Point; } });
Object.defineProperty(exports, "Signature", { enumerable: true, get: function () { return secp256k1_2.Signature; } });
Object.defineProperty(exports, "schnorr", { enumerable: true, get: function () { return secp256k1_2.schnorr; } });
// Enable sync API for noble-secp256k1
secp256k1_1.utils.hmacSha256Sync = (key, ...messages) => {
    const h = hmac_1.hmac.create(sha256_1.sha256, key);
    messages.forEach(msg => h.update(msg));
    return h.digest();
};
