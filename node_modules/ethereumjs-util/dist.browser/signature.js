"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.hashPersonalMessage = exports.isValidSignature = exports.fromRpcSig = exports.toCompactSig = exports.toRpcSig = exports.ecrecover = exports.ecsign = void 0;
var secp256k1_1 = require("ethereum-cryptography/secp256k1");
var externals_1 = require("./externals");
var bytes_1 = require("./bytes");
var hash_1 = require("./hash");
var helpers_1 = require("./helpers");
var types_1 = require("./types");
function ecsign(msgHash, privateKey, chainId) {
    var _a = (0, secp256k1_1.ecdsaSign)(msgHash, privateKey), signature = _a.signature, recovery = _a.recid;
    var r = Buffer.from(signature.slice(0, 32));
    var s = Buffer.from(signature.slice(32, 64));
    if (!chainId || typeof chainId === 'number') {
        // return legacy type ECDSASignature (deprecated in favor of ECDSASignatureBuffer to handle large chainIds)
        if (chainId && !Number.isSafeInteger(chainId)) {
            throw new Error('The provided number is greater than MAX_SAFE_INTEGER (please use an alternative input type)');
        }
        var v_1 = chainId ? recovery + (chainId * 2 + 35) : recovery + 27;
        return { r: r, s: s, v: v_1 };
    }
    var chainIdBN = (0, types_1.toType)(chainId, types_1.TypeOutput.BN);
    var v = chainIdBN.muln(2).addn(35).addn(recovery).toArrayLike(Buffer);
    return { r: r, s: s, v: v };
}
exports.ecsign = ecsign;
function calculateSigRecovery(v, chainId) {
    var vBN = (0, types_1.toType)(v, types_1.TypeOutput.BN);
    if (vBN.eqn(0) || vBN.eqn(1))
        return (0, types_1.toType)(v, types_1.TypeOutput.BN);
    if (!chainId) {
        return vBN.subn(27);
    }
    var chainIdBN = (0, types_1.toType)(chainId, types_1.TypeOutput.BN);
    return vBN.sub(chainIdBN.muln(2).addn(35));
}
function isValidSigRecovery(recovery) {
    var rec = new externals_1.BN(recovery);
    return rec.eqn(0) || rec.eqn(1);
}
/**
 * ECDSA public key recovery from signature.
 * NOTE: Accepts `v == 0 | v == 1` for EIP1559 transactions
 * @returns Recovered public key
 */
var ecrecover = function (msgHash, v, r, s, chainId) {
    var signature = Buffer.concat([(0, bytes_1.setLengthLeft)(r, 32), (0, bytes_1.setLengthLeft)(s, 32)], 64);
    var recovery = calculateSigRecovery(v, chainId);
    if (!isValidSigRecovery(recovery)) {
        throw new Error('Invalid signature v value');
    }
    var senderPubKey = (0, secp256k1_1.ecdsaRecover)(signature, recovery.toNumber(), msgHash);
    return Buffer.from((0, secp256k1_1.publicKeyConvert)(senderPubKey, false).slice(1));
};
exports.ecrecover = ecrecover;
/**
 * Convert signature parameters into the format of `eth_sign` RPC method.
 * NOTE: Accepts `v == 0 | v == 1` for EIP1559 transactions
 * @returns Signature
 */
var toRpcSig = function (v, r, s, chainId) {
    var recovery = calculateSigRecovery(v, chainId);
    if (!isValidSigRecovery(recovery)) {
        throw new Error('Invalid signature v value');
    }
    // geth (and the RPC eth_sign method) uses the 65 byte format used by Bitcoin
    return (0, bytes_1.bufferToHex)(Buffer.concat([(0, bytes_1.setLengthLeft)(r, 32), (0, bytes_1.setLengthLeft)(s, 32), (0, bytes_1.toBuffer)(v)]));
};
exports.toRpcSig = toRpcSig;
/**
 * Convert signature parameters into the format of Compact Signature Representation (EIP-2098).
 * NOTE: Accepts `v == 0 | v == 1` for EIP1559 transactions
 * @returns Signature
 */
var toCompactSig = function (v, r, s, chainId) {
    var recovery = calculateSigRecovery(v, chainId);
    if (!isValidSigRecovery(recovery)) {
        throw new Error('Invalid signature v value');
    }
    var vn = (0, types_1.toType)(v, types_1.TypeOutput.Number);
    var ss = s;
    if ((vn > 28 && vn % 2 === 1) || vn === 1 || vn === 28) {
        ss = Buffer.from(s);
        ss[0] |= 0x80;
    }
    return (0, bytes_1.bufferToHex)(Buffer.concat([(0, bytes_1.setLengthLeft)(r, 32), (0, bytes_1.setLengthLeft)(ss, 32)]));
};
exports.toCompactSig = toCompactSig;
/**
 * Convert signature format of the `eth_sign` RPC method to signature parameters
 * NOTE: all because of a bug in geth: https://github.com/ethereum/go-ethereum/issues/2053
 * NOTE: After EIP1559, `v` could be `0` or `1` but this function assumes
 * it's a signed message (EIP-191 or EIP-712) adding `27` at the end. Remove if needed.
 */
var fromRpcSig = function (sig) {
    var buf = (0, bytes_1.toBuffer)(sig);
    var r;
    var s;
    var v;
    if (buf.length >= 65) {
        r = buf.slice(0, 32);
        s = buf.slice(32, 64);
        v = (0, bytes_1.bufferToInt)(buf.slice(64));
    }
    else if (buf.length === 64) {
        // Compact Signature Representation (https://eips.ethereum.org/EIPS/eip-2098)
        r = buf.slice(0, 32);
        s = buf.slice(32, 64);
        v = (0, bytes_1.bufferToInt)(buf.slice(32, 33)) >> 7;
        s[0] &= 0x7f;
    }
    else {
        throw new Error('Invalid signature length');
    }
    // support both versions of `eth_sign` responses
    if (v < 27) {
        v += 27;
    }
    return {
        v: v,
        r: r,
        s: s,
    };
};
exports.fromRpcSig = fromRpcSig;
/**
 * Validate a ECDSA signature.
 * NOTE: Accepts `v == 0 | v == 1` for EIP1559 transactions
 * @param homesteadOrLater Indicates whether this is being used on either the homestead hardfork or a later one
 */
var isValidSignature = function (v, r, s, homesteadOrLater, chainId) {
    if (homesteadOrLater === void 0) { homesteadOrLater = true; }
    var SECP256K1_N_DIV_2 = new externals_1.BN('7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0', 16);
    var SECP256K1_N = new externals_1.BN('fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141', 16);
    if (r.length !== 32 || s.length !== 32) {
        return false;
    }
    if (!isValidSigRecovery(calculateSigRecovery(v, chainId))) {
        return false;
    }
    var rBN = new externals_1.BN(r);
    var sBN = new externals_1.BN(s);
    if (rBN.isZero() || rBN.gt(SECP256K1_N) || sBN.isZero() || sBN.gt(SECP256K1_N)) {
        return false;
    }
    if (homesteadOrLater && sBN.cmp(SECP256K1_N_DIV_2) === 1) {
        return false;
    }
    return true;
};
exports.isValidSignature = isValidSignature;
/**
 * Returns the keccak-256 hash of `message`, prefixed with the header used by the `eth_sign` RPC call.
 * The output of this function can be fed into `ecsign` to produce the same signature as the `eth_sign`
 * call for a given `message`, or fed to `ecrecover` along with a signature to recover the public key
 * used to produce the signature.
 */
var hashPersonalMessage = function (message) {
    (0, helpers_1.assertIsBuffer)(message);
    var prefix = Buffer.from("\u0019Ethereum Signed Message:\n".concat(message.length), 'utf-8');
    return (0, hash_1.keccak)(Buffer.concat([prefix, message]));
};
exports.hashPersonalMessage = hashPersonalMessage;
//# sourceMappingURL=signature.js.map