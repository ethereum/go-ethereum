"use strict";
import { UrlJsonRpcProvider } from "./url-json-rpc-provider";
import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);
// Special API key provided by Quicknode for ethers.js
const defaultApiKey = "919b412a057b5e9c9b6dce193c5a60242d6efadb";
export class QuickNodeProvider extends UrlJsonRpcProvider {
    static getApiKey(apiKey) {
        if (apiKey && typeof (apiKey) !== "string") {
            logger.throwArgumentError("invalid apiKey", "apiKey", apiKey);
        }
        return apiKey || defaultApiKey;
    }
    static getUrl(network, apiKey) {
        let host = null;
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
    }
}
//# sourceMappingURL=quicknode-provider.js.map