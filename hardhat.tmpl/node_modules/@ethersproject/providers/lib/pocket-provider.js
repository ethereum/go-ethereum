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
exports.PocketProvider = void 0;
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var url_json_rpc_provider_1 = require("./url-json-rpc-provider");
var defaultApplicationId = "62e1ad51b37b8e00394bda3b";
var PocketProvider = /** @class */ (function (_super) {
    __extends(PocketProvider, _super);
    function PocketProvider() {
        return _super !== null && _super.apply(this, arguments) || this;
    }
    PocketProvider.getApiKey = function (apiKey) {
        var apiKeyObj = {
            applicationId: null,
            loadBalancer: true,
            applicationSecretKey: null
        };
        // Parse applicationId and applicationSecretKey
        if (apiKey == null) {
            apiKeyObj.applicationId = defaultApplicationId;
        }
        else if (typeof (apiKey) === "string") {
            apiKeyObj.applicationId = apiKey;
        }
        else if (apiKey.applicationSecretKey != null) {
            apiKeyObj.applicationId = apiKey.applicationId;
            apiKeyObj.applicationSecretKey = apiKey.applicationSecretKey;
        }
        else if (apiKey.applicationId) {
            apiKeyObj.applicationId = apiKey.applicationId;
        }
        else {
            logger.throwArgumentError("unsupported PocketProvider apiKey", "apiKey", apiKey);
        }
        return apiKeyObj;
    };
    PocketProvider.getUrl = function (network, apiKey) {
        var host = null;
        switch (network ? network.name : "unknown") {
            case "goerli":
                host = "eth-goerli.gateway.pokt.network";
                break;
            case "homestead":
                host = "eth-mainnet.gateway.pokt.network";
                break;
            case "kovan":
                host = "poa-kovan.gateway.pokt.network";
                break;
            case "matic":
                host = "poly-mainnet.gateway.pokt.network";
                break;
            case "maticmum":
                host = "polygon-mumbai-rpc.gateway.pokt.network";
                break;
            case "rinkeby":
                host = "eth-rinkeby.gateway.pokt.network";
                break;
            case "ropsten":
                host = "eth-ropsten.gateway.pokt.network";
                break;
            default:
                logger.throwError("unsupported network", logger_1.Logger.errors.INVALID_ARGUMENT, {
                    argument: "network",
                    value: network
                });
        }
        var url = "https://" + host + "/v1/lb/" + apiKey.applicationId;
        var connection = { headers: {}, url: url };
        if (apiKey.applicationSecretKey != null) {
            connection.user = "";
            connection.password = apiKey.applicationSecretKey;
        }
        return connection;
    };
    PocketProvider.prototype.isCommunityResource = function () {
        return (this.applicationId === defaultApplicationId);
    };
    return PocketProvider;
}(url_json_rpc_provider_1.UrlJsonRpcProvider));
exports.PocketProvider = PocketProvider;
//# sourceMappingURL=pocket-provider.js.map