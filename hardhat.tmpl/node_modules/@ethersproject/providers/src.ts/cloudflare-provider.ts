"use strict";

import { Network } from "@ethersproject/networks";
import { UrlJsonRpcProvider } from "./url-json-rpc-provider";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

export class CloudflareProvider extends UrlJsonRpcProvider {

    static getApiKey(apiKey: any): any {
        if (apiKey != null) {
            logger.throwArgumentError("apiKey not supported for cloudflare", "apiKey", apiKey);
        }
        return null;
    }

    static getUrl(network: Network, apiKey?: any): string {
        let host = null;
        switch (network.name) {
            case "homestead":
                host = "https://cloudflare-eth.com/";
                break;
            default:
               logger.throwArgumentError("unsupported network", "network", arguments[0]);
        }

        return host;
    }

    async perform(method: string, params: any): Promise<any> {
        // The Cloudflare provider does not support eth_blockNumber,
        // so we get the latest block and pull it from that
        if (method === "getBlockNumber") {
            const block = await super.perform("getBlock", { blockTag: "latest" });
            return block.number;
        }

        return super.perform(method, params);
    }
}
