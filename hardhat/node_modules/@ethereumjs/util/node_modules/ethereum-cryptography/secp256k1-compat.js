"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createPrivateKeySync = createPrivateKeySync;
exports.createPrivateKey = createPrivateKey;
exports.privateKeyVerify = privateKeyVerify;
exports.publicKeyCreate = publicKeyCreate;
exports.publicKeyVerify = publicKeyVerify;
exports.publicKeyConvert = publicKeyConvert;
exports.ecdsaSign = ecdsaSign;
exports.ecdsaRecover = ecdsaRecover;
exports.ecdsaVerify = ecdsaVerify;
exports.privateKeyTweakAdd = privateKeyTweakAdd;
exports.privateKeyNegate = privateKeyNegate;
exports.publicKeyNegate = publicKeyNegate;
exports.publicKeyCombine = publicKeyCombine;
exports.publicKeyTweakAdd = publicKeyTweakAdd;
exports.publicKeyTweakMul = publicKeyTweakMul;
exports.privateKeyTweakMul = privateKeyTweakMul;
exports.signatureExport = signatureExport;
exports.signatureImport = signatureImport;
exports.signatureNormalize = signatureNormalize;
exports.ecdh = ecdh;
exports.contextRandomize = contextRandomize;
const sha256_1 = require("@noble/hashes/sha256");
const modular_1 = require("@noble/curves/abstract/modular");
const secp256k1_js_1 = require("./secp256k1.js");
const utils_js_1 = require("./utils.js");
// Use `secp256k1` module directly.
// This is a legacy compatibility layer for the npm package `secp256k1` via noble-secp256k1
const Point = secp256k1_js_1.secp256k1.ProjectivePoint;
function hexToNumber(hex) {
    if (typeof hex !== "string") {
        throw new TypeError("hexToNumber: expected string, got " + typeof hex);
    }
    return BigInt(`0x${hex}`);
}
// Copy-paste from secp256k1, maybe export it?
const bytesToNumber = (bytes) => hexToNumber((0, utils_js_1.toHex)(bytes));
const numberToHex = (num) => num.toString(16).padStart(64, "0");
const numberToBytes = (num) => (0, utils_js_1.hexToBytes)(numberToHex(num));
const ORDER = secp256k1_js_1.secp256k1.CURVE.n;
function output(out = (len) => new Uint8Array(len), length, value) {
    if (typeof out === "function") {
        out = out(length);
    }
    (0, utils_js_1.assertBytes)(out, length);
    if (value) {
        out.set(value);
    }
    return out;
}
function getSignature(signature) {
    (0, utils_js_1.assertBytes)(signature, 64);
    return secp256k1_js_1.secp256k1.Signature.fromCompact(signature);
}
function createPrivateKeySync() {
    return secp256k1_js_1.secp256k1.utils.randomPrivateKey();
}
async function createPrivateKey() {
    return createPrivateKeySync();
}
function privateKeyVerify(privateKey) {
    (0, utils_js_1.assertBytes)(privateKey, 32);
    return secp256k1_js_1.secp256k1.utils.isValidPrivateKey(privateKey);
}
function publicKeyCreate(privateKey, compressed = true, out) {
    (0, utils_js_1.assertBytes)(privateKey, 32);
    (0, utils_js_1.assertBool)(compressed);
    const res = secp256k1_js_1.secp256k1.getPublicKey(privateKey, compressed);
    return output(out, compressed ? 33 : 65, res);
}
function publicKeyVerify(publicKey) {
    (0, utils_js_1.assertBytes)(publicKey, 33, 65);
    try {
        Point.fromHex(publicKey);
        return true;
    }
    catch (e) {
        return false;
    }
}
function publicKeyConvert(publicKey, compressed = true, out) {
    (0, utils_js_1.assertBytes)(publicKey, 33, 65);
    (0, utils_js_1.assertBool)(compressed);
    const res = Point.fromHex(publicKey).toRawBytes(compressed);
    return output(out, compressed ? 33 : 65, res);
}
function ecdsaSign(msgHash, privateKey, options = { noncefn: undefined, data: undefined }, out) {
    (0, utils_js_1.assertBytes)(msgHash, 32);
    (0, utils_js_1.assertBytes)(privateKey, 32);
    if (typeof options !== "object" || options === null) {
        throw new TypeError("secp256k1.ecdsaSign: options should be object");
    }
    // noble-secp256k1 uses hmac instead of hmac-drbg here
    if (options &&
        (options.noncefn !== undefined || options.data !== undefined)) {
        throw new Error("Secp256k1: noncefn && data is unsupported");
    }
    const sig = secp256k1_js_1.secp256k1.sign(msgHash, privateKey);
    const recid = sig.recovery;
    return { signature: output(out, 64, sig.toCompactRawBytes()), recid };
}
function ecdsaRecover(signature, recid, msgHash, compressed = true, out) {
    (0, utils_js_1.assertBytes)(msgHash, 32);
    (0, utils_js_1.assertBool)(compressed);
    const sign = getSignature(signature);
    const point = sign.addRecoveryBit(recid).recoverPublicKey(msgHash);
    return output(out, compressed ? 33 : 65, point.toRawBytes(compressed));
}
function ecdsaVerify(signature, msgHash, publicKey) {
    (0, utils_js_1.assertBytes)(signature, 64);
    (0, utils_js_1.assertBytes)(msgHash, 32);
    (0, utils_js_1.assertBytes)(publicKey, 33, 65);
    (0, utils_js_1.assertBytes)(signature, 64);
    const r = bytesToNumber(signature.slice(0, 32));
    const s = bytesToNumber(signature.slice(32, 64));
    if (r >= ORDER || s >= ORDER) {
        throw new Error("Cannot parse signature");
    }
    const pub = Point.fromHex(publicKey); // can throw error
    pub; // typescript
    let sig;
    try {
        sig = getSignature(signature);
    }
    catch (error) {
        return false;
    }
    return secp256k1_js_1.secp256k1.verify(sig, msgHash, publicKey);
}
function privateKeyTweakAdd(privateKey, tweak) {
    (0, utils_js_1.assertBytes)(privateKey, 32);
    (0, utils_js_1.assertBytes)(tweak, 32);
    let t = bytesToNumber(tweak);
    if (t === 0n) {
        throw new Error("Tweak must not be zero");
    }
    if (t >= ORDER) {
        throw new Error("Tweak bigger than curve order");
    }
    t += bytesToNumber(privateKey);
    if (t >= ORDER) {
        t -= ORDER;
    }
    if (t === 0n) {
        throw new Error("The tweak was out of range or the resulted private key is invalid");
    }
    privateKey.set((0, utils_js_1.hexToBytes)(numberToHex(t)));
    return privateKey;
}
function privateKeyNegate(privateKey) {
    (0, utils_js_1.assertBytes)(privateKey, 32);
    const bn = (0, modular_1.mod)(-bytesToNumber(privateKey), ORDER);
    privateKey.set((0, utils_js_1.hexToBytes)(numberToHex(bn)));
    return privateKey;
}
function publicKeyNegate(publicKey, compressed = true, out) {
    (0, utils_js_1.assertBytes)(publicKey, 33, 65);
    (0, utils_js_1.assertBool)(compressed);
    const point = Point.fromHex(publicKey).negate();
    return output(out, compressed ? 33 : 65, point.toRawBytes(compressed));
}
function publicKeyCombine(publicKeys, compressed = true, out) {
    if (!Array.isArray(publicKeys) || !publicKeys.length) {
        throw new TypeError(`Expected array with one or more items, not ${publicKeys}`);
    }
    for (const publicKey of publicKeys) {
        (0, utils_js_1.assertBytes)(publicKey, 33, 65);
    }
    (0, utils_js_1.assertBool)(compressed);
    const combined = publicKeys
        .map((pub) => Point.fromHex(pub))
        .reduce((res, curr) => res.add(curr), Point.ZERO);
    // Prohibit returning ZERO point
    if (combined.equals(Point.ZERO)) {
        throw new Error("Combined result must not be zero");
    }
    return output(out, compressed ? 33 : 65, combined.toRawBytes(compressed));
}
function publicKeyTweakAdd(publicKey, tweak, compressed = true, out) {
    (0, utils_js_1.assertBytes)(publicKey, 33, 65);
    (0, utils_js_1.assertBytes)(tweak, 32);
    (0, utils_js_1.assertBool)(compressed);
    const p1 = Point.fromHex(publicKey);
    const p2 = Point.fromPrivateKey(tweak);
    const point = p1.add(p2);
    if (p2.equals(Point.ZERO) || point.equals(Point.ZERO)) {
        throw new Error("Tweak must not be zero");
    }
    return output(out, compressed ? 33 : 65, point.toRawBytes(compressed));
}
function publicKeyTweakMul(publicKey, tweak, compressed = true, out) {
    (0, utils_js_1.assertBytes)(publicKey, 33, 65);
    (0, utils_js_1.assertBytes)(tweak, 32);
    (0, utils_js_1.assertBool)(compressed);
    const bn = bytesToNumber(tweak);
    if (bn === 0n) {
        throw new Error("Tweak must not be zero");
    }
    if (bn <= 1 || bn >= ORDER) {
        throw new Error("Tweak is zero or bigger than curve order");
    }
    const point = Point.fromHex(publicKey).multiply(bn);
    return output(out, compressed ? 33 : 65, point.toRawBytes(compressed));
}
function privateKeyTweakMul(privateKey, tweak) {
    (0, utils_js_1.assertBytes)(privateKey, 32);
    (0, utils_js_1.assertBytes)(tweak, 32);
    const bn = bytesToNumber(tweak);
    if (bn <= 1 || bn >= ORDER) {
        throw new Error("Tweak is zero or bigger than curve order");
    }
    const res = (0, modular_1.mod)(bn * bytesToNumber(privateKey), ORDER);
    if (res === 0n) {
        throw new Error("The tweak was out of range or the resulted private key is invalid");
    }
    privateKey.set((0, utils_js_1.hexToBytes)(numberToHex(res)));
    return privateKey;
}
// internal -> DER
function signatureExport(signature, out) {
    const res = getSignature(signature).toDERRawBytes();
    return output(out, 72, res.slice()).slice(0, res.length);
}
// DER -> internal
function signatureImport(signature, out) {
    (0, utils_js_1.assertBytes)(signature);
    const sig = secp256k1_js_1.secp256k1.Signature.fromDER(signature);
    return output(out, 64, (0, utils_js_1.hexToBytes)(sig.toCompactHex()));
}
function signatureNormalize(signature) {
    const res = getSignature(signature);
    if (res.s > ORDER / 2n) {
        signature.set(numberToBytes(ORDER - res.s), 32);
    }
    return signature;
}
function ecdh(publicKey, privateKey, options = {}, out) {
    (0, utils_js_1.assertBytes)(publicKey, 33, 65);
    (0, utils_js_1.assertBytes)(privateKey, 32);
    if (typeof options !== "object" || options === null) {
        throw new TypeError("secp256k1.ecdh: options should be object");
    }
    if (options.data !== undefined) {
        (0, utils_js_1.assertBytes)(options.data);
    }
    const point = Point.fromHex(secp256k1_js_1.secp256k1.getSharedSecret(privateKey, publicKey));
    if (options.hashfn === undefined) {
        return output(out, 32, (0, sha256_1.sha256)(point.toRawBytes(true)));
    }
    if (typeof options.hashfn !== "function") {
        throw new TypeError("secp256k1.ecdh: options.hashfn should be function");
    }
    if (options.xbuf !== undefined) {
        (0, utils_js_1.assertBytes)(options.xbuf, 32);
    }
    if (options.ybuf !== undefined) {
        (0, utils_js_1.assertBytes)(options.ybuf, 32);
    }
    (0, utils_js_1.assertBytes)(out, 32);
    const { x, y } = point.toAffine();
    const xbuf = options.xbuf || new Uint8Array(32);
    xbuf.set(numberToBytes(x));
    const ybuf = options.ybuf || new Uint8Array(32);
    ybuf.set(numberToBytes(y));
    const hash = options.hashfn(xbuf, ybuf, options.data);
    if (!(hash instanceof Uint8Array) || hash.length !== 32) {
        throw new Error("secp256k1.ecdh: invalid options.hashfn output");
    }
    return output(out, 32, hash);
}
function contextRandomize(seed) {
    if (seed !== null) {
        (0, utils_js_1.assertBytes)(seed, 32);
    }
    // There is no context to randomize
}
