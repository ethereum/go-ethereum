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
exports.EtherscanProvider = void 0;
var bytes_1 = require("@ethersproject/bytes");
var properties_1 = require("@ethersproject/properties");
var transactions_1 = require("@ethersproject/transactions");
var web_1 = require("@ethersproject/web");
var formatter_1 = require("./formatter");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var base_provider_1 = require("./base-provider");
// The transaction has already been sanitized by the calls in Provider
function getTransactionPostData(transaction) {
    var result = {};
    for (var key in transaction) {
        if (transaction[key] == null) {
            continue;
        }
        var value = transaction[key];
        if (key === "type" && value === 0) {
            continue;
        }
        // Quantity-types require no leading zero, unless 0
        if ({ type: true, gasLimit: true, gasPrice: true, maxFeePerGs: true, maxPriorityFeePerGas: true, nonce: true, value: true }[key]) {
            value = (0, bytes_1.hexValue)((0, bytes_1.hexlify)(value));
        }
        else if (key === "accessList") {
            value = "[" + (0, transactions_1.accessListify)(value).map(function (set) {
                return "{address:\"" + set.address + "\",storageKeys:[\"" + set.storageKeys.join('","') + "\"]}";
            }).join(",") + "]";
        }
        else {
            value = (0, bytes_1.hexlify)(value);
        }
        result[key] = value;
    }
    return result;
}
function getResult(result) {
    // getLogs, getHistory have weird success responses
    if (result.status == 0 && (result.message === "No records found" || result.message === "No transactions found")) {
        return result.result;
    }
    if (result.status != 1 || typeof (result.message) !== "string" || !result.message.match(/^OK/)) {
        var error = new Error("invalid response");
        error.result = JSON.stringify(result);
        if ((result.result || "").toLowerCase().indexOf("rate limit") >= 0) {
            error.throttleRetry = true;
        }
        throw error;
    }
    return result.result;
}
function getJsonResult(result) {
    // This response indicates we are being throttled
    if (result && result.status == 0 && result.message == "NOTOK" && (result.result || "").toLowerCase().indexOf("rate limit") >= 0) {
        var error = new Error("throttled response");
        error.result = JSON.stringify(result);
        error.throttleRetry = true;
        throw error;
    }
    if (result.jsonrpc != "2.0") {
        // @TODO: not any
        var error = new Error("invalid response");
        error.result = JSON.stringify(result);
        throw error;
    }
    if (result.error) {
        // @TODO: not any
        var error = new Error(result.error.message || "unknown error");
        if (result.error.code) {
            error.code = result.error.code;
        }
        if (result.error.data) {
            error.data = result.error.data;
        }
        throw error;
    }
    return result.result;
}
// The blockTag was normalized as a string by the Provider pre-perform operations
function checkLogTag(blockTag) {
    if (blockTag === "pending") {
        throw new Error("pending not supported");
    }
    if (blockTag === "latest") {
        return blockTag;
    }
    return parseInt(blockTag.substring(2), 16);
}
function checkError(method, error, transaction) {
    // Undo the "convenience" some nodes are attempting to prevent backwards
    // incompatibility; maybe for v6 consider forwarding reverts as errors
    if (method === "call" && error.code === logger_1.Logger.errors.SERVER_ERROR) {
        var e = error.error;
        // Etherscan keeps changing their string
        if (e && (e.message.match(/reverted/i) || e.message.match(/VM execution error/i))) {
            // Etherscan prefixes the data like "Reverted 0x1234"
            var data = e.data;
            if (data) {
                data = "0x" + data.replace(/^.*0x/i, "");
            }
            if ((0, bytes_1.isHexString)(data)) {
                return data;
            }
            logger.throwError("missing revert data in call exception", logger_1.Logger.errors.CALL_EXCEPTION, {
                error: error,
                data: "0x"
            });
        }
    }
    // Get the message from any nested error structure
    var message = error.message;
    if (error.code === logger_1.Logger.errors.SERVER_ERROR) {
        if (error.error && typeof (error.error.message) === "string") {
            message = error.error.message;
        }
        else if (typeof (error.body) === "string") {
            message = error.body;
        }
        else if (typeof (error.responseText) === "string") {
            message = error.responseText;
        }
    }
    message = (message || "").toLowerCase();
    // "Insufficient funds. The account you tried to send transaction from does not have enough funds. Required 21464000000000 and got: 0"
    if (message.match(/insufficient funds/)) {
        logger.throwError("insufficient funds for intrinsic transaction cost", logger_1.Logger.errors.INSUFFICIENT_FUNDS, {
            error: error,
            method: method,
            transaction: transaction
        });
    }
    // "Transaction with the same hash was already imported."
    if (message.match(/same hash was already imported|transaction nonce is too low|nonce too low/)) {
        logger.throwError("nonce has already been used", logger_1.Logger.errors.NONCE_EXPIRED, {
            error: error,
            method: method,
            transaction: transaction
        });
    }
    // "Transaction gas price is too low. There is another transaction with same nonce in the queue. Try increasing the gas price or incrementing the nonce."
    if (message.match(/another transaction with same nonce/)) {
        logger.throwError("replacement fee too low", logger_1.Logger.errors.REPLACEMENT_UNDERPRICED, {
            error: error,
            method: method,
            transaction: transaction
        });
    }
    if (message.match(/execution failed due to an exception|execution reverted/)) {
        logger.throwError("cannot estimate gas; transaction may fail or may require manual gas limit", logger_1.Logger.errors.UNPREDICTABLE_GAS_LIMIT, {
            error: error,
            method: method,
            transaction: transaction
        });
    }
    throw error;
}
var EtherscanProvider = /** @class */ (function (_super) {
    __extends(EtherscanProvider, _super);
    function EtherscanProvider(network, apiKey) {
        var _this = _super.call(this, network) || this;
        (0, properties_1.defineReadOnly)(_this, "baseUrl", _this.getBaseUrl());
        (0, properties_1.defineReadOnly)(_this, "apiKey", apiKey || null);
        return _this;
    }
    EtherscanProvider.prototype.getBaseUrl = function () {
        switch (this.network ? this.network.name : "invalid") {
            case "homestead":
                return "https:/\/api.etherscan.io";
            case "goerli":
                return "https:/\/api-goerli.etherscan.io";
            case "sepolia":
                return "https:/\/api-sepolia.etherscan.io";
            case "matic":
                return "https:/\/api.polygonscan.com";
            case "maticmum":
                return "https:/\/api-testnet.polygonscan.com";
            case "arbitrum":
                return "https:/\/api.arbiscan.io";
            case "arbitrum-goerli":
                return "https:/\/api-goerli.arbiscan.io";
            case "optimism":
                return "https:/\/api-optimistic.etherscan.io";
            case "optimism-goerli":
                return "https:/\/api-goerli-optimistic.etherscan.io";
            default:
        }
        return logger.throwArgumentError("unsupported network", "network", this.network.name);
    };
    EtherscanProvider.prototype.getUrl = function (module, params) {
        var query = Object.keys(params).reduce(function (accum, key) {
            var value = params[key];
            if (value != null) {
                accum += "&" + key + "=" + value;
            }
            return accum;
        }, "");
        var apiKey = ((this.apiKey) ? "&apikey=" + this.apiKey : "");
        return this.baseUrl + "/api?module=" + module + query + apiKey;
    };
    EtherscanProvider.prototype.getPostUrl = function () {
        return this.baseUrl + "/api";
    };
    EtherscanProvider.prototype.getPostData = function (module, params) {
        params.module = module;
        params.apikey = this.apiKey;
        return params;
    };
    EtherscanProvider.prototype.fetch = function (module, params, post) {
        return __awaiter(this, void 0, void 0, function () {
            var url, payload, procFunc, connection, payloadStr, result;
            var _this = this;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        url = (post ? this.getPostUrl() : this.getUrl(module, params));
                        payload = (post ? this.getPostData(module, params) : null);
                        procFunc = (module === "proxy") ? getJsonResult : getResult;
                        this.emit("debug", {
                            action: "request",
                            request: url,
                            provider: this
                        });
                        connection = {
                            url: url,
                            throttleSlotInterval: 1000,
                            throttleCallback: function (attempt, url) {
                                if (_this.isCommunityResource()) {
                                    (0, formatter_1.showThrottleMessage)();
                                }
                                return Promise.resolve(true);
                            }
                        };
                        payloadStr = null;
                        if (payload) {
                            connection.headers = { "content-type": "application/x-www-form-urlencoded; charset=UTF-8" };
                            payloadStr = Object.keys(payload).map(function (key) {
                                return key + "=" + payload[key];
                            }).join("&");
                        }
                        return [4 /*yield*/, (0, web_1.fetchJson)(connection, payloadStr, procFunc || getJsonResult)];
                    case 1:
                        result = _a.sent();
                        this.emit("debug", {
                            action: "response",
                            request: url,
                            response: (0, properties_1.deepCopy)(result),
                            provider: this
                        });
                        return [2 /*return*/, result];
                }
            });
        });
    };
    EtherscanProvider.prototype.detectNetwork = function () {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                return [2 /*return*/, this.network];
            });
        });
    };
    EtherscanProvider.prototype.perform = function (method, params) {
        return __awaiter(this, void 0, void 0, function () {
            var _a, postData, error_1, postData, error_2, args, topic0, logs, blocks, i, log, block, _b;
            return __generator(this, function (_c) {
                switch (_c.label) {
                    case 0:
                        _a = method;
                        switch (_a) {
                            case "getBlockNumber": return [3 /*break*/, 1];
                            case "getGasPrice": return [3 /*break*/, 2];
                            case "getBalance": return [3 /*break*/, 3];
                            case "getTransactionCount": return [3 /*break*/, 4];
                            case "getCode": return [3 /*break*/, 5];
                            case "getStorageAt": return [3 /*break*/, 6];
                            case "sendTransaction": return [3 /*break*/, 7];
                            case "getBlock": return [3 /*break*/, 8];
                            case "getTransaction": return [3 /*break*/, 9];
                            case "getTransactionReceipt": return [3 /*break*/, 10];
                            case "call": return [3 /*break*/, 11];
                            case "estimateGas": return [3 /*break*/, 15];
                            case "getLogs": return [3 /*break*/, 19];
                            case "getEtherPrice": return [3 /*break*/, 26];
                        }
                        return [3 /*break*/, 28];
                    case 1: return [2 /*return*/, this.fetch("proxy", { action: "eth_blockNumber" })];
                    case 2: return [2 /*return*/, this.fetch("proxy", { action: "eth_gasPrice" })];
                    case 3: 
                    // Returns base-10 result
                    return [2 /*return*/, this.fetch("account", {
                            action: "balance",
                            address: params.address,
                            tag: params.blockTag
                        })];
                    case 4: return [2 /*return*/, this.fetch("proxy", {
                            action: "eth_getTransactionCount",
                            address: params.address,
                            tag: params.blockTag
                        })];
                    case 5: return [2 /*return*/, this.fetch("proxy", {
                            action: "eth_getCode",
                            address: params.address,
                            tag: params.blockTag
                        })];
                    case 6: return [2 /*return*/, this.fetch("proxy", {
                            action: "eth_getStorageAt",
                            address: params.address,
                            position: params.position,
                            tag: params.blockTag
                        })];
                    case 7: return [2 /*return*/, this.fetch("proxy", {
                            action: "eth_sendRawTransaction",
                            hex: params.signedTransaction
                        }, true).catch(function (error) {
                            return checkError("sendTransaction", error, params.signedTransaction);
                        })];
                    case 8:
                        if (params.blockTag) {
                            return [2 /*return*/, this.fetch("proxy", {
                                    action: "eth_getBlockByNumber",
                                    tag: params.blockTag,
                                    boolean: (params.includeTransactions ? "true" : "false")
                                })];
                        }
                        throw new Error("getBlock by blockHash not implemented");
                    case 9: return [2 /*return*/, this.fetch("proxy", {
                            action: "eth_getTransactionByHash",
                            txhash: params.transactionHash
                        })];
                    case 10: return [2 /*return*/, this.fetch("proxy", {
                            action: "eth_getTransactionReceipt",
                            txhash: params.transactionHash
                        })];
                    case 11:
                        if (params.blockTag !== "latest") {
                            throw new Error("EtherscanProvider does not support blockTag for call");
                        }
                        postData = getTransactionPostData(params.transaction);
                        postData.module = "proxy";
                        postData.action = "eth_call";
                        _c.label = 12;
                    case 12:
                        _c.trys.push([12, 14, , 15]);
                        return [4 /*yield*/, this.fetch("proxy", postData, true)];
                    case 13: return [2 /*return*/, _c.sent()];
                    case 14:
                        error_1 = _c.sent();
                        return [2 /*return*/, checkError("call", error_1, params.transaction)];
                    case 15:
                        postData = getTransactionPostData(params.transaction);
                        postData.module = "proxy";
                        postData.action = "eth_estimateGas";
                        _c.label = 16;
                    case 16:
                        _c.trys.push([16, 18, , 19]);
                        return [4 /*yield*/, this.fetch("proxy", postData, true)];
                    case 17: return [2 /*return*/, _c.sent()];
                    case 18:
                        error_2 = _c.sent();
                        return [2 /*return*/, checkError("estimateGas", error_2, params.transaction)];
                    case 19:
                        args = { action: "getLogs" };
                        if (params.filter.fromBlock) {
                            args.fromBlock = checkLogTag(params.filter.fromBlock);
                        }
                        if (params.filter.toBlock) {
                            args.toBlock = checkLogTag(params.filter.toBlock);
                        }
                        if (params.filter.address) {
                            args.address = params.filter.address;
                        }
                        // @TODO: We can handle slightly more complicated logs using the logs API
                        if (params.filter.topics && params.filter.topics.length > 0) {
                            if (params.filter.topics.length > 1) {
                                logger.throwError("unsupported topic count", logger_1.Logger.errors.UNSUPPORTED_OPERATION, { topics: params.filter.topics });
                            }
                            if (params.filter.topics.length === 1) {
                                topic0 = params.filter.topics[0];
                                if (typeof (topic0) !== "string" || topic0.length !== 66) {
                                    logger.throwError("unsupported topic format", logger_1.Logger.errors.UNSUPPORTED_OPERATION, { topic0: topic0 });
                                }
                                args.topic0 = topic0;
                            }
                        }
                        return [4 /*yield*/, this.fetch("logs", args)];
                    case 20:
                        logs = _c.sent();
                        blocks = {};
                        i = 0;
                        _c.label = 21;
                    case 21:
                        if (!(i < logs.length)) return [3 /*break*/, 25];
                        log = logs[i];
                        if (log.blockHash != null) {
                            return [3 /*break*/, 24];
                        }
                        if (!(blocks[log.blockNumber] == null)) return [3 /*break*/, 23];
                        return [4 /*yield*/, this.getBlock(log.blockNumber)];
                    case 22:
                        block = _c.sent();
                        if (block) {
                            blocks[log.blockNumber] = block.hash;
                        }
                        _c.label = 23;
                    case 23:
                        log.blockHash = blocks[log.blockNumber];
                        _c.label = 24;
                    case 24:
                        i++;
                        return [3 /*break*/, 21];
                    case 25: return [2 /*return*/, logs];
                    case 26:
                        if (this.network.name !== "homestead") {
                            return [2 /*return*/, 0.0];
                        }
                        _b = parseFloat;
                        return [4 /*yield*/, this.fetch("stats", { action: "ethprice" })];
                    case 27: return [2 /*return*/, _b.apply(void 0, [(_c.sent()).ethusd])];
                    case 28: return [3 /*break*/, 29];
                    case 29: return [2 /*return*/, _super.prototype.perform.call(this, method, params)];
                }
            });
        });
    };
    // Note: The `page` page parameter only allows pagination within the
    //       10,000 window available without a page and offset parameter
    //       Error: Result window is too large, PageNo x Offset size must
    //              be less than or equal to 10000
    EtherscanProvider.prototype.getHistory = function (addressOrName, startBlock, endBlock) {
        return __awaiter(this, void 0, void 0, function () {
            var params, result;
            var _a;
            var _this = this;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        _a = {
                            action: "txlist"
                        };
                        return [4 /*yield*/, this.resolveName(addressOrName)];
                    case 1:
                        params = (_a.address = (_b.sent()),
                            _a.startblock = ((startBlock == null) ? 0 : startBlock),
                            _a.endblock = ((endBlock == null) ? 99999999 : endBlock),
                            _a.sort = "asc",
                            _a);
                        return [4 /*yield*/, this.fetch("account", params)];
                    case 2:
                        result = _b.sent();
                        return [2 /*return*/, result.map(function (tx) {
                                ["contractAddress", "to"].forEach(function (key) {
                                    if (tx[key] == "") {
                                        delete tx[key];
                                    }
                                });
                                if (tx.creates == null && tx.contractAddress != null) {
                                    tx.creates = tx.contractAddress;
                                }
                                var item = _this.formatter.transactionResponse(tx);
                                if (tx.timeStamp) {
                                    item.timestamp = parseInt(tx.timeStamp);
                                }
                                return item;
                            })];
                }
            });
        });
    };
    EtherscanProvider.prototype.isCommunityResource = function () {
        return (this.apiKey == null);
    };
    return EtherscanProvider;
}(base_provider_1.BaseProvider));
exports.EtherscanProvider = EtherscanProvider;
//# sourceMappingURL=etherscan-provider.js.map