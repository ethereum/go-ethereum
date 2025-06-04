"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.contextRandomize = exports.ecdh = exports.signatureNormalize = exports.signatureImport = exports.signatureExport = exports.privateKeyTweakMul = exports.publicKeyTweakMul = exports.publicKeyTweakAdd = exports.publicKeyCombine = exports.publicKeyNegate = exports.privateKeyNegate = exports.privateKeyTweakAdd = exports.ecdsaVerify = exports.ecdsaRecover = exports.ecdsaSign = exports.publicKeyConvert = exports.publicKeyVerify = exports.publicKeyCreate = exports.privateKeyVerify = exports.createPrivateKey = exports.createPrivateKeySync = void 0;
const sha256_1 = require("@noble/hashes/sha256");
const secp = __importStar(require("./secp256k1"));
const utils_1 = require("./utils");
// Use `secp256k1` module directly.
// This is a legacy compatibility layer for the npm package `secp256k1` via noble-secp256k1
function hexToNumber(hex) {
    if (typeof hex !== "string") {
        throw new TypeError("hexToNumber: expected string, got " + typeof hex);
    }
    return BigInt(`0x${hex}`);
}
// Copy-paste from secp256k1, maybe export it?
const bytesToNumber = (bytes) => hexToNumber((0, utils_1.toHex)(bytes));
const numberToHex = (num) => num.toString(16).padStart(64, "0");
const numberToBytes = (num) => (0, utils_1.hexToBytes)(numberToHex(num));
const { mod } = secp.utils;
const ORDER = secp.CURVE.n;
function output(out = (len) => new Uint8Array(len), length, value) {
    if (typeof out === "function") {
        out = out(length);
    }
    (0, utils_1.assertBytes)(out, length);
    if (value) {
        out.set(value);
    }
    return out;
}
function getSignature(signature) {
    (0, utils_1.assertBytes)(signature, 64);
    return secp.Signature.fromCompact(signature);
}
function createPrivateKeySync() {
    return secp.utils.randomPrivateKey();
}
exports.createPrivateKeySync = createPrivateKeySync;
async function createPrivateKey() {
    return createPrivateKeySync();
}
exports.createPrivateKey = createPrivateKey;
function privateKeyVerify(privateKey) {
    (0, utils_1.assertBytes)(privateKey, 32);
    return secp.utils.isValidPrivateKey(privateKey);
}
exports.privateKeyVerify = privateKeyVerify;
function publicKeyCreate(privateKey, compressed = true, out) {
    (0, utils_1.assertBytes)(privateKey, 32);
    (0, utils_1.assertBool)(compressed);
    const res = secp.getPublicKey(privateKey, compressed);
    return output(out, compressed ? 33 : 65, res);
}
exports.publicKeyCreate = publicKeyCreate;
function publicKeyVerify(publicKey) {
    (0, utils_1.assertBytes)(publicKey, 33, 65);
    try {
        secp.Point.fromHex(publicKey);
        return true;
    }
    catch (e) {
        return false;
    }
}
exports.publicKeyVerify = publicKeyVerify;
function publicKeyConvert(publicKey, compressed = true, out) {
    (0, utils_1.assertBytes)(publicKey, 33, 65);
    (0, utils_1.assertBool)(compressed);
    const res = secp.Point.fromHex(publicKey).toRawBytes(compressed);
    return output(out, compressed ? 33 : 65, res);
}
exports.publicKeyConvert = publicKeyConvert;
function ecdsaSign(msgHash, privateKey, options = { noncefn: undefined, data: undefined }, out) {
    (0, utils_1.assertBytes)(msgHash, 32);
    (0, utils_1.assertBytes)(privateKey, 32);
    if (typeof options !== "object" || options === null) {
        throw new TypeError("secp256k1.ecdsaSign: options should be object");
    }
    // noble-secp256k1 uses hmac instead of hmac-drbg here
    if (options &&
        (options.noncefn !== undefined || options.data !== undefined)) {
        throw new Error("Secp256k1: noncefn && data is unsupported");
    }
    const [signature, recid] = secp.signSync(msgHash, privateKey, {
        recovered: true,
        der: false,
    });
    return { signature: output(out, 64, signature), recid };
}
exports.ecdsaSign = ecdsaSign;
function ecdsaRecover(signature, recid, msgHash, compressed = true, out) {
    (0, utils_1.assertBytes)(msgHash, 32);
    (0, utils_1.assertBool)(compressed);
    const sign = getSignature(signature).toHex();
    const point = secp.Point.fromSignature(msgHash, sign, recid);
    return output(out, compressed ? 33 : 65, point.toRawBytes(compressed));
}
exports.ecdsaRecover = ecdsaRecover;
function ecdsaVerify(signature, msgHash, publicKey) {
    (0, utils_1.assertBytes)(signature, 64);
    (0, utils_1.assertBytes)(msgHash, 32);
    (0, utils_1.assertBytes)(publicKey, 33, 65);
    (0, utils_1.assertBytes)(signature, 64);
    const r = bytesToNumber(signature.slice(0, 32));
    const s = bytesToNumber(signature.slice(32, 64));
    if (r >= ORDER || s >= ORDER) {
        throw new Error("Cannot parse signature");
    }
    const pub = secp.Point.fromHex(publicKey); // should not throw error
    let sig;
    try {
        sig = getSignature(signature);
    }
    catch (error) {
        return false;
    }
    return secp.verify(sig, msgHash, pub);
}
exports.ecdsaVerify = ecdsaVerify;
function privateKeyTweakAdd(privateKey, tweak) {
    (0, utils_1.assertBytes)(privateKey, 32);
    (0, utils_1.assertBytes)(tweak, 32);
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
    privateKey.set((0, utils_1.hexToBytes)(numberToHex(t)));
    return privateKey;
}
exports.privateKeyTweakAdd = privateKeyTweakAdd;
function privateKeyNegate(privateKey) {
    (0, utils_1.assertBytes)(privateKey, 32);
    const bn = mod(-bytesToNumber(privateKey), ORDER);
    privateKey.set((0, utils_1.hexToBytes)(numberToHex(bn)));
    return privateKey;
}
exports.privateKeyNegate = privateKeyNegate;
function publicKeyNegate(publicKey, compressed = true, out) {
    (0, utils_1.assertBytes)(publicKey, 33, 65);
    (0, utils_1.assertBool)(compressed);
    const point = secp.Point.fromHex(publicKey).negate();
    return output(out, compressed ? 33 : 65, point.toRawBytes(compressed));
}
exports.publicKeyNegate = publicKeyNegate;
function publicKeyCombine(publicKeys, compressed = true, out) {
    if (!Array.isArray(publicKeys) || !publicKeys.length) {
        throw new TypeError(`Expected array with one or more items, not ${publicKeys}`);
    }
    for (const publicKey of publicKeys) {
        (0, utils_1.assertBytes)(publicKey, 33, 65);
    }
    (0, utils_1.assertBool)(compressed);
    const combined = publicKeys
        .map((pub) => secp.Point.fromHex(pub))
        .reduce((res, curr) => res.add(curr), secp.Point.ZERO);
    // Prohibit returning ZERO point
    if (combined.equals(secp.Point.ZERO)) {
        throw new Error("Combined result must not be zero");
    }
    return output(out, compressed ? 33 : 65, combined.toRawBytes(compressed));
}
exports.publicKeyCombine = publicKeyCombine;
function publicKeyTweakAdd(publicKey, tweak, compressed = true, out) {
    (0, utils_1.assertBytes)(publicKey, 33, 65);
    (0, utils_1.assertBytes)(tweak, 32);
    (0, utils_1.assertBool)(compressed);
    const p1 = secp.Point.fromHex(publicKey);
    const p2 = secp.Point.fromPrivateKey(tweak);
    const point = p1.add(p2);
    if (p2.equals(secp.Point.ZERO) || point.equals(secp.Point.ZERO)) {
        throw new Error("Tweak must not be zero");
    }
    return output(out, compressed ? 33 : 65, point.toRawBytes(compressed));
}
exports.publicKeyTweakAdd = publicKeyTweakAdd;
function publicKeyTweakMul(publicKey, tweak, compressed = true, out) {
    (0, utils_1.assertBytes)(publicKey, 33, 65);
    (0, utils_1.assertBytes)(tweak, 32);
    (0, utils_1.assertBool)(compressed);
    const bn = bytesToNumber(tweak);
    if (bn === 0n) {
        throw new Error("Tweak must not be zero");
    }
    if (bn <= 1 || bn >= ORDER) {
        throw new Error("Tweak is zero or bigger than curve order");
    }
    const point = secp.Point.fromHex(publicKey).multiply(bn);
    return output(out, compressed ? 33 : 65, point.toRawBytes(compressed));
}
exports.publicKeyTweakMul = publicKeyTweakMul;
function privateKeyTweakMul(privateKey, tweak) {
    (0, utils_1.assertBytes)(privateKey, 32);
    (0, utils_1.assertBytes)(tweak, 32);
    const bn = bytesToNumber(tweak);
    if (bn <= 1 || bn >= ORDER) {
        throw new Error("Tweak is zero or bigger than curve order");
    }
    const res = mod(bn * bytesToNumber(privateKey), ORDER);
    if (res === 0n) {
        throw new Error("The tweak was out of range or the resulted private key is invalid");
    }
    privateKey.set((0, utils_1.hexToBytes)(numberToHex(res)));
    return privateKey;
}
exports.privateKeyTweakMul = privateKeyTweakMul;
// internal -> DER
function signatureExport(signature, out) {
    const res = getSignature(signature).toRawBytes();
    return output(out, 72, getSignature(signature).toRawBytes()).slice(0, res.length);
}
exports.signatureExport = signatureExport;
// DER -> internal
function signatureImport(signature, out) {
    (0, utils_1.assertBytes)(signature);
    const sig = secp.Signature.fromDER(signature);
    return output(out, 64, (0, utils_1.hexToBytes)(sig.toCompactHex()));
}
exports.signatureImport = signatureImport;
function signatureNormalize(signature) {
    const res = getSignature(signature);
    if (res.s > ORDER / 2n) {
        signature.set(numberToBytes(ORDER - res.s), 32);
    }
    return signature;
}
exports.signatureNormalize = signatureNormalize;
function ecdh(publicKey, privateKey, options = {}, out) {
    (0, utils_1.assertBytes)(publicKey, 33, 65);
    (0, utils_1.assertBytes)(privateKey, 32);
    if (typeof options !== "object" || options === null) {
        throw new TypeError("secp256k1.ecdh: options should be object");
    }
    if (options.data !== undefined) {
        (0, utils_1.assertBytes)(options.data);
    }
    const point = secp.Point.fromHex(secp.getSharedSecret(privateKey, publicKey));
    if (options.hashfn === undefined) {
        return output(out, 32, (0, sha256_1.sha256)(point.toRawBytes(true)));
    }
    if (typeof options.hashfn !== "function") {
        throw new TypeError("secp256k1.ecdh: options.hashfn should be function");
    }
    if (options.xbuf !== undefined) {
        (0, utils_1.assertBytes)(options.xbuf, 32);
    }
    if (options.ybuf !== undefined) {
        (0, utils_1.assertBytes)(options.ybuf, 32);
    }
    (0, utils_1.assertBytes)(out, 32);
    const xbuf = options.xbuf || new Uint8Array(32);
    xbuf.set(numberToBytes(point.x));
    const ybuf = options.ybuf || new Uint8Array(32);
    ybuf.set(numberToBytes(point.y));
    const hash = options.hashfn(xbuf, ybuf, options.data);
    if (!(hash instanceof Uint8Array) || hash.length !== 32) {
        throw new Error("secp256k1.ecdh: invalid options.hashfn output");
    }
    return output(out, 32, hash);
}
exports.ecdh = ecdh;
function contextRandomize(seed) {
    if (seed !== null) {
        (0, utils_1.assertBytes)(seed, 32);
    }
    // There is no context to randomize
}
exports.contextRandomize = contextRandomize;
