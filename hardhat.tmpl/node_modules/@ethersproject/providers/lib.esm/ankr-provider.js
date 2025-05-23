import { showThrottleMessage } from "./formatter";
import { UrlJsonRpcProvider } from "./url-json-rpc-provider";
import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);
const defaultApiKey = "9f7d929b018cdffb338517efa06f58359e86ff1ffd350bc889738523659e7972";
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
        case "sepolia":
            return "rpc.ankr.com/eth_sepolia/";
        case "matic":
            return "rpc.ankr.com/polygon/";
        case "maticmum":
            return "rpc.ankr.com/polygon_mumbai/";
        case "optimism":
            return "rpc.ankr.com/optimism/";
        case "optimism-goerli":
            return "rpc.ankr.com/optimism_testnet/";
        case "optimism-sepolia":
            return "rpc.ankr.com/optimism_sepolia/";
        case "arbitrum":
            return "rpc.ankr.com/arbitrum/";
    }
    return logger.throwArgumentError("unsupported network", "name", name);
}
export class AnkrProvider extends UrlJsonRpcProvider {
    isCommunityResource() {
        return (this.apiKey === defaultApiKey);
    }
    static getApiKey(apiKey) {
        if (apiKey == null) {
            return defaultApiKey;
        }
        return apiKey;
    }
    static getUrl(network, apiKey) {
        if (apiKey == null) {
            apiKey = defaultApiKey;
        }
        const connection = {
            allowGzip: true,
            url: ("https:/\/" + getHost(network.name) + apiKey),
            throttleCallback: (attempt, url) => {
                if (apiKey.apiKey === defaultApiKey) {
                    showThrottleMessage();
                }
                return Promise.resolve(true);
            }
        };
        if (apiKey.projectSecret != null) {
            connection.user = "";
            connection.password = apiKey.projectSecret;
        }
        return connection;
    }
}
//# sourceMappingURL=ankr-provider.js.map