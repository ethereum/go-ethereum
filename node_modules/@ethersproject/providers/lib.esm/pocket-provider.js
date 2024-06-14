"use strict";
import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);
import { UrlJsonRpcProvider } from "./url-json-rpc-provider";
const defaultApplicationId = "62e1ad51b37b8e00394bda3b";
export class PocketProvider extends UrlJsonRpcProvider {
    static getApiKey(apiKey) {
        const apiKeyObj = {
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
    }
    static getUrl(network, apiKey) {
        let host = null;
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
                logger.throwError("unsupported network", Logger.errors.INVALID_ARGUMENT, {
                    argument: "network",
                    value: network
                });
        }
        const url = `https:/\/${host}/v1/lb/${apiKey.applicationId}`;
        const connection = { headers: {}, url };
        if (apiKey.applicationSecretKey != null) {
            connection.user = "";
            connection.password = apiKey.applicationSecretKey;
        }
        return connection;
    }
    isCommunityResource() {
        return (this.applicationId === defaultApplicationId);
    }
}
//# sourceMappingURL=pocket-provider.js.map