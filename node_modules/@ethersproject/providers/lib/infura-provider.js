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
exports.InfuraProvider = exports.InfuraWebSocketProvider = void 0;
var properties_1 = require("@ethersproject/properties");
var websocket_provider_1 = require("./websocket-provider");
var formatter_1 = require("./formatter");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var url_json_rpc_provider_1 = require("./url-json-rpc-provider");
var defaultProjectId = "84842078b09946638c03157f83405213";
var InfuraWebSocketProvider = /** @class */ (function (_super) {
    __extends(InfuraWebSocketProvider, _super);
    function InfuraWebSocketProvider(network, apiKey) {
        var _this = this;
        var provider = new InfuraProvider(network, apiKey);
        var connection = provider.connection;
        if (connection.password) {
            logger.throwError("INFURA WebSocket project secrets unsupported", logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
                operation: "InfuraProvider.getWebSocketProvider()"
            });
        }
        var url = connection.url.replace(/^http/i, "ws").replace("/v3/", "/ws/v3/");
        _this = _super.call(this, url, network) || this;
        (0, properties_1.defineReadOnly)(_this, "apiKey", provider.projectId);
        (0, properties_1.defineReadOnly)(_this, "projectId", provider.projectId);
        (0, properties_1.defineReadOnly)(_this, "projectSecret", provider.projectSecret);
        return _this;
    }
    InfuraWebSocketProvider.prototype.isCommunityResource = function () {
        return (this.projectId === defaultProjectId);
    };
    return InfuraWebSocketProvider;
}(websocket_provider_1.WebSocketProvider));
exports.InfuraWebSocketProvider = InfuraWebSocketProvider;
var InfuraProvider = /** @class */ (function (_super) {
    __extends(InfuraProvider, _super);
    function InfuraProvider() {
        return _super !== null && _super.apply(this, arguments) || this;
    }
    InfuraProvider.getWebSocketProvider = function (network, apiKey) {
        return new InfuraWebSocketProvider(network, apiKey);
    };
    InfuraProvider.getApiKey = function (apiKey) {
        var apiKeyObj = {
            apiKey: defaultProjectId,
            projectId: defaultProjectId,
            projectSecret: null
        };
        if (apiKey == null) {
            return apiKeyObj;
        }
        if (typeof (apiKey) === "string") {
            apiKeyObj.projectId = apiKey;
        }
        else if (apiKey.projectSecret != null) {
            logger.assertArgument((typeof (apiKey.projectId) === "string"), "projectSecret requires a projectId", "projectId", apiKey.projectId);
            logger.assertArgument((typeof (apiKey.projectSecret) === "string"), "invalid projectSecret", "projectSecret", "[REDACTED]");
            apiKeyObj.projectId = apiKey.projectId;
            apiKeyObj.projectSecret = apiKey.projectSecret;
        }
        else if (apiKey.projectId) {
            apiKeyObj.projectId = apiKey.projectId;
        }
        apiKeyObj.apiKey = apiKeyObj.projectId;
        return apiKeyObj;
    };
    InfuraProvider.getUrl = function (network, apiKey) {
        var host = null;
        switch (network ? network.name : "unknown") {
            case "homestead":
                host = "mainnet.infura.io";
                break;
            case "goerli":
                host = "goerli.infura.io";
                break;
            case "sepolia":
                host = "sepolia.infura.io";
                break;
            case "matic":
                host = "polygon-mainnet.infura.io";
                break;
            case "maticmum":
                host = "polygon-mumbai.infura.io";
                break;
            case "optimism":
                host = "optimism-mainnet.infura.io";
                break;
            case "optimism-goerli":
                host = "optimism-goerli.infura.io";
                break;
            case "arbitrum":
                host = "arbitrum-mainnet.infura.io";
                break;
            case "arbitrum-goerli":
                host = "arbitrum-goerli.infura.io";
                break;
            default:
                logger.throwError("unsupported network", logger_1.Logger.errors.INVALID_ARGUMENT, {
                    argument: "network",
                    value: network
                });
        }
        var connection = {
            allowGzip: true,
            url: ("https:/" + "/" + host + "/v3/" + apiKey.projectId),
            throttleCallback: function (attempt, url) {
                if (apiKey.projectId === defaultProjectId) {
                    (0, formatter_1.showThrottleMessage)();
                }
                return Promise.resolve(true);
            }
        };
        if (apiKey.projectSecret != null) {
            connection.user = "";
            connection.password = apiKey.projectSecret;
        }
        return connection;
    };
    InfuraProvider.prototype.isCommunityResource = function () {
        return (this.projectId === defaultProjectId);
    };
    return InfuraProvider;
}(url_json_rpc_provider_1.UrlJsonRpcProvider));
exports.InfuraProvider = InfuraProvider;
//# sourceMappingURL=infura-provider.js.map