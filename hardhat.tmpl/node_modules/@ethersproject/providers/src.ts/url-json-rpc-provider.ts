
"use strict";

import { Network, Networkish } from "@ethersproject/networks";
import { defineReadOnly, getStatic } from "@ethersproject/properties";
import { ConnectionInfo } from "@ethersproject/web";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

import { CommunityResourcable } from "./formatter";
import { JsonRpcProvider, JsonRpcSigner } from "./json-rpc-provider";

type getUrlFunc = (network: Network, apiKey: string) => string | ConnectionInfo;

// A StaticJsonRpcProvider is useful when you *know* for certain that
// the backend will never change, as it never calls eth_chainId to
// verify its backend. However, if the backend does change, the effects
// are undefined and may include:
// - inconsistent results
// - locking up the UI
// - block skew warnings
// - wrong results
// If the network is not explicit (i.e. auto-detection is expected), the
// node MUST be running and available to respond to requests BEFORE this
// is instantiated.
export class StaticJsonRpcProvider extends JsonRpcProvider {
    async detectNetwork(): Promise<Network> {
        let network = this.network;
        if (network == null) {
            network = await super.detectNetwork();

            if (!network) {
                logger.throwError("no network detected", Logger.errors.UNKNOWN_ERROR, { });
            }

            // If still not set, set it
            if (this._network == null) {
                // A static network does not support "any"
                defineReadOnly(this, "_network", network);

                this.emit("network", network, null);
            }
        }
        return network;
    }
}

export abstract class UrlJsonRpcProvider extends StaticJsonRpcProvider implements CommunityResourcable {
    readonly apiKey: any;

    constructor(network?: Networkish, apiKey?: any) {
        logger.checkAbstract(new.target, UrlJsonRpcProvider);

        // Normalize the Network and API Key
        network = getStatic<(network: Networkish) => Network>(new.target, "getNetwork")(network);
        apiKey = getStatic<(apiKey: string) => string>(new.target, "getApiKey")(apiKey);

        const connection = getStatic<getUrlFunc>(new.target, "getUrl")(network, apiKey);

        super(connection, network);

        if (typeof(apiKey) === "string") {
            defineReadOnly(this, "apiKey", apiKey);
        } else if (apiKey != null) {
            Object.keys(apiKey).forEach((key) => {
                defineReadOnly<any, any>(this, key, apiKey[key]);
            });
        }
    }

    _startPending(): void {
        logger.warn("WARNING: API provider does not support pending filters");
    }

    isCommunityResource(): boolean {
        return false;
    }

    getSigner(address?: string): JsonRpcSigner {
        return logger.throwError(
            "API provider does not support signing",
            Logger.errors.UNSUPPORTED_OPERATION,
            { operation: "getSigner" }
        );
    }

    listAccounts(): Promise<Array<string>> {
        return Promise.resolve([]);
    }

    // Return a defaultApiKey if null, otherwise validate the API key
    static getApiKey(apiKey: any): any {
        return apiKey;
    }

    // Returns the url or connection for the given network and API key. The
    // API key will have been sanitized by the getApiKey first, so any validation
    // or transformations can be done there.
    static getUrl(network: Network, apiKey: any): string | ConnectionInfo {
        return logger.throwError("not implemented; sub-classes must override getUrl", Logger.errors.NOT_IMPLEMENTED, {
            operation: "getUrl"
        });
    }
}
