"use strict";
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (Object.hasOwnProperty.call(mod, k)) result[k] = mod[k];
    result["default"] = mod;
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
var secp256k1 = __importStar(require("secp256k1"));
function privateKeyVerify(privateKey) {
    return secp256k1.privateKeyVerify(privateKey);
}
exports.privateKeyVerify = privateKeyVerify;
function publicKeyCreate(privateKey, compressed) {
    if (compressed === void 0) { compressed = true; }
    return Buffer.from(secp256k1.publicKeyCreate(privateKey, compressed));
}
exports.publicKeyCreate = publicKeyCreate;
function publicKeyVerify(publicKey) {
    return secp256k1.publicKeyVerify(publicKey);
}
exports.publicKeyVerify = publicKeyVerify;
function publicKeyConvert(publicKey, compressed) {
    if (compressed === void 0) { compressed = true; }
    return Buffer.from(secp256k1.publicKeyConvert(publicKey, compressed));
}
exports.publicKeyConvert = publicKeyConvert;
function privateKeyTweakAdd(publicKey, tweak) {
    return Buffer.from(secp256k1.privateKeyTweakAdd(Buffer.from(publicKey), tweak));
}
exports.privateKeyTweakAdd = privateKeyTweakAdd;
function publicKeyTweakAdd(publicKey, tweak, compressed) {
    if (compressed === void 0) { compressed = true; }
    return Buffer.from(secp256k1.publicKeyTweakAdd(Buffer.from(publicKey), tweak, compressed));
}
exports.publicKeyTweakAdd = publicKeyTweakAdd;
function sign(message, privateKey) {
    var ret = secp256k1.ecdsaSign(message, privateKey);
    return { signature: Buffer.from(ret.signature), recovery: ret.recid };
}
exports.sign = sign;
function verify(message, signature, publicKey) {
    return secp256k1.ecdsaVerify(signature, message, publicKey);
}
exports.verify = verify;
//# sourceMappingURL=hdkey-secp256k1v3.js.map