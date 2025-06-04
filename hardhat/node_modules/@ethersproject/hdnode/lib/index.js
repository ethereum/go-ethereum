"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getAccountPath = exports.isValidMnemonic = exports.entropyToMnemonic = exports.mnemonicToEntropy = exports.mnemonicToSeed = exports.HDNode = exports.defaultPath = void 0;
var basex_1 = require("@ethersproject/basex");
var bytes_1 = require("@ethersproject/bytes");
var bignumber_1 = require("@ethersproject/bignumber");
var strings_1 = require("@ethersproject/strings");
var pbkdf2_1 = require("@ethersproject/pbkdf2");
var properties_1 = require("@ethersproject/properties");
var signing_key_1 = require("@ethersproject/signing-key");
var sha2_1 = require("@ethersproject/sha2");
var transactions_1 = require("@ethersproject/transactions");
var wordlists_1 = require("@ethersproject/wordlists");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var N = bignumber_1.BigNumber.from("0xfffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141");
// "Bitcoin seed"
var MasterSecret = (0, strings_1.toUtf8Bytes)("Bitcoin seed");
var HardenedBit = 0x80000000;
// Returns a byte with the MSB bits set
function getUpperMask(bits) {
    return ((1 << bits) - 1) << (8 - bits);
}
// Returns a byte with the LSB bits set
function getLowerMask(bits) {
    return (1 << bits) - 1;
}
function bytes32(value) {
    return (0, bytes_1.hexZeroPad)((0, bytes_1.hexlify)(value), 32);
}
function base58check(data) {
    return basex_1.Base58.encode((0, bytes_1.concat)([data, (0, bytes_1.hexDataSlice)((0, sha2_1.sha256)((0, sha2_1.sha256)(data)), 0, 4)]));
}
function getWordlist(wordlist) {
    if (wordlist == null) {
        return wordlists_1.wordlists["en"];
    }
    if (typeof (wordlist) === "string") {
        var words = wordlists_1.wordlists[wordlist];
        if (words == null) {
            logger.throwArgumentError("unknown locale", "wordlist", wordlist);
        }
        return words;
    }
    return wordlist;
}
var _constructorGuard = {};
exports.defaultPath = "m/44'/60'/0'/0/0";
;
var HDNode = /** @class */ (function () {
    /**
     *  This constructor should not be called directly.
     *
     *  Please use:
     *   - fromMnemonic
     *   - fromSeed
     */
    function HDNode(constructorGuard, privateKey, publicKey, parentFingerprint, chainCode, index, depth, mnemonicOrPath) {
        /* istanbul ignore if */
        if (constructorGuard !== _constructorGuard) {
            throw new Error("HDNode constructor cannot be called directly");
        }
        if (privateKey) {
            var signingKey = new signing_key_1.SigningKey(privateKey);
            (0, properties_1.defineReadOnly)(this, "privateKey", signingKey.privateKey);
            (0, properties_1.defineReadOnly)(this, "publicKey", signingKey.compressedPublicKey);
        }
        else {
            (0, properties_1.defineReadOnly)(this, "privateKey", null);
            (0, properties_1.defineReadOnly)(this, "publicKey", (0, bytes_1.hexlify)(publicKey));
        }
        (0, properties_1.defineReadOnly)(this, "parentFingerprint", parentFingerprint);
        (0, properties_1.defineReadOnly)(this, "fingerprint", (0, bytes_1.hexDataSlice)((0, sha2_1.ripemd160)((0, sha2_1.sha256)(this.publicKey)), 0, 4));
        (0, properties_1.defineReadOnly)(this, "address", (0, transactions_1.computeAddress)(this.publicKey));
        (0, properties_1.defineReadOnly)(this, "chainCode", chainCode);
        (0, properties_1.defineReadOnly)(this, "index", index);
        (0, properties_1.defineReadOnly)(this, "depth", depth);
        if (mnemonicOrPath == null) {
            // From a source that does not preserve the path (e.g. extended keys)
            (0, properties_1.defineReadOnly)(this, "mnemonic", null);
            (0, properties_1.defineReadOnly)(this, "path", null);
        }
        else if (typeof (mnemonicOrPath) === "string") {
            // From a source that does not preserve the mnemonic (e.g. neutered)
            (0, properties_1.defineReadOnly)(this, "mnemonic", null);
            (0, properties_1.defineReadOnly)(this, "path", mnemonicOrPath);
        }
        else {
            // From a fully qualified source
            (0, properties_1.defineReadOnly)(this, "mnemonic", mnemonicOrPath);
            (0, properties_1.defineReadOnly)(this, "path", mnemonicOrPath.path);
        }
    }
    Object.defineProperty(HDNode.prototype, "extendedKey", {
        get: function () {
            // We only support the mainnet values for now, but if anyone needs
            // testnet values, let me know. I believe current sentiment is that
            // we should always use mainnet, and use BIP-44 to derive the network
            //   - Mainnet: public=0x0488B21E, private=0x0488ADE4
            //   - Testnet: public=0x043587CF, private=0x04358394
            if (this.depth >= 256) {
                throw new Error("Depth too large!");
            }
            return base58check((0, bytes_1.concat)([
                ((this.privateKey != null) ? "0x0488ADE4" : "0x0488B21E"),
                (0, bytes_1.hexlify)(this.depth),
                this.parentFingerprint,
                (0, bytes_1.hexZeroPad)((0, bytes_1.hexlify)(this.index), 4),
                this.chainCode,
                ((this.privateKey != null) ? (0, bytes_1.concat)(["0x00", this.privateKey]) : this.publicKey),
            ]));
        },
        enumerable: false,
        configurable: true
    });
    HDNode.prototype.neuter = function () {
        return new HDNode(_constructorGuard, null, this.publicKey, this.parentFingerprint, this.chainCode, this.index, this.depth, this.path);
    };
    HDNode.prototype._derive = function (index) {
        if (index > 0xffffffff) {
            throw new Error("invalid index - " + String(index));
        }
        // Base path
        var path = this.path;
        if (path) {
            path += "/" + (index & ~HardenedBit);
        }
        var data = new Uint8Array(37);
        if (index & HardenedBit) {
            if (!this.privateKey) {
                throw new Error("cannot derive child of neutered node");
            }
            // Data = 0x00 || ser_256(k_par)
            data.set((0, bytes_1.arrayify)(this.privateKey), 1);
            // Hardened path
            if (path) {
                path += "'";
            }
        }
        else {
            // Data = ser_p(point(k_par))
            data.set((0, bytes_1.arrayify)(this.publicKey));
        }
        // Data += ser_32(i)
        for (var i = 24; i >= 0; i -= 8) {
            data[33 + (i >> 3)] = ((index >> (24 - i)) & 0xff);
        }
        var I = (0, bytes_1.arrayify)((0, sha2_1.computeHmac)(sha2_1.SupportedAlgorithm.sha512, this.chainCode, data));
        var IL = I.slice(0, 32);
        var IR = I.slice(32);
        // The private key
        var ki = null;
        // The public key
        var Ki = null;
        if (this.privateKey) {
            ki = bytes32(bignumber_1.BigNumber.from(IL).add(this.privateKey).mod(N));
        }
        else {
            var ek = new signing_key_1.SigningKey((0, bytes_1.hexlify)(IL));
            Ki = ek._addPoint(this.publicKey);
        }
        var mnemonicOrPath = path;
        var srcMnemonic = this.mnemonic;
        if (srcMnemonic) {
            mnemonicOrPath = Object.freeze({
                phrase: srcMnemonic.phrase,
                path: path,
                locale: (srcMnemonic.locale || "en")
            });
        }
        return new HDNode(_constructorGuard, ki, Ki, this.fingerprint, bytes32(IR), index, this.depth + 1, mnemonicOrPath);
    };
    HDNode.prototype.derivePath = function (path) {
        var components = path.split("/");
        if (components.length === 0 || (components[0] === "m" && this.depth !== 0)) {
            throw new Error("invalid path - " + path);
        }
        if (components[0] === "m") {
            components.shift();
        }
        var result = this;
        for (var i = 0; i < components.length; i++) {
            var component = components[i];
            if (component.match(/^[0-9]+'$/)) {
                var index = parseInt(component.substring(0, component.length - 1));
                if (index >= HardenedBit) {
                    throw new Error("invalid path index - " + component);
                }
                result = result._derive(HardenedBit + index);
            }
            else if (component.match(/^[0-9]+$/)) {
                var index = parseInt(component);
                if (index >= HardenedBit) {
                    throw new Error("invalid path index - " + component);
                }
                result = result._derive(index);
            }
            else {
                throw new Error("invalid path component - " + component);
            }
        }
        return result;
    };
    HDNode._fromSeed = function (seed, mnemonic) {
        var seedArray = (0, bytes_1.arrayify)(seed);
        if (seedArray.length < 16 || seedArray.length > 64) {
            throw new Error("invalid seed");
        }
        var I = (0, bytes_1.arrayify)((0, sha2_1.computeHmac)(sha2_1.SupportedAlgorithm.sha512, MasterSecret, seedArray));
        return new HDNode(_constructorGuard, bytes32(I.slice(0, 32)), null, "0x00000000", bytes32(I.slice(32)), 0, 0, mnemonic);
    };
    HDNode.fromMnemonic = function (mnemonic, password, wordlist) {
        // If a locale name was passed in, find the associated wordlist
        wordlist = getWordlist(wordlist);
        // Normalize the case and spacing in the mnemonic (throws if the mnemonic is invalid)
        mnemonic = entropyToMnemonic(mnemonicToEntropy(mnemonic, wordlist), wordlist);
        return HDNode._fromSeed(mnemonicToSeed(mnemonic, password), {
            phrase: mnemonic,
            path: "m",
            locale: wordlist.locale
        });
    };
    HDNode.fromSeed = function (seed) {
        return HDNode._fromSeed(seed, null);
    };
    HDNode.fromExtendedKey = function (extendedKey) {
        var bytes = basex_1.Base58.decode(extendedKey);
        if (bytes.length !== 82 || base58check(bytes.slice(0, 78)) !== extendedKey) {
            logger.throwArgumentError("invalid extended key", "extendedKey", "[REDACTED]");
        }
        var depth = bytes[4];
        var parentFingerprint = (0, bytes_1.hexlify)(bytes.slice(5, 9));
        var index = parseInt((0, bytes_1.hexlify)(bytes.slice(9, 13)).substring(2), 16);
        var chainCode = (0, bytes_1.hexlify)(bytes.slice(13, 45));
        var key = bytes.slice(45, 78);
        switch ((0, bytes_1.hexlify)(bytes.slice(0, 4))) {
            // Public Key
            case "0x0488b21e":
            case "0x043587cf":
                return new HDNode(_constructorGuard, null, (0, bytes_1.hexlify)(key), parentFingerprint, chainCode, index, depth, null);
            // Private Key
            case "0x0488ade4":
            case "0x04358394 ":
                if (key[0] !== 0) {
                    break;
                }
                return new HDNode(_constructorGuard, (0, bytes_1.hexlify)(key.slice(1)), null, parentFingerprint, chainCode, index, depth, null);
        }
        return logger.throwArgumentError("invalid extended key", "extendedKey", "[REDACTED]");
    };
    return HDNode;
}());
exports.HDNode = HDNode;
function mnemonicToSeed(mnemonic, password) {
    if (!password) {
        password = "";
    }
    var salt = (0, strings_1.toUtf8Bytes)("mnemonic" + password, strings_1.UnicodeNormalizationForm.NFKD);
    return (0, pbkdf2_1.pbkdf2)((0, strings_1.toUtf8Bytes)(mnemonic, strings_1.UnicodeNormalizationForm.NFKD), salt, 2048, 64, "sha512");
}
exports.mnemonicToSeed = mnemonicToSeed;
function mnemonicToEntropy(mnemonic, wordlist) {
    wordlist = getWordlist(wordlist);
    logger.checkNormalize();
    var words = wordlist.split(mnemonic);
    if ((words.length % 3) !== 0) {
        throw new Error("invalid mnemonic");
    }
    var entropy = (0, bytes_1.arrayify)(new Uint8Array(Math.ceil(11 * words.length / 8)));
    var offset = 0;
    for (var i = 0; i < words.length; i++) {
        var index = wordlist.getWordIndex(words[i].normalize("NFKD"));
        if (index === -1) {
            throw new Error("invalid mnemonic");
        }
        for (var bit = 0; bit < 11; bit++) {
            if (index & (1 << (10 - bit))) {
                entropy[offset >> 3] |= (1 << (7 - (offset % 8)));
            }
            offset++;
        }
    }
    var entropyBits = 32 * words.length / 3;
    var checksumBits = words.length / 3;
    var checksumMask = getUpperMask(checksumBits);
    var checksum = (0, bytes_1.arrayify)((0, sha2_1.sha256)(entropy.slice(0, entropyBits / 8)))[0] & checksumMask;
    if (checksum !== (entropy[entropy.length - 1] & checksumMask)) {
        throw new Error("invalid checksum");
    }
    return (0, bytes_1.hexlify)(entropy.slice(0, entropyBits / 8));
}
exports.mnemonicToEntropy = mnemonicToEntropy;
function entropyToMnemonic(entropy, wordlist) {
    wordlist = getWordlist(wordlist);
    entropy = (0, bytes_1.arrayify)(entropy);
    if ((entropy.length % 4) !== 0 || entropy.length < 16 || entropy.length > 32) {
        throw new Error("invalid entropy");
    }
    var indices = [0];
    var remainingBits = 11;
    for (var i = 0; i < entropy.length; i++) {
        // Consume the whole byte (with still more to go)
        if (remainingBits > 8) {
            indices[indices.length - 1] <<= 8;
            indices[indices.length - 1] |= entropy[i];
            remainingBits -= 8;
            // This byte will complete an 11-bit index
        }
        else {
            indices[indices.length - 1] <<= remainingBits;
            indices[indices.length - 1] |= entropy[i] >> (8 - remainingBits);
            // Start the next word
            indices.push(entropy[i] & getLowerMask(8 - remainingBits));
            remainingBits += 3;
        }
    }
    // Compute the checksum bits
    var checksumBits = entropy.length / 4;
    var checksum = (0, bytes_1.arrayify)((0, sha2_1.sha256)(entropy))[0] & getUpperMask(checksumBits);
    // Shift the checksum into the word indices
    indices[indices.length - 1] <<= checksumBits;
    indices[indices.length - 1] |= (checksum >> (8 - checksumBits));
    return wordlist.join(indices.map(function (index) { return wordlist.getWord(index); }));
}
exports.entropyToMnemonic = entropyToMnemonic;
function isValidMnemonic(mnemonic, wordlist) {
    try {
        mnemonicToEntropy(mnemonic, wordlist);
        return true;
    }
    catch (error) { }
    return false;
}
exports.isValidMnemonic = isValidMnemonic;
function getAccountPath(index) {
    if (typeof (index) !== "number" || index < 0 || index >= HardenedBit || index % 1) {
        logger.throwArgumentError("invalid account index", "index", index);
    }
    return "m/44'/60'/" + index + "'/0/0";
}
exports.getAccountPath = getAccountPath;
//# sourceMappingURL=index.js.map