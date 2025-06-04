"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Formatter = exports.showThrottleMessage = exports.isCommunityResourcable = exports.isCommunityResource = exports.getNetwork = exports.getDefaultProvider = exports.JsonRpcSigner = exports.IpcProvider = exports.WebSocketProvider = exports.Web3Provider = exports.StaticJsonRpcProvider = exports.QuickNodeProvider = exports.PocketProvider = exports.NodesmithProvider = exports.JsonRpcBatchProvider = exports.JsonRpcProvider = exports.InfuraWebSocketProvider = exports.InfuraProvider = exports.EtherscanProvider = exports.CloudflareProvider = exports.AnkrProvider = exports.AlchemyWebSocketProvider = exports.AlchemyProvider = exports.FallbackProvider = exports.UrlJsonRpcProvider = exports.Resolver = exports.BaseProvider = exports.Provider = void 0;
var abstract_provider_1 = require("@ethersproject/abstract-provider");
Object.defineProperty(exports, "Provider", { enumerable: true, get: function () { return abstract_provider_1.Provider; } });
var networks_1 = require("@ethersproject/networks");
Object.defineProperty(exports, "getNetwork", { enumerable: true, get: function () { return networks_1.getNetwork; } });
var base_provider_1 = require("./base-provider");
Object.defineProperty(exports, "BaseProvider", { enumerable: true, get: function () { return base_provider_1.BaseProvider; } });
Object.defineProperty(exports, "Resolver", { enumerable: true, get: function () { return base_provider_1.Resolver; } });
var alchemy_provider_1 = require("./alchemy-provider");
Object.defineProperty(exports, "AlchemyProvider", { enumerable: true, get: function () { return alchemy_provider_1.AlchemyProvider; } });
Object.defineProperty(exports, "AlchemyWebSocketProvider", { enumerable: true, get: function () { return alchemy_provider_1.AlchemyWebSocketProvider; } });
var ankr_provider_1 = require("./ankr-provider");
Object.defineProperty(exports, "AnkrProvider", { enumerable: true, get: function () { return ankr_provider_1.AnkrProvider; } });
var cloudflare_provider_1 = require("./cloudflare-provider");
Object.defineProperty(exports, "CloudflareProvider", { enumerable: true, get: function () { return cloudflare_provider_1.CloudflareProvider; } });
var etherscan_provider_1 = require("./etherscan-provider");
Object.defineProperty(exports, "EtherscanProvider", { enumerable: true, get: function () { return etherscan_provider_1.EtherscanProvider; } });
var fallback_provider_1 = require("./fallback-provider");
Object.defineProperty(exports, "FallbackProvider", { enumerable: true, get: function () { return fallback_provider_1.FallbackProvider; } });
var ipc_provider_1 = require("./ipc-provider");
Object.defineProperty(exports, "IpcProvider", { enumerable: true, get: function () { return ipc_provider_1.IpcProvider; } });
var infura_provider_1 = require("./infura-provider");
Object.defineProperty(exports, "InfuraProvider", { enumerable: true, get: function () { return infura_provider_1.InfuraProvider; } });
Object.defineProperty(exports, "InfuraWebSocketProvider", { enumerable: true, get: function () { return infura_provider_1.InfuraWebSocketProvider; } });
var json_rpc_provider_1 = require("./json-rpc-provider");
Object.defineProperty(exports, "JsonRpcProvider", { enumerable: true, get: function () { return json_rpc_provider_1.JsonRpcProvider; } });
Object.defineProperty(exports, "JsonRpcSigner", { enumerable: true, get: function () { return json_rpc_provider_1.JsonRpcSigner; } });
var json_rpc_batch_provider_1 = require("./json-rpc-batch-provider");
Object.defineProperty(exports, "JsonRpcBatchProvider", { enumerable: true, get: function () { return json_rpc_batch_provider_1.JsonRpcBatchProvider; } });
var nodesmith_provider_1 = require("./nodesmith-provider");
Object.defineProperty(exports, "NodesmithProvider", { enumerable: true, get: function () { return nodesmith_provider_1.NodesmithProvider; } });
var pocket_provider_1 = require("./pocket-provider");
Object.defineProperty(exports, "PocketProvider", { enumerable: true, get: function () { return pocket_provider_1.PocketProvider; } });
var quicknode_provider_1 = require("./quicknode-provider");
Object.defineProperty(exports, "QuickNodeProvider", { enumerable: true, get: function () { return quicknode_provider_1.QuickNodeProvider; } });
var url_json_rpc_provider_1 = require("./url-json-rpc-provider");
Object.defineProperty(exports, "StaticJsonRpcProvider", { enumerable: true, get: function () { return url_json_rpc_provider_1.StaticJsonRpcProvider; } });
Object.defineProperty(exports, "UrlJsonRpcProvider", { enumerable: true, get: function () { return url_json_rpc_provider_1.UrlJsonRpcProvider; } });
var web3_provider_1 = require("./web3-provider");
Object.defineProperty(exports, "Web3Provider", { enumerable: true, get: function () { return web3_provider_1.Web3Provider; } });
var websocket_provider_1 = require("./websocket-provider");
Object.defineProperty(exports, "WebSocketProvider", { enumerable: true, get: function () { return websocket_provider_1.WebSocketProvider; } });
var formatter_1 = require("./formatter");
Object.defineProperty(exports, "Formatter", { enumerable: true, get: function () { return formatter_1.Formatter; } });
Object.defineProperty(exports, "isCommunityResourcable", { enumerable: true, get: function () { return formatter_1.isCommunityResourcable; } });
Object.defineProperty(exports, "isCommunityResource", { enumerable: true, get: function () { return formatter_1.isCommunityResource; } });
Object.defineProperty(exports, "showThrottleMessage", { enumerable: true, get: function () { return formatter_1.showThrottleMessage; } });
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
////////////////////////
// Helper Functions
function getDefaultProvider(network, options) {
    if (network == null) {
        network = "homestead";
    }
    // If passed a URL, figure out the right type of provider based on the scheme
    if (typeof (network) === "string") {
        // @TODO: Add support for IpcProvider; maybe if it ends in ".ipc"?
        // Handle http and ws (and their secure variants)
        var match = network.match(/^(ws|http)s?:/i);
        if (match) {
            switch (match[1].toLowerCase()) {
                case "http":
                case "https":
                    return new json_rpc_provider_1.JsonRpcProvider(network);
                case "ws":
                case "wss":
                    return new websocket_provider_1.WebSocketProvider(network);
                default:
                    logger.throwArgumentError("unsupported URL scheme", "network", network);
            }
        }
    }
    var n = (0, networks_1.getNetwork)(network);
    if (!n || !n._defaultProvider) {
        logger.throwError("unsupported getDefaultProvider network", logger_1.Logger.errors.NETWORK_ERROR, {
            operation: "getDefaultProvider",
            network: network
        });
    }
    return n._defaultProvider({
        FallbackProvider: fallback_provider_1.FallbackProvider,
        AlchemyProvider: alchemy_provider_1.AlchemyProvider,
        AnkrProvider: ankr_provider_1.AnkrProvider,
        CloudflareProvider: cloudflare_provider_1.CloudflareProvider,
        EtherscanProvider: etherscan_provider_1.EtherscanProvider,
        InfuraProvider: infura_provider_1.InfuraProvider,
        JsonRpcProvider: json_rpc_provider_1.JsonRpcProvider,
        NodesmithProvider: nodesmith_provider_1.NodesmithProvider,
        PocketProvider: pocket_provider_1.PocketProvider,
        QuickNodeProvider: quicknode_provider_1.QuickNodeProvider,
        Web3Provider: web3_provider_1.Web3Provider,
        IpcProvider: ipc_provider_1.IpcProvider,
    }, options);
}
exports.getDefaultProvider = getDefaultProvider;
//# sourceMappingURL=index.js.map