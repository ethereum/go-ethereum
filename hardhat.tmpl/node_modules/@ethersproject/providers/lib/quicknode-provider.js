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
exports.QuickNodeProvider = void 0;
var url_json_rpc_provider_1 = require("./url-json-rpc-provider");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
// Special API key provided by Quicknode for ethers.js
var defaultApiKey = "919b412a057b5e9c9b6dce193c5a60242d6efadb";
var QuickNodeProvider = /** @class */ (function (_super) {
    __extends(QuickNodeProvider, _super);
    function QuickNodeProvider() {
        return _super !== null && _super.apply(this, arguments) || this;
    }
    QuickNodeProvider.getApiKey = function (apiKey) {
        if (apiKey && typeof (apiKey) !== "string") {
            logger.throwArgumentError("invalid apiKey", "apiKey", apiKey);
        }
        return apiKey || defaultApiKey;
    };
    QuickNodeProvider.getUrl = function (network, apiKey) {
        var host = null;
        switch (network.name) {
            case "homestead":
                host = "ethers.quiknode.pro";
                break;
            case "goerli":
                host = "ethers.ethereum-goerli.quiknode.pro";
                break;
            case "sepolia":
                host = "ethers.ethereum-sepolia.quiknode.pro";
                break;
            case "holesky":
                host = "ethers.ethereum-holesky.quiknode.pro";
                break;
            case "arbitrum":
                host = "ethers.arbitrum-mainnet.quiknode.pro";
                break;
            case "arbitrum-goerli":
                host = "ethers.arbitrum-goerli.quiknode.pro";
                break;
            case "arbitrum-sepolia":
                host = "ethers.arbitrum-sepolia.quiknode.pro";
                break;
            case "base":
                host = "ethers.base-mainnet.quiknode.pro";
                break;
            case "base-goerli":
                host = "ethers.base-goerli.quiknode.pro";
                break;
            case "base-spolia":
                host = "ethers.base-sepolia.quiknode.pro";
                break;
            case "bnb":
                host = "ethers.bsc.quiknode.pro";
                break;
            case "bnbt":
                host = "ethers.bsc-testnet.quiknode.pro";
                break;
            case "matic":
                host = "ethers.matic.quiknode.pro";
                break;
            case "maticmum":
                host = "ethers.matic-testnet.quiknode.pro";
                break;
            case "optimism":
                host = "ethers.optimism.quiknode.pro";
                break;
            case "optimism-goerli":
                host = "ethers.optimism-goerli.quiknode.pro";
                break;
            case "optimism-sepolia":
                host = "ethers.optimism-sepolia.quiknode.pro";
                break;
            case "xdai":
                host = "ethers.xdai.quiknode.pro";
                break;
            default:
                logger.throwArgumentError("unsupported network", "network", arguments[0]);
        }
        return ("https:/" + "/" + host + "/" + apiKey);
    };
    return QuickNodeProvider;
}(url_json_rpc_provider_1.UrlJsonRpcProvider));
exports.QuickNodeProvider = QuickNodeProvider;
//# sourceMappingURL=quicknode-provider.js.map