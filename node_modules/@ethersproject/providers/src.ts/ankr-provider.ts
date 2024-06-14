
import { Network } from "@ethersproject/networks";

import { showThrottleMessage } from "./formatter";
import { UrlJsonRpcProvider } from "./url-json-rpc-provider";

import type { ConnectionInfo } from "@ethersproject/web";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);


const defaultApiKey = "9f7d929b018cdffb338517efa06f58359e86ff1ffd350bc889738523659e7972";

function getHost(name: string): string {
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

export class AnkrProvider extends UrlJsonRpcProvider {
    readonly apiKey: string;

    isCommunityResource(): boolean {
        return (this.apiKey === defaultApiKey);
    }

    static getApiKey(apiKey: any): any {
        if (apiKey == null) { return defaultApiKey; }
        return apiKey;
    }

    static getUrl(network: Network, apiKey: any): ConnectionInfo {
        if (apiKey == null) { apiKey = defaultApiKey; }
        const connection: ConnectionInfo = {
            allowGzip: true,
            url: ("https:/\/" + getHost(network.name) + apiKey),
            throttleCallback: (attempt: number, url: string) => {
                if (apiKey.apiKey === defaultApiKey) {
                    showThrottleMessage();
                }
                return Promise.resolve(true);
            }
        };

        if (apiKey.projectSecret != null) {
            connection.user = "";
            connection.password = apiKey.projectSecret
        }

        return connection;
    }
}
