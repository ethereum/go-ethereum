"use strict";
import { defineReadOnly } from "@ethersproject/properties";
import { showThrottleMessage } from "./formatter";
import { WebSocketProvider } from "./websocket-provider";
import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);
import { UrlJsonRpcProvider } from "./url-json-rpc-provider";
// This key was provided to ethers.js by Alchemy to be used by the
// default provider, but it is recommended that for your own
// production environments, that you acquire your own API key at:
//   https://dashboard.alchemyapi.io
const defaultApiKey = "_gg7wSSi0KMBsdKnGVfHDueq6xMB9EkC";
export class AlchemyWebSocketProvider extends WebSocketProvider {
    constructor(network, apiKey) {
        const provider = new AlchemyProvider(network, apiKey);
        const url = provider.connection.url.replace(/^http/i, "ws")
            .replace(".alchemyapi.", ".ws.alchemyapi.");
        super(url, provider.network);
        defineReadOnly(this, "apiKey", provider.apiKey);
    }
    isCommunityResource() {
        return (this.apiKey === defaultApiKey);
    }
}
export class AlchemyProvider extends UrlJsonRpcProvider {
    static getWebSocketProvider(network, apiKey) {
        return new AlchemyWebSocketProvider(network, apiKey);
    }
    static getApiKey(apiKey) {
        if (apiKey == null) {
            return defaultApiKey;
        }
        if (apiKey && typeof (apiKey) !== "string") {
            logger.throwArgumentError("invalid apiKey", "apiKey", apiKey);
        }
        return apiKey;
    }
    static getUrl(network, apiKey) {
        let host = null;
        switch (network.name) {
            case "homestead":
                host = "eth-mainnet.alchemyapi.io/v2/";
                break;
            case "goerli":
                host = "eth-goerli.g.alchemy.com/v2/";
                break;
            case "matic":
                host = "polygon-mainnet.g.alchemy.com/v2/";
                break;
            case "maticmum":
                host = "polygon-mumbai.g.alchemy.com/v2/";
                break;
            case "arbitrum":
                host = "arb-mainnet.g.alchemy.com/v2/";
                break;
            case "arbitrum-goerli":
                host = "arb-goerli.g.alchemy.com/v2/";
                break;
            case "optimism":
                host = "opt-mainnet.g.alchemy.com/v2/";
                break;
            case "optimism-goerli":
                host = "opt-goerli.g.alchemy.com/v2/";
                break;
            default:
                logger.throwArgumentError("unsupported network", "network", arguments[0]);
        }
        return {
            allowGzip: true,
            url: ("https:/" + "/" + host + apiKey),
            throttleCallback: (attempt, url) => {
                if (apiKey === defaultApiKey) {
                    showThrottleMessage();
                }
                return Promise.resolve(true);
            }
        };
    }
    isCommunityResource() {
        return (this.apiKey === defaultApiKey);
    }
}
//# sourceMappingURL=alchemy-provider.js.map