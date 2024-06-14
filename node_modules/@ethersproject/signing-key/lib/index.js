"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.computePublicKey = exports.recoverPublicKey = exports.SigningKey = void 0;
var elliptic_1 = require("./elliptic");
var bytes_1 = require("@ethersproject/bytes");
var properties_1 = require("@ethersproject/properties");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var _curve = null;
function getCurve() {
    if (!_curve) {
        _curve = new elliptic_1.EC("secp256k1");
    }
    return _curve;
}
var SigningKey = /** @class */ (function () {
    function SigningKey(privateKey) {
        (0, properties_1.defineReadOnly)(this, "curve", "secp256k1");
        (0, properties_1.defineReadOnly)(this, "privateKey", (0, bytes_1.hexlify)(privateKey));
        if ((0, bytes_1.hexDataLength)(this.privateKey) !== 32) {
            logger.throwArgumentError("invalid private key", "privateKey", "[[ REDACTED ]]");
        }
        var keyPair = getCurve().keyFromPrivate((0, bytes_1.arrayify)(this.privateKey));
        (0, properties_1.defineReadOnly)(this, "publicKey", "0x" + keyPair.getPublic(false, "hex"));
        (0, properties_1.defineReadOnly)(this, "compressedPublicKey", "0x" + keyPair.getPublic(true, "hex"));
        (0, properties_1.defineReadOnly)(this, "_isSigningKey", true);
    }
    SigningKey.prototype._addPoint = function (other) {
        var p0 = getCurve().keyFromPublic((0, bytes_1.arrayify)(this.publicKey));
        var p1 = getCurve().keyFromPublic((0, bytes_1.arrayify)(other));
        return "0x" + p0.pub.add(p1.pub).encodeCompressed("hex");
    };
    SigningKey.prototype.signDigest = function (digest) {
        var keyPair = getCurve().keyFromPrivate((0, bytes_1.arrayify)(this.privateKey));
        var digestBytes = (0, bytes_1.arrayify)(digest);
        if (digestBytes.length !== 32) {
            logger.throwArgumentError("bad digest length", "digest", digest);
        }
        var signature = keyPair.sign(digestBytes, { canonical: true });
        return (0, bytes_1.splitSignature)({
            recoveryParam: signature.recoveryParam,
            r: (0, bytes_1.hexZeroPad)("0x" + signature.r.toString(16), 32),
            s: (0, bytes_1.hexZeroPad)("0x" + signature.s.toString(16), 32),
        });
    };
    SigningKey.prototype.computeSharedSecret = function (otherKey) {
        var keyPair = getCurve().keyFromPrivate((0, bytes_1.arrayify)(this.privateKey));
        var otherKeyPair = getCurve().keyFromPublic((0, bytes_1.arrayify)(computePublicKey(otherKey)));
        return (0, bytes_1.hexZeroPad)("0x" + keyPair.derive(otherKeyPair.getPublic()).toString(16), 32);
    };
    SigningKey.isSigningKey = function (value) {
        return !!(value && value._isSigningKey);
    };
    return SigningKey;
}());
exports.SigningKey = SigningKey;
function recoverPublicKey(digest, signature) {
    var sig = (0, bytes_1.splitSignature)(signature);
    var rs = { r: (0, bytes_1.arrayify)(sig.r), s: (0, bytes_1.arrayify)(sig.s) };
    return "0x" + getCurve().recoverPubKey((0, bytes_1.arrayify)(digest), rs, sig.recoveryParam).encode("hex", false);
}
exports.recoverPublicKey = recoverPublicKey;
function computePublicKey(key, compressed) {
    var bytes = (0, bytes_1.arrayify)(key);
    if (bytes.length === 32) {
        var signingKey = new SigningKey(bytes);
        if (compressed) {
            return "0x" + getCurve().keyFromPrivate(bytes).getPublic(true, "hex");
        }
        return signingKey.publicKey;
    }
    else if (bytes.length === 33) {
        if (compressed) {
            return (0, bytes_1.hexlify)(bytes);
        }
        return "0x" + getCurve().keyFromPublic(bytes).getPublic(false, "hex");
    }
    else if (bytes.length === 65) {
        if (!compressed) {
            return (0, bytes_1.hexlify)(bytes);
        }
        return "0x" + getCurve().keyFromPublic(bytes).getPublic(true, "hex");
    }
    return logger.throwArgumentError("invalid public or private key", "key", "[REDACTED]");
}
exports.computePublicKey = computePublicKey;
//# sourceMappingURL=index.js.map