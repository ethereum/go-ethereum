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
exports.AnkrProvider = void 0;
var formatter_1 = require("./formatter");
var url_json_rpc_provider_1 = require("./url-json-rpc-provider");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var defaultApiKey = "9f7d929b018cdffb338517efa06f58359e86ff1ffd350bc889738523659e7972";
function getHost(name) {
    switch (name) {
        case "homestead":
            return "rpc.ankr.com/eth/";
        case "ropsten":
            return "rpc.ankr.com/eth_ropsten/";
        case "rinkeby":
            return "rpc.ankr.com/eth_rinkeby/";
        case "goerli":
            return "rpc.ankr.com/eth_goerli/";
        case "matic":
            return "rpc.ankr.com/polygon/";
        case "arbitrum":
            return "rpc.ankr.com/arbitrum/";
    }
    return logger.throwArgumentError("unsupported network", "name", name);
}
var AnkrProvider = /** @class */ (function (_super) {
    __extends(AnkrProvider, _super);
    function AnkrProvider() {
        return _super !== null && _super.apply(this, arguments) || this;
    }
    AnkrProvider.prototype.isCommunityResource = function () {
        return (this.apiKey === defaultApiKey);
    };
    AnkrProvider.getApiKey = function (apiKey) {
        if (apiKey == null) {
            return defaultApiKey;
        }
        return apiKey;
    };
    AnkrProvider.getUrl = function (network, apiKey) {
        if (apiKey == null) {
            apiKey = defaultApiKey;
        }
        var connection = {
            allowGzip: true,
            url: ("https:/\/" + getHost(network.name) + apiKey),
            throttleCallback: function (attempt, url) {
                if (apiKey.apiKey === defaultApiKey) {
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
    return AnkrProvider;
}(url_json_rpc_provider_1.UrlJsonRpcProvider));
exports.AnkrProvider = AnkrProvider;
//# sourceMappingURL=ankr-provider.js.map