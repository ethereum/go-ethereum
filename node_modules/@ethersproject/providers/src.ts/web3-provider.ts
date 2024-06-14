"use strict";

import { Networkish } from "@ethersproject/networks";
import { deepCopy, defineReadOnly } from "@ethersproject/properties";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

import { JsonRpcProvider } from "./json-rpc-provider";

// Exported Types
export type ExternalProvider = {
    isMetaMask?: boolean;
    isStatus?: boolean;
    host?: string;
    path?: string;
    sendAsync?: (request: { method: string, params?: Array<any> }, callback: (error: any, response: any) => void) => void
    send?: (request: { method: string, params?: Array<any> }, callback: (error: any, response: any) => void) => void
    request?: (request: { method: string, params?: Array<any> }) => Promise<any>
}

let _nextId = 1;

export type JsonRpcFetchFunc = (method: string, params?: Array<any>) => Promise<any>;

type Web3LegacySend = (request: any, callback: (error: Error, response: any) => void) => void;

function buildWeb3LegacyFetcher(provider: ExternalProvider, sendFunc: Web3LegacySend) : JsonRpcFetchFunc {
    const fetcher = "Web3LegacyFetcher";

    return function(method: string, params: Array<any>): Promise<any> {
        const request = {
            method: method,
            params: params,
            id: (_nextId++),
            jsonrpc: "2.0"
        };

        return new Promise((resolve, reject) => {
            this.emit("debug", {
                action: "request",
                fetcher,
                request: deepCopy(request),
                provider: this
            });

            sendFunc(request, (error, response) => {

                if (error) {
                    this.emit("debug", {
                        action: "response",
                        fetcher,
                        error,
                        request,
                        provider: this
                    });

                    return reject(error);
                }

                this.emit("debug", {
                    action: "response",
                    fetcher,
                    request,
                    response,
                    provider: this
                });

                if (response.error) {
                    const error = new Error(response.error.message);
                    (<any>error).code = response.error.code;
                    (<any>error).data = response.error.data;
                    return reject(error);
                }

                resolve(response.result);
            });
        });
    }
}

function buildEip1193Fetcher(provider: ExternalProvider): JsonRpcFetchFunc {
    return function(method: string, params: Array<any>): Promise<any> {
        if (params == null) { params = [ ]; }

        const request = { method, params };

        this.emit("debug", {
            action: "request",
            fetcher: "Eip1193Fetcher",
            request: deepCopy(request),
            provider: this
        });

        return provider.request(request).then((response) => {
            this.emit("debug", {
                action: "response",
                fetcher: "Eip1193Fetcher",
                request,
                response,
                provider: this
            });

            return response;

        }, (error) => {
            this.emit("debug", {
                action: "response",
                fetcher: "Eip1193Fetcher",
                request,
                error,
                provider: this
            });

            throw error;
        });
    }
}

export class Web3Provider extends JsonRpcProvider {
    readonly provider: ExternalProvider;
    readonly jsonRpcFetchFunc: JsonRpcFetchFunc;

    constructor(provider: ExternalProvider | JsonRpcFetchFunc, network?: Networkish) {
        if (provider == null) {
            logger.throwArgumentError("missing provider", "provider", provider);
        }

        let path: string = null;
        let jsonRpcFetchFunc: JsonRpcFetchFunc = null;
        let subprovider: ExternalProvider = null;

        if (typeof(provider) === "function") {
            path = "unknown:";
            jsonRpcFetchFunc = provider;

        } else {
            path = provider.host || provider.path || "";
            if (!path && provider.isMetaMask) {
                path = "metamask";
            }

            subprovider = provider;

            if (provider.request) {
                if (path === "") { path = "eip-1193:"; }
                jsonRpcFetchFunc = buildEip1193Fetcher(provider);
            } else if (provider.sendAsync) {
                jsonRpcFetchFunc = buildWeb3LegacyFetcher(provider, provider.sendAsync.bind(provider));
            } else if (provider.send) {
                jsonRpcFetchFunc = buildWeb3LegacyFetcher(provider, provider.send.bind(provider));
            } else {
                logger.throwArgumentError("unsupported provider", "provider", provider);
            }

            if (!path) { path = "unknown:"; }
        }

        super(path, network);

        defineReadOnly(this, "jsonRpcFetchFunc", jsonRpcFetchFunc);
        defineReadOnly(this, "provider", subprovider);
    }

    send(method: string, params: Array<any>): Promise<any> {
        return this.jsonRpcFetchFunc(method, params);
    }
}
