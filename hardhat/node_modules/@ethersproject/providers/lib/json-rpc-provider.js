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
exports.JsonRpcProvider = exports.JsonRpcSigner = void 0;
var abstract_signer_1 = require("@ethersproject/abstract-signer");
var bignumber_1 = require("@ethersproject/bignumber");
var bytes_1 = require("@ethersproject/bytes");
var hash_1 = require("@ethersproject/hash");
var properties_1 = require("@ethersproject/properties");
var strings_1 = require("@ethersproject/strings");
var transactions_1 = require("@ethersproject/transactions");
var web_1 = require("@ethersproject/web");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var base_provider_1 = require("./base-provider");
var errorGas = ["call", "estimateGas"];
function spelunk(value, requireData) {
    if (value == null) {
        return null;
    }
    // These *are* the droids we're looking for.
    if (typeof (value.message) === "string" && value.message.match("reverted")) {
        var data = (0, bytes_1.isHexString)(value.data) ? value.data : null;
        if (!requireData || data) {
            return { message: value.message, data: data };
        }
    }
    // Spelunk further...
    if (typeof (value) === "object") {
        for (var key in value) {
            var result = spelunk(value[key], requireData);
            if (result) {
                return result;
            }
        }
        return null;
    }
    // Might be a JSON string we can further descend...
    if (typeof (value) === "string") {
        try {
            return spelunk(JSON.parse(value), requireData);
        }
        catch (error) { }
    }
    return null;
}
function checkError(method, error, params) {
    var transaction = params.transaction || params.signedTransaction;
    // Undo the "convenience" some nodes are attempting to prevent backwards
    // incompatibility; maybe for v6 consider forwarding reverts as errors
    if (method === "call") {
        var result = spelunk(error, true);
        if (result) {
            return result.data;
        }
        // Nothing descriptive..
        logger.throwError("missing revert data in call exception; Transaction reverted without a reason string", logger_1.Logger.errors.CALL_EXCEPTION, {
            data: "0x",
            transaction: transaction,
            error: error
        });
    }
    if (method === "estimateGas") {
        // Try to find something, with a preference on SERVER_ERROR body
        var result = spelunk(error.body, false);
        if (result == null) {
            result = spelunk(error, false);
        }
        // Found "reverted", this is a CALL_EXCEPTION
        if (result) {
            logger.throwError("cannot estimate gas; transaction may fail or may require manual gas limit", logger_1.Logger.errors.UNPREDICTABLE_GAS_LIMIT, {
                reason: result.message,
                method: method,
                transaction: transaction,
                error: error
            });
        }
    }
    // @TODO: Should we spelunk for message too?
    var message = error.message;
    if (error.code === logger_1.Logger.errors.SERVER_ERROR && error.error && typeof (error.error.message) === "string") {
        message = error.error.message;
    }
    else if (typeof (error.body) === "string") {
        message = error.body;
    }
    else if (typeof (error.responseText) === "string") {
        message = error.responseText;
    }
    message = (message || "").toLowerCase();
    // "insufficient funds for gas * price + value + cost(data)"
    if (message.match(/insufficient funds|base fee exceeds gas limit|InsufficientFunds/i)) {
        logger.throwError("insufficient funds for intrinsic transaction cost", logger_1.Logger.errors.INSUFFICIENT_FUNDS, {
            error: error,
            method: method,
            transaction: transaction
        });
    }
    // "nonce too low"
    if (message.match(/nonce (is )?too low/i)) {
        logger.throwError("nonce has already been used", logger_1.Logger.errors.NONCE_EXPIRED, {
            error: error,
            method: method,
            transaction: transaction
        });
    }
    // "replacement transaction underpriced"
    if (message.match(/replacement transaction underpriced|transaction gas price.*too low/i)) {
        logger.throwError("replacement fee too low", logger_1.Logger.errors.REPLACEMENT_UNDERPRICED, {
            error: error,
            method: method,
            transaction: transaction
        });
    }
    // "replacement transaction underpriced"
    if (message.match(/only replay-protected/i)) {
        logger.throwError("legacy pre-eip-155 transactions not supported", logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
            error: error,
            method: method,
            transaction: transaction
        });
    }
    if (errorGas.indexOf(method) >= 0 && message.match(/gas required exceeds allowance|always failing transaction|execution reverted|revert/)) {
        logger.throwError("cannot estimate gas; transaction may fail or may require manual gas limit", logger_1.Logger.errors.UNPREDICTABLE_GAS_LIMIT, {
            error: error,
            method: method,
            transaction: transaction
        });
    }
    throw error;
}
function timer(timeout) {
    return new Promise(function (resolve) {
        setTimeout(resolve, timeout);
    });
}
function getResult(payload) {
    if (payload.error) {
        // @TODO: not any
        var error = new Error(payload.error.message);
        error.code = payload.error.code;
        error.data = payload.error.data;
        throw error;
    }
    return payload.result;
}
function getLowerCase(value) {
    if (value) {
        return value.toLowerCase();
    }
    return value;
}
var _constructorGuard = {};
var JsonRpcSigner = /** @class */ (function (_super) {
    __extends(JsonRpcSigner, _super);
    function JsonRpcSigner(constructorGuard, provider, addressOrIndex) {
        var _this = _super.call(this) || this;
        if (constructorGuard !== _constructorGuard) {
            throw new Error("do not call the JsonRpcSigner constructor directly; use provider.getSigner");
        }
        (0, properties_1.defineReadOnly)(_this, "provider", provider);
        if (addressOrIndex == null) {
            addressOrIndex = 0;
        }
        if (typeof (addressOrIndex) === "string") {
            (0, properties_1.defineReadOnly)(_this, "_address", _this.provider.formatter.address(addressOrIndex));
            (0, properties_1.defineReadOnly)(_this, "_index", null);
        }
        else if (typeof (addressOrIndex) === "number") {
            (0, properties_1.defineReadOnly)(_this, "_index", addressOrIndex);
            (0, properties_1.defineReadOnly)(_this, "_address", null);
        }
        else {
            logger.throwArgumentError("invalid address or index", "addressOrIndex", addressOrIndex);
        }
        return _this;
    }
    JsonRpcSigner.prototype.connect = function (provider) {
        return logger.throwError("cannot alter JSON-RPC Signer connection", logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
            operation: "connect"
        });
    };
    JsonRpcSigner.prototype.connectUnchecked = function () {
        return new UncheckedJsonRpcSigner(_constructorGuard, this.provider, this._address || this._index);
    };
    JsonRpcSigner.prototype.getAddress = function () {
        var _this = this;
        if (this._address) {
            return Promise.resolve(this._address);
        }
        return this.provider.send("eth_accounts", []).then(function (accounts) {
            if (accounts.length <= _this._index) {
                logger.throwError("unknown account #" + _this._index, logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
                    operation: "getAddress"
                });
            }
            return _this.provider.formatter.address(accounts[_this._index]);
        });
    };
    JsonRpcSigner.prototype.sendUncheckedTransaction = function (transaction) {
        var _this = this;
        transaction = (0, properties_1.shallowCopy)(transaction);
        var fromAddress = this.getAddress().then(function (address) {
            if (address) {
                address = address.toLowerCase();
            }
            return address;
        });
        // The JSON-RPC for eth_sendTransaction uses 90000 gas; if the user
        // wishes to use this, it is easy to specify explicitly, otherwise
        // we look it up for them.
        if (transaction.gasLimit == null) {
            var estimate = (0, properties_1.shallowCopy)(transaction);
            estimate.from = fromAddress;
            transaction.gasLimit = this.provider.estimateGas(estimate);
        }
        if (transaction.to != null) {
            transaction.to = Promise.resolve(transaction.to).then(function (to) { return __awaiter(_this, void 0, void 0, function () {
                var address;
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (to == null) {
                                return [2 /*return*/, null];
                            }
                            return [4 /*yield*/, this.provider.resolveName(to)];
                        case 1:
                            address = _a.sent();
                            if (address == null) {
                                logger.throwArgumentError("provided ENS name resolves to null", "tx.to", to);
                            }
                            return [2 /*return*/, address];
                    }
                });
            }); });
        }
        return (0, properties_1.resolveProperties)({
            tx: (0, properties_1.resolveProperties)(transaction),
            sender: fromAddress
        }).then(function (_a) {
            var tx = _a.tx, sender = _a.sender;
            if (tx.from != null) {
                if (tx.from.toLowerCase() !== sender) {
                    logger.throwArgumentError("from address mismatch", "transaction", transaction);
                }
            }
            else {
                tx.from = sender;
            }
            var hexTx = _this.provider.constructor.hexlifyTransaction(tx, { from: true });
            return _this.provider.send("eth_sendTransaction", [hexTx]).then(function (hash) {
                return hash;
            }, function (error) {
                if (typeof (error.message) === "string" && error.message.match(/user denied/i)) {
                    logger.throwError("user rejected transaction", logger_1.Logger.errors.ACTION_REJECTED, {
                        action: "sendTransaction",
                        transaction: tx
                    });
                }
                return checkError("sendTransaction", error, hexTx);
            });
        });
    };
    JsonRpcSigner.prototype.signTransaction = function (transaction) {
        return logger.throwError("signing transactions is unsupported", logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
            operation: "signTransaction"
        });
    };
    JsonRpcSigner.prototype.sendTransaction = function (transaction) {
        return __awaiter(this, void 0, void 0, function () {
            var blockNumber, hash, error_1;
            var _this = this;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.provider._getInternalBlockNumber(100 + 2 * this.provider.pollingInterval)];
                    case 1:
                        blockNumber = _a.sent();
                        return [4 /*yield*/, this.sendUncheckedTransaction(transaction)];
                    case 2:
                        hash = _a.sent();
                        _a.label = 3;
                    case 3:
                        _a.trys.push([3, 5, , 6]);
                        return [4 /*yield*/, (0, web_1.poll)(function () { return __awaiter(_this, void 0, void 0, function () {
                                var tx;
                                return __generator(this, function (_a) {
                                    switch (_a.label) {
                                        case 0: return [4 /*yield*/, this.provider.getTransaction(hash)];
                                        case 1:
                                            tx = _a.sent();
                                            if (tx === null) {
                                                return [2 /*return*/, undefined];
                                            }
                                            return [2 /*return*/, this.provider._wrapTransaction(tx, hash, blockNumber)];
                                    }
                                });
                            }); }, { oncePoll: this.provider })];
                    case 4: 
                    // Unfortunately, JSON-RPC only provides and opaque transaction hash
                    // for a response, and we need the actual transaction, so we poll
                    // for it; it should show up very quickly
                    return [2 /*return*/, _a.sent()];
                    case 5:
                        error_1 = _a.sent();
                        error_1.transactionHash = hash;
                        throw error_1;
                    case 6: return [2 /*return*/];
                }
            });
        });
    };
    JsonRpcSigner.prototype.signMessage = function (message) {
        return __awaiter(this, void 0, void 0, function () {
            var data, address, error_2;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        data = ((typeof (message) === "string") ? (0, strings_1.toUtf8Bytes)(message) : message);
                        return [4 /*yield*/, this.getAddress()];
                    case 1:
                        address = _a.sent();
                        _a.label = 2;
                    case 2:
                        _a.trys.push([2, 4, , 5]);
                        return [4 /*yield*/, this.provider.send("personal_sign", [(0, bytes_1.hexlify)(data), address.toLowerCase()])];
                    case 3: return [2 /*return*/, _a.sent()];
                    case 4:
                        error_2 = _a.sent();
                        if (typeof (error_2.message) === "string" && error_2.message.match(/user denied/i)) {
                            logger.throwError("user rejected signing", logger_1.Logger.errors.ACTION_REJECTED, {
                                action: "signMessage",
                                from: address,
                                messageData: message
                            });
                        }
                        throw error_2;
                    case 5: return [2 /*return*/];
                }
            });
        });
    };
    JsonRpcSigner.prototype._legacySignMessage = function (message) {
        return __awaiter(this, void 0, void 0, function () {
            var data, address, error_3;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        data = ((typeof (message) === "string") ? (0, strings_1.toUtf8Bytes)(message) : message);
                        return [4 /*yield*/, this.getAddress()];
                    case 1:
                        address = _a.sent();
                        _a.label = 2;
                    case 2:
                        _a.trys.push([2, 4, , 5]);
                        return [4 /*yield*/, this.provider.send("eth_sign", [address.toLowerCase(), (0, bytes_1.hexlify)(data)])];
                    case 3: 
                    // https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_sign
                    return [2 /*return*/, _a.sent()];
                    case 4:
                        error_3 = _a.sent();
                        if (typeof (error_3.message) === "string" && error_3.message.match(/user denied/i)) {
                            logger.throwError("user rejected signing", logger_1.Logger.errors.ACTION_REJECTED, {
                                action: "_legacySignMessage",
                                from: address,
                                messageData: message
                            });
                        }
                        throw error_3;
                    case 5: return [2 /*return*/];
                }
            });
        });
    };
    JsonRpcSigner.prototype._signTypedData = function (domain, types, value) {
        return __awaiter(this, void 0, void 0, function () {
            var populated, address, error_4;
            var _this = this;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, hash_1._TypedDataEncoder.resolveNames(domain, types, value, function (name) {
                            return _this.provider.resolveName(name);
                        })];
                    case 1:
                        populated = _a.sent();
                        return [4 /*yield*/, this.getAddress()];
                    case 2:
                        address = _a.sent();
                        _a.label = 3;
                    case 3:
                        _a.trys.push([3, 5, , 6]);
                        return [4 /*yield*/, this.provider.send("eth_signTypedData_v4", [
                                address.toLowerCase(),
                                JSON.stringify(hash_1._TypedDataEncoder.getPayload(populated.domain, types, populated.value))
                            ])];
                    case 4: return [2 /*return*/, _a.sent()];
                    case 5:
                        error_4 = _a.sent();
                        if (typeof (error_4.message) === "string" && error_4.message.match(/user denied/i)) {
                            logger.throwError("user rejected signing", logger_1.Logger.errors.ACTION_REJECTED, {
                                action: "_signTypedData",
                                from: address,
                                messageData: { domain: populated.domain, types: types, value: populated.value }
                            });
                        }
                        throw error_4;
                    case 6: return [2 /*return*/];
                }
            });
        });
    };
    JsonRpcSigner.prototype.unlock = function (password) {
        return __awaiter(this, void 0, void 0, function () {
            var provider, address;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        provider = this.provider;
                        return [4 /*yield*/, this.getAddress()];
                    case 1:
                        address = _a.sent();
                        return [2 /*return*/, provider.send("personal_unlockAccount", [address.toLowerCase(), password, null])];
                }
            });
        });
    };
    return JsonRpcSigner;
}(abstract_signer_1.Signer));
exports.JsonRpcSigner = JsonRpcSigner;
var UncheckedJsonRpcSigner = /** @class */ (function (_super) {
    __extends(UncheckedJsonRpcSigner, _super);
    function UncheckedJsonRpcSigner() {
        return _super !== null && _super.apply(this, arguments) || this;
    }
    UncheckedJsonRpcSigner.prototype.sendTransaction = function (transaction) {
        var _this = this;
        return this.sendUncheckedTransaction(transaction).then(function (hash) {
            return {
                hash: hash,
                nonce: null,
                gasLimit: null,
                gasPrice: null,
                data: null,
                value: null,
                chainId: null,
                confirmations: 0,
                from: null,
                wait: function (confirmations) { return _this.provider.waitForTransaction(hash, confirmations); }
            };
        });
    };
    return UncheckedJsonRpcSigner;
}(JsonRpcSigner));
var allowedTransactionKeys = {
    chainId: true, data: true, gasLimit: true, gasPrice: true, nonce: true, to: true, value: true,
    type: true, accessList: true,
    maxFeePerGas: true, maxPriorityFeePerGas: true
};
var JsonRpcProvider = /** @class */ (function (_super) {
    __extends(JsonRpcProvider, _super);
    function JsonRpcProvider(url, network) {
        var _this = this;
        var networkOrReady = network;
        // The network is unknown, query the JSON-RPC for it
        if (networkOrReady == null) {
            networkOrReady = new Promise(function (resolve, reject) {
                setTimeout(function () {
                    _this.detectNetwork().then(function (network) {
                        resolve(network);
                    }, function (error) {
                        reject(error);
                    });
                }, 0);
            });
        }
        _this = _super.call(this, networkOrReady) || this;
        // Default URL
        if (!url) {
            url = (0, properties_1.getStatic)(_this.constructor, "defaultUrl")();
        }
        if (typeof (url) === "string") {
            (0, properties_1.defineReadOnly)(_this, "connection", Object.freeze({
                url: url
            }));
        }
        else {
            (0, properties_1.defineReadOnly)(_this, "connection", Object.freeze((0, properties_1.shallowCopy)(url)));
        }
        _this._nextId = 42;
        return _this;
    }
    Object.defineProperty(JsonRpcProvider.prototype, "_cache", {
        get: function () {
            if (this._eventLoopCache == null) {
                this._eventLoopCache = {};
            }
            return this._eventLoopCache;
        },
        enumerable: false,
        configurable: true
    });
    JsonRpcProvider.defaultUrl = function () {
        return "http:/\/localhost:8545";
    };
    JsonRpcProvider.prototype.detectNetwork = function () {
        var _this = this;
        if (!this._cache["detectNetwork"]) {
            this._cache["detectNetwork"] = this._uncachedDetectNetwork();
            // Clear this cache at the beginning of the next event loop
            setTimeout(function () {
                _this._cache["detectNetwork"] = null;
            }, 0);
        }
        return this._cache["detectNetwork"];
    };
    JsonRpcProvider.prototype._uncachedDetectNetwork = function () {
        return __awaiter(this, void 0, void 0, function () {
            var chainId, error_5, error_6, getNetwork;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, timer(0)];
                    case 1:
                        _a.sent();
                        chainId = null;
                        _a.label = 2;
                    case 2:
                        _a.trys.push([2, 4, , 9]);
                        return [4 /*yield*/, this.send("eth_chainId", [])];
                    case 3:
                        chainId = _a.sent();
                        return [3 /*break*/, 9];
                    case 4:
                        error_5 = _a.sent();
                        _a.label = 5;
                    case 5:
                        _a.trys.push([5, 7, , 8]);
                        return [4 /*yield*/, this.send("net_version", [])];
                    case 6:
                        chainId = _a.sent();
                        return [3 /*break*/, 8];
                    case 7:
                        error_6 = _a.sent();
                        return [3 /*break*/, 8];
                    case 8: return [3 /*break*/, 9];
                    case 9:
                        if (chainId != null) {
                            getNetwork = (0, properties_1.getStatic)(this.constructor, "getNetwork");
                            try {
                                return [2 /*return*/, getNetwork(bignumber_1.BigNumber.from(chainId).toNumber())];
                            }
                            catch (error) {
                                return [2 /*return*/, logger.throwError("could not detect network", logger_1.Logger.errors.NETWORK_ERROR, {
                                        chainId: chainId,
                                        event: "invalidNetwork",
                                        serverError: error
                                    })];
                            }
                        }
                        return [2 /*return*/, logger.throwError("could not detect network", logger_1.Logger.errors.NETWORK_ERROR, {
                                event: "noNetwork"
                            })];
                }
            });
        });
    };
    JsonRpcProvider.prototype.getSigner = function (addressOrIndex) {
        return new JsonRpcSigner(_constructorGuard, this, addressOrIndex);
    };
    JsonRpcProvider.prototype.getUncheckedSigner = function (addressOrIndex) {
        return this.getSigner(addressOrIndex).connectUnchecked();
    };
    JsonRpcProvider.prototype.listAccounts = function () {
        var _this = this;
        return this.send("eth_accounts", []).then(function (accounts) {
            return accounts.map(function (a) { return _this.formatter.address(a); });
        });
    };
    JsonRpcProvider.prototype.send = function (method, params) {
        var _this = this;
        var request = {
            method: method,
            params: params,
            id: (this._nextId++),
            jsonrpc: "2.0"
        };
        this.emit("debug", {
            action: "request",
            request: (0, properties_1.deepCopy)(request),
            provider: this
        });
        // We can expand this in the future to any call, but for now these
        // are the biggest wins and do not require any serializing parameters.
        var cache = (["eth_chainId", "eth_blockNumber"].indexOf(method) >= 0);
        if (cache && this._cache[method]) {
            return this._cache[method];
        }
        var result = (0, web_1.fetchJson)(this.connection, JSON.stringify(request), getResult).then(function (result) {
            _this.emit("debug", {
                action: "response",
                request: request,
                response: result,
                provider: _this
            });
            return result;
        }, function (error) {
            _this.emit("debug", {
                action: "response",
                error: error,
                request: request,
                provider: _this
            });
            throw error;
        });
        // Cache the fetch, but clear it on the next event loop
        if (cache) {
            this._cache[method] = result;
            setTimeout(function () {
                _this._cache[method] = null;
            }, 0);
        }
        return result;
    };
    JsonRpcProvider.prototype.prepareRequest = function (method, params) {
        switch (method) {
            case "getBlockNumber":
                return ["eth_blockNumber", []];
            case "getGasPrice":
                return ["eth_gasPrice", []];
            case "getBalance":
                return ["eth_getBalance", [getLowerCase(params.address), params.blockTag]];
            case "getTransactionCount":
                return ["eth_getTransactionCount", [getLowerCase(params.address), params.blockTag]];
            case "getCode":
                return ["eth_getCode", [getLowerCase(params.address), params.blockTag]];
            case "getStorageAt":
                return ["eth_getStorageAt", [getLowerCase(params.address), (0, bytes_1.hexZeroPad)(params.position, 32), params.blockTag]];
            case "sendTransaction":
                return ["eth_sendRawTransaction", [params.signedTransaction]];
            case "getBlock":
                if (params.blockTag) {
                    return ["eth_getBlockByNumber", [params.blockTag, !!params.includeTransactions]];
                }
                else if (params.blockHash) {
                    return ["eth_getBlockByHash", [params.blockHash, !!params.includeTransactions]];
                }
                return null;
            case "getTransaction":
                return ["eth_getTransactionByHash", [params.transactionHash]];
            case "getTransactionReceipt":
                return ["eth_getTransactionReceipt", [params.transactionHash]];
            case "call": {
                var hexlifyTransaction = (0, properties_1.getStatic)(this.constructor, "hexlifyTransaction");
                return ["eth_call", [hexlifyTransaction(params.transaction, { from: true }), params.blockTag]];
            }
            case "estimateGas": {
                var hexlifyTransaction = (0, properties_1.getStatic)(this.constructor, "hexlifyTransaction");
                return ["eth_estimateGas", [hexlifyTransaction(params.transaction, { from: true })]];
            }
            case "getLogs":
                if (params.filter && params.filter.address != null) {
                    params.filter.address = getLowerCase(params.filter.address);
                }
                return ["eth_getLogs", [params.filter]];
            default:
                break;
        }
        return null;
    };
    JsonRpcProvider.prototype.perform = function (method, params) {
        return __awaiter(this, void 0, void 0, function () {
            var tx, feeData, args, error_7;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!(method === "call" || method === "estimateGas")) return [3 /*break*/, 2];
                        tx = params.transaction;
                        if (!(tx && tx.type != null && bignumber_1.BigNumber.from(tx.type).isZero())) return [3 /*break*/, 2];
                        if (!(tx.maxFeePerGas == null && tx.maxPriorityFeePerGas == null)) return [3 /*break*/, 2];
                        return [4 /*yield*/, this.getFeeData()];
                    case 1:
                        feeData = _a.sent();
                        if (feeData.maxFeePerGas == null && feeData.maxPriorityFeePerGas == null) {
                            // Network doesn't know about EIP-1559 (and hence type)
                            params = (0, properties_1.shallowCopy)(params);
                            params.transaction = (0, properties_1.shallowCopy)(tx);
                            delete params.transaction.type;
                        }
                        _a.label = 2;
                    case 2:
                        args = this.prepareRequest(method, params);
                        if (args == null) {
                            logger.throwError(method + " not implemented", logger_1.Logger.errors.NOT_IMPLEMENTED, { operation: method });
                        }
                        _a.label = 3;
                    case 3:
                        _a.trys.push([3, 5, , 6]);
                        return [4 /*yield*/, this.send(args[0], args[1])];
                    case 4: return [2 /*return*/, _a.sent()];
                    case 5:
                        error_7 = _a.sent();
                        return [2 /*return*/, checkError(method, error_7, params)];
                    case 6: return [2 /*return*/];
                }
            });
        });
    };
    JsonRpcProvider.prototype._startEvent = function (event) {
        if (event.tag === "pending") {
            this._startPending();
        }
        _super.prototype._startEvent.call(this, event);
    };
    JsonRpcProvider.prototype._startPending = function () {
        if (this._pendingFilter != null) {
            return;
        }
        var self = this;
        var pendingFilter = this.send("eth_newPendingTransactionFilter", []);
        this._pendingFilter = pendingFilter;
        pendingFilter.then(function (filterId) {
            function poll() {
                self.send("eth_getFilterChanges", [filterId]).then(function (hashes) {
                    if (self._pendingFilter != pendingFilter) {
                        return null;
                    }
                    var seq = Promise.resolve();
                    hashes.forEach(function (hash) {
                        // @TODO: This should be garbage collected at some point... How? When?
                        self._emitted["t:" + hash.toLowerCase()] = "pending";
                        seq = seq.then(function () {
                            return self.getTransaction(hash).then(function (tx) {
                                self.emit("pending", tx);
                                return null;
                            });
                        });
                    });
                    return seq.then(function () {
                        return timer(1000);
                    });
                }).then(function () {
                    if (self._pendingFilter != pendingFilter) {
                        self.send("eth_uninstallFilter", [filterId]);
                        return;
                    }
                    setTimeout(function () { poll(); }, 0);
                    return null;
                }).catch(function (error) { });
            }
            poll();
            return filterId;
        }).catch(function (error) { });
    };
    JsonRpcProvider.prototype._stopEvent = function (event) {
        if (event.tag === "pending" && this.listenerCount("pending") === 0) {
            this._pendingFilter = null;
        }
        _super.prototype._stopEvent.call(this, event);
    };
    // Convert an ethers.js transaction into a JSON-RPC transaction
    //  - gasLimit => gas
    //  - All values hexlified
    //  - All numeric values zero-striped
    //  - All addresses are lowercased
    // NOTE: This allows a TransactionRequest, but all values should be resolved
    //       before this is called
    // @TODO: This will likely be removed in future versions and prepareRequest
    //        will be the preferred method for this.
    JsonRpcProvider.hexlifyTransaction = function (transaction, allowExtra) {
        // Check only allowed properties are given
        var allowed = (0, properties_1.shallowCopy)(allowedTransactionKeys);
        if (allowExtra) {
            for (var key in allowExtra) {
                if (allowExtra[key]) {
                    allowed[key] = true;
                }
            }
        }
        (0, properties_1.checkProperties)(transaction, allowed);
        var result = {};
        // JSON-RPC now requires numeric values to be "quantity" values
        ["chainId", "gasLimit", "gasPrice", "type", "maxFeePerGas", "maxPriorityFeePerGas", "nonce", "value"].forEach(function (key) {
            if (transaction[key] == null) {
                return;
            }
            var value = (0, bytes_1.hexValue)(bignumber_1.BigNumber.from(transaction[key]));
            if (key === "gasLimit") {
                key = "gas";
            }
            result[key] = value;
        });
        ["from", "to", "data"].forEach(function (key) {
            if (transaction[key] == null) {
                return;
            }
            result[key] = (0, bytes_1.hexlify)(transaction[key]);
        });
        if (transaction.accessList) {
            result["accessList"] = (0, transactions_1.accessListify)(transaction.accessList);
        }
        return result;
    };
    return JsonRpcProvider;
}(base_provider_1.BaseProvider));
exports.JsonRpcProvider = JsonRpcProvider;
//# sourceMappingURL=json-rpc-provider.js.map