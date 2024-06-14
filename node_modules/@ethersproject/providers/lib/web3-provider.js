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
Object.defineProperty(exports, "__esModule", { value: true });
exports.Web3Provider = void 0;
var properties_1 = require("@ethersproject/properties");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var json_rpc_provider_1 = require("./json-rpc-provider");
var _nextId = 1;
function buildWeb3LegacyFetcher(provider, sendFunc) {
    var fetcher = "Web3LegacyFetcher";
    return function (method, params) {
        var _this = this;
        var request = {
            method: method,
            params: params,
            id: (_nextId++),
            jsonrpc: "2.0"
        };
        return new Promise(function (resolve, reject) {
            _this.emit("debug", {
                action: "request",
                fetcher: fetcher,
                request: (0, properties_1.deepCopy)(request),
                provider: _this
            });
            sendFunc(request, function (error, response) {
                if (error) {
                    _this.emit("debug", {
                        action: "response",
                        fetcher: fetcher,
                        error: error,
                        request: request,
                        provider: _this
                    });
                    return reject(error);
                }
                _this.emit("debug", {
                    action: "response",
                    fetcher: fetcher,
                    request: request,
                    response: response,
                    provider: _this
                });
                if (response.error) {
                    var error_1 = new Error(response.error.message);
                    error_1.code = response.error.code;
                    error_1.data = response.error.data;
                    return reject(error_1);
                }
                resolve(response.result);
            });
        });
    };
}
function buildEip1193Fetcher(provider) {
    return function (method, params) {
        var _this = this;
        if (params == null) {
            params = [];
        }
        var request = { method: method, params: params };
        this.emit("debug", {
            action: "request",
            fetcher: "Eip1193Fetcher",
            request: (0, properties_1.deepCopy)(request),
            provider: this
        });
        return provider.request(request).then(function (response) {
            _this.emit("debug", {
                action: "response",
                fetcher: "Eip1193Fetcher",
                request: request,
                response: response,
                provider: _this
            });
            return response;
        }, function (error) {
            _this.emit("debug", {
                action: "response",
                fetcher: "Eip1193Fetcher",
                request: request,
                error: error,
                provider: _this
            });
            throw error;
        });
    };
}
var Web3Provider = /** @class */ (function (_super) {
    __extends(Web3Provider, _super);
    function Web3Provider(provider, network) {
        var _this = this;
        if (provider == null) {
            logger.throwArgumentError("missing provider", "provider", provider);
        }
        var path = null;
        var jsonRpcFetchFunc = null;
        var subprovider = null;
        if (typeof (provider) === "function") {
            path = "unknown:";
            jsonRpcFetchFunc = provider;
        }
        else {
            path = provider.host || provider.path || "";
            if (!path && provider.isMetaMask) {
                path = "metamask";
            }
            subprovider = provider;
            if (provider.request) {
                if (path === "") {
                    path = "eip-1193:";
                }
                jsonRpcFetchFunc = buildEip1193Fetcher(provider);
            }
            else if (provider.sendAsync) {
                jsonRpcFetchFunc = buildWeb3LegacyFetcher(provider, provider.sendAsync.bind(provider));
            }
            else if (provider.send) {
                jsonRpcFetchFunc = buildWeb3LegacyFetcher(provider, provider.send.bind(provider));
            }
            else {
                logger.throwArgumentError("unsupported provider", "provider", provider);
            }
            if (!path) {
                path = "unknown:";
            }
        }
        _this = _super.call(this, path, network) || this;
        (0, properties_1.defineReadOnly)(_this, "jsonRpcFetchFunc", jsonRpcFetchFunc);
        (0, properties_1.defineReadOnly)(_this, "provider", subprovider);
        return _this;
    }
    Web3Provider.prototype.send = function (method, params) {
        return this.jsonRpcFetchFunc(method, params);
    };
    return Web3Provider;
}(json_rpc_provider_1.JsonRpcProvider));
exports.Web3Provider = Web3Provider;
//# sourceMappingURL=web3-provider.js.map