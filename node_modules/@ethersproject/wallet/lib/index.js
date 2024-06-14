"use strict";
var __extends = (this && this.__extends) || (function () {
    var extendStatics = function (d, b) {
        extendStatics = Object.setPrototypeOf ||
            ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
            function (d, b) { for (var p in b) if (Object.prototype.hasOwnProperty.call(b, p)) d[p] = b[p]; };
        return extendStatics(d, b);
    };
    return function (d, b) {
        if (typeof b !== "function" && b !== null)
            throw new TypeError("Class extends value " + String(b) + " is not a constructor or null");
        extendStatics(d, b);
        function __() { this.constructor = d; }
        d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
    };
})();
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (_) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.verifyTypedData = exports.verifyMessage = exports.Wallet = void 0;
var address_1 = require("@ethersproject/address");
var abstract_provider_1 = require("@ethersproject/abstract-provider");
var abstract_signer_1 = require("@ethersproject/abstract-signer");
var bytes_1 = require("@ethersproject/bytes");
var hash_1 = require("@ethersproject/hash");
var hdnode_1 = require("@ethersproject/hdnode");
var keccak256_1 = require("@ethersproject/keccak256");
var properties_1 = require("@ethersproject/properties");
var random_1 = require("@ethersproject/random");
var signing_key_1 = require("@ethersproject/signing-key");
var json_wallets_1 = require("@ethersproject/json-wallets");
var transactions_1 = require("@ethersproject/transactions");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
function isAccount(value) {
    return (value != null && (0, bytes_1.isHexString)(value.privateKey, 32) && value.address != null);
}
function hasMnemonic(value) {
    var mnemonic = value.mnemonic;
    return (mnemonic && mnemonic.phrase);
}
var Wallet = /** @class */ (function (_super) {
    __extends(Wallet, _super);
    function Wallet(privateKey, provider) {
        var _this = _super.call(this) || this;
        if (isAccount(privateKey)) {
            var signingKey_1 = new signing_key_1.SigningKey(privateKey.privateKey);
            (0, properties_1.defineReadOnly)(_this, "_signingKey", function () { return signingKey_1; });
            (0, properties_1.defineReadOnly)(_this, "address", (0, transactions_1.computeAddress)(_this.publicKey));
            if (_this.address !== (0, address_1.getAddress)(privateKey.address)) {
                logger.throwArgumentError("privateKey/address mismatch", "privateKey", "[REDACTED]");
            }
            if (hasMnemonic(privateKey)) {
                var srcMnemonic_1 = privateKey.mnemonic;
                (0, properties_1.defineReadOnly)(_this, "_mnemonic", function () { return ({
                    phrase: srcMnemonic_1.phrase,
                    path: srcMnemonic_1.path || hdnode_1.defaultPath,
                    locale: srcMnemonic_1.locale || "en"
                }); });
                var mnemonic = _this.mnemonic;
                var node = hdnode_1.HDNode.fromMnemonic(mnemonic.phrase, null, mnemonic.locale).derivePath(mnemonic.path);
                if ((0, transactions_1.computeAddress)(node.privateKey) !== _this.address) {
                    logger.throwArgumentError("mnemonic/address mismatch", "privateKey", "[REDACTED]");
                }
            }
            else {
                (0, properties_1.defineReadOnly)(_this, "_mnemonic", function () { return null; });
            }
        }
        else {
            if (signing_key_1.SigningKey.isSigningKey(privateKey)) {
                /* istanbul ignore if */
                if (privateKey.curve !== "secp256k1") {
                    logger.throwArgumentError("unsupported curve; must be secp256k1", "privateKey", "[REDACTED]");
                }
                (0, properties_1.defineReadOnly)(_this, "_signingKey", function () { return privateKey; });
            }
            else {
                // A lot of common tools do not prefix private keys with a 0x (see: #1166)
                if (typeof (privateKey) === "string") {
                    if (privateKey.match(/^[0-9a-f]*$/i) && privateKey.length === 64) {
                        privateKey = "0x" + privateKey;
                    }
                }
                var signingKey_2 = new signing_key_1.SigningKey(privateKey);
                (0, properties_1.defineReadOnly)(_this, "_signingKey", function () { return signingKey_2; });
            }
            (0, properties_1.defineReadOnly)(_this, "_mnemonic", function () { return null; });
            (0, properties_1.defineReadOnly)(_this, "address", (0, transactions_1.computeAddress)(_this.publicKey));
        }
        /* istanbul ignore if */
        if (provider && !abstract_provider_1.Provider.isProvider(provider)) {
            logger.throwArgumentError("invalid provider", "provider", provider);
        }
        (0, properties_1.defineReadOnly)(_this, "provider", provider || null);
        return _this;
    }
    Object.defineProperty(Wallet.prototype, "mnemonic", {
        get: function () { return this._mnemonic(); },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Wallet.prototype, "privateKey", {
        get: function () { return this._signingKey().privateKey; },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Wallet.prototype, "publicKey", {
        get: function () { return this._signingKey().publicKey; },
        enumerable: false,
        configurable: true
    });
    Wallet.prototype.getAddress = function () {
        return Promise.resolve(this.address);
    };
    Wallet.prototype.connect = function (provider) {
        return new Wallet(this, provider);
    };
    Wallet.prototype.signTransaction = function (transaction) {
        var _this = this;
        return (0, properties_1.resolveProperties)(transaction).then(function (tx) {
            if (tx.from != null) {
                if ((0, address_1.getAddress)(tx.from) !== _this.address) {
                    logger.throwArgumentError("transaction from address mismatch", "transaction.from", transaction.from);
                }
                delete tx.from;
            }
            var signature = _this._signingKey().signDigest((0, keccak256_1.keccak256)((0, transactions_1.serialize)(tx)));
            return (0, transactions_1.serialize)(tx, signature);
        });
    };
    Wallet.prototype.signMessage = function (message) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                return [2 /*return*/, (0, bytes_1.joinSignature)(this._signingKey().signDigest((0, hash_1.hashMessage)(message)))];
            });
        });
    };
    Wallet.prototype._signTypedData = function (domain, types, value) {
        return __awaiter(this, void 0, void 0, function () {
            var populated;
            var _this = this;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, hash_1._TypedDataEncoder.resolveNames(domain, types, value, function (name) {
                            if (_this.provider == null) {
                                logger.throwError("cannot resolve ENS names without a provider", logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
                                    operation: "resolveName",
                                    value: name
                                });
                            }
                            return _this.provider.resolveName(name);
                        })];
                    case 1:
                        populated = _a.sent();
                        return [2 /*return*/, (0, bytes_1.joinSignature)(this._signingKey().signDigest(hash_1._TypedDataEncoder.hash(populated.domain, types, populated.value)))];
                }
            });
        });
    };
    Wallet.prototype.encrypt = function (password, options, progressCallback) {
        if (typeof (options) === "function" && !progressCallback) {
            progressCallback = options;
            options = {};
        }
        if (progressCallback && typeof (progressCallback) !== "function") {
            throw new Error("invalid callback");
        }
        if (!options) {
            options = {};
        }
        return (0, json_wallets_1.encryptKeystore)(this, password, options, progressCallback);
    };
    /**
     *  Static methods to create Wallet instances.
     */
    Wallet.createRandom = function (options) {
        var entropy = (0, random_1.randomBytes)(16);
        if (!options) {
            options = {};
        }
        if (options.extraEntropy) {
            entropy = (0, bytes_1.arrayify)((0, bytes_1.hexDataSlice)((0, keccak256_1.keccak256)((0, bytes_1.concat)([entropy, options.extraEntropy])), 0, 16));
        }
        var mnemonic = (0, hdnode_1.entropyToMnemonic)(entropy, options.locale);
        return Wallet.fromMnemonic(mnemonic, options.path, options.locale);
    };
    Wallet.fromEncryptedJson = function (json, password, progressCallback) {
        return (0, json_wallets_1.decryptJsonWallet)(json, password, progressCallback).then(function (account) {
            return new Wallet(account);
        });
    };
    Wallet.fromEncryptedJsonSync = function (json, password) {
        return new Wallet((0, json_wallets_1.decryptJsonWalletSync)(json, password));
    };
    Wallet.fromMnemonic = function (mnemonic, path, wordlist) {
        if (!path) {
            path = hdnode_1.defaultPath;
        }
        return new Wallet(hdnode_1.HDNode.fromMnemonic(mnemonic, null, wordlist).derivePath(path));
    };
    return Wallet;
}(abstract_signer_1.Signer));
exports.Wallet = Wallet;
function verifyMessage(message, signature) {
    return (0, transactions_1.recoverAddress)((0, hash_1.hashMessage)(message), signature);
}
exports.verifyMessage = verifyMessage;
function verifyTypedData(domain, types, value, signature) {
    return (0, transactions_1.recoverAddress)(hash_1._TypedDataEncoder.hash(domain, types, value), signature);
}
exports.verifyTypedData = verifyTypedData;
//# sourceMappingURL=index.js.map