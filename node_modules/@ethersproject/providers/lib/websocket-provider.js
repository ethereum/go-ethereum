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
exports.WebSocketProvider = void 0;
var bignumber_1 = require("@ethersproject/bignumber");
var properties_1 = require("@ethersproject/properties");
var json_rpc_provider_1 = require("./json-rpc-provider");
var ws_1 = require("./ws");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
/**
 *  Notes:
 *
 *  This provider differs a bit from the polling providers. One main
 *  difference is how it handles consistency. The polling providers
 *  will stall responses to ensure a consistent state, while this
 *  WebSocket provider assumes the connected backend will manage this.
 *
 *  For example, if a polling provider emits an event which indicates
 *  the event occurred in blockhash XXX, a call to fetch that block by
 *  its hash XXX, if not present will retry until it is present. This
 *  can occur when querying a pool of nodes that are mildly out of sync
 *  with each other.
 */
var NextId = 1;
// For more info about the Real-time Event API see:
//   https://geth.ethereum.org/docs/rpc/pubsub
var WebSocketProvider = /** @class */ (function (_super) {
    __extends(WebSocketProvider, _super);
    function WebSocketProvider(url, network) {
        var _this = this;
        // This will be added in the future; please open an issue to expedite
        if (network === "any") {
            logger.throwError("WebSocketProvider does not support 'any' network yet", logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
                operation: "network:any"
            });
        }
        if (typeof (url) === "string") {
            _this = _super.call(this, url, network) || this;
        }
        else {
            _this = _super.call(this, "_websocket", network) || this;
        }
        _this._pollingInterval = -1;
        _this._wsReady = false;
        if (typeof (url) === "string") {
            (0, properties_1.defineReadOnly)(_this, "_websocket", new ws_1.WebSocket(_this.connection.url));
        }
        else {
            (0, properties_1.defineReadOnly)(_this, "_websocket", url);
        }
        (0, properties_1.defineReadOnly)(_this, "_requests", {});
        (0, properties_1.defineReadOnly)(_this, "_subs", {});
        (0, properties_1.defineReadOnly)(_this, "_subIds", {});
        (0, properties_1.defineReadOnly)(_this, "_detectNetwork", _super.prototype.detectNetwork.call(_this));
        // Stall sending requests until the socket is open...
        _this.websocket.onopen = function () {
            _this._wsReady = true;
            Object.keys(_this._requests).forEach(function (id) {
                _this.websocket.send(_this._requests[id].payload);
            });
        };
        _this.websocket.onmessage = function (messageEvent) {
            var data = messageEvent.data;
            var result = JSON.parse(data);
            if (result.id != null) {
                var id = String(result.id);
                var request = _this._requests[id];
                delete _this._requests[id];
                if (result.result !== undefined) {
                    request.callback(null, result.result);
                    _this.emit("debug", {
                        action: "response",
                        request: JSON.parse(request.payload),
                        response: result.result,
                        provider: _this
                    });
                }
                else {
                    var error = null;
                    if (result.error) {
                        error = new Error(result.error.message || "unknown error");
                        (0, properties_1.defineReadOnly)(error, "code", result.error.code || null);
                        (0, properties_1.defineReadOnly)(error, "response", data);
                    }
                    else {
                        error = new Error("unknown error");
                    }
                    request.callback(error, undefined);
                    _this.emit("debug", {
                        action: "response",
                        error: error,
                        request: JSON.parse(request.payload),
                        provider: _this
                    });
                }
            }
            else if (result.method === "eth_subscription") {
                // Subscription...
                var sub = _this._subs[result.params.subscription];
                if (sub) {
                    //this.emit.apply(this,                  );
                    sub.processFunc(result.params.result);
                }
            }
            else {
                console.warn("this should not happen");
            }
        };
        // This Provider does not actually poll, but we want to trigger
        // poll events for things that depend on them (like stalling for
        // block and transaction lookups)
        var fauxPoll = setInterval(function () {
            _this.emit("poll");
        }, 1000);
        if (fauxPoll.unref) {
            fauxPoll.unref();
        }
        return _this;
    }
    Object.defineProperty(WebSocketProvider.prototype, "websocket", {
        // Cannot narrow the type of _websocket, as that is not backwards compatible
        // so we add a getter and let the WebSocket be a public API.
        get: function () { return this._websocket; },
        enumerable: false,
        configurable: true
    });
    WebSocketProvider.prototype.detectNetwork = function () {
        return this._detectNetwork;
    };
    Object.defineProperty(WebSocketProvider.prototype, "pollingInterval", {
        get: function () {
            return 0;
        },
        set: function (value) {
            logger.throwError("cannot set polling interval on WebSocketProvider", logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
                operation: "setPollingInterval"
            });
        },
        enumerable: false,
        configurable: true
    });
    WebSocketProvider.prototype.resetEventsBlock = function (blockNumber) {
        logger.throwError("cannot reset events block on WebSocketProvider", logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
            operation: "resetEventBlock"
        });
    };
    WebSocketProvider.prototype.poll = function () {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                return [2 /*return*/, null];
            });
        });
    };
    Object.defineProperty(WebSocketProvider.prototype, "polling", {
        set: function (value) {
            if (!value) {
                return;
            }
            logger.throwError("cannot set polling on WebSocketProvider", logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
                operation: "setPolling"
            });
        },
        enumerable: false,
        configurable: true
    });
    WebSocketProvider.prototype.send = function (method, params) {
        var _this = this;
        var rid = NextId++;
        return new Promise(function (resolve, reject) {
            function callback(error, result) {
                if (error) {
                    return reject(error);
                }
                return resolve(result);
            }
            var payload = JSON.stringify({
                method: method,
                params: params,
                id: rid,
                jsonrpc: "2.0"
            });
            _this.emit("debug", {
                action: "request",
                request: JSON.parse(payload),
                provider: _this
            });
            _this._requests[String(rid)] = { callback: callback, payload: payload };
            if (_this._wsReady) {
                _this.websocket.send(payload);
            }
        });
    };
    WebSocketProvider.defaultUrl = function () {
        return "ws:/\/localhost:8546";
    };
    WebSocketProvider.prototype._subscribe = function (tag, param, processFunc) {
        return __awaiter(this, void 0, void 0, function () {
            var subIdPromise, subId;
            var _this = this;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        subIdPromise = this._subIds[tag];
                        if (subIdPromise == null) {
                            subIdPromise = Promise.all(param).then(function (param) {
                                return _this.send("eth_subscribe", param);
                            });
                            this._subIds[tag] = subIdPromise;
                        }
                        return [4 /*yield*/, subIdPromise];
                    case 1:
                        subId = _a.sent();
                        this._subs[subId] = { tag: tag, processFunc: processFunc };
                        return [2 /*return*/];
                }
            });
        });
    };
    WebSocketProvider.prototype._startEvent = function (event) {
        var _this = this;
        switch (event.type) {
            case "block":
                this._subscribe("block", ["newHeads"], function (result) {
                    var blockNumber = bignumber_1.BigNumber.from(result.number).toNumber();
                    _this._emitted.block = blockNumber;
                    _this.emit("block", blockNumber);
                });
                break;
            case "pending":
                this._subscribe("pending", ["newPendingTransactions"], function (result) {
                    _this.emit("pending", result);
                });
                break;
            case "filter":
                this._subscribe(event.tag, ["logs", this._getFilter(event.filter)], function (result) {
                    if (result.removed == null) {
                        result.removed = false;
                    }
                    _this.emit(event.filter, _this.formatter.filterLog(result));
                });
                break;
            case "tx": {
                var emitReceipt_1 = function (event) {
                    var hash = event.hash;
                    _this.getTransactionReceipt(hash).then(function (receipt) {
                        if (!receipt) {
                            return;
                        }
                        _this.emit(hash, receipt);
                    });
                };
                // In case it is already mined
                emitReceipt_1(event);
                // To keep things simple, we start up a single newHeads subscription
                // to keep an eye out for transactions we are watching for.
                // Starting a subscription for an event (i.e. "tx") that is already
                // running is (basically) a nop.
                this._subscribe("tx", ["newHeads"], function (result) {
                    _this._events.filter(function (e) { return (e.type === "tx"); }).forEach(emitReceipt_1);
                });
                break;
            }
            // Nothing is needed
            case "debug":
            case "poll":
            case "willPoll":
            case "didPoll":
            case "error":
                break;
            default:
                console.log("unhandled:", event);
                break;
        }
    };
    WebSocketProvider.prototype._stopEvent = function (event) {
        var _this = this;
        var tag = event.tag;
        if (event.type === "tx") {
            // There are remaining transaction event listeners
            if (this._events.filter(function (e) { return (e.type === "tx"); }).length) {
                return;
            }
            tag = "tx";
        }
        else if (this.listenerCount(event.event)) {
            // There are remaining event listeners
            return;
        }
        var subId = this._subIds[tag];
        if (!subId) {
            return;
        }
        delete this._subIds[tag];
        subId.then(function (subId) {
            if (!_this._subs[subId]) {
                return;
            }
            delete _this._subs[subId];
            _this.send("eth_unsubscribe", [subId]);
        });
    };
    WebSocketProvider.prototype.destroy = function () {
        return __awaiter(this, void 0, void 0, function () {
            var _this = this;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!(this.websocket.readyState === ws_1.WebSocket.CONNECTING)) return [3 /*break*/, 2];
                        return [4 /*yield*/, (new Promise(function (resolve) {
                                _this.websocket.onopen = function () {
                                    resolve(true);
                                };
                                _this.websocket.onerror = function () {
                                    resolve(false);
                                };
                            }))];
                    case 1:
                        _a.sent();
                        _a.label = 2;
                    case 2:
                        // Hangup
                        // See: https://developer.mozilla.org/en-US/docs/Web/API/CloseEvent#Status_codes
                        this.websocket.close(1000);
                        return [2 /*return*/];
                }
            });
        });
    };
    return WebSocketProvider;
}(json_rpc_provider_1.JsonRpcProvider));
exports.WebSocketProvider = WebSocketProvider;
//# sourceMappingURL=websocket-provider.js.map