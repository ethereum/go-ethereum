"use strict";

import { BlockTag, TransactionRequest, TransactionResponse } from "@ethersproject/abstract-provider";
import { hexlify, hexValue, isHexString } from "@ethersproject/bytes";
import { Network, Networkish } from "@ethersproject/networks";
import { deepCopy, defineReadOnly } from "@ethersproject/properties";
import { accessListify } from "@ethersproject/transactions";
import { ConnectionInfo, fetchJson } from "@ethersproject/web";

import { showThrottleMessage } from "./formatter";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

import { BaseProvider } from "./base-provider";


// The transaction has already been sanitized by the calls in Provider
function getTransactionPostData(transaction: TransactionRequest): Record<string, string> {
    const result: Record<string, string> = { };
    for (let key in transaction) {
        if ((<any>transaction)[key] == null) { continue; }
        let value = (<any>transaction)[key];
        if (key === "type" && value === 0) { continue; }

        // Quantity-types require no leading zero, unless 0
        if ((<any>{ type: true, gasLimit: true, gasPrice: true, maxFeePerGs: true, maxPriorityFeePerGas: true, nonce: true, value: true })[key]) {
            value = hexValue(hexlify(value));
        } else if (key === "accessList") {
            value = "[" + accessListify(value).map((set) => {
                return `{address:"${ set.address }",storageKeys:["${ set.storageKeys.join('","') }"]}`;
            }).join(",") + "]";
        } else {
            value = hexlify(value);
        }
        result[key] = value;
    }
    return result;
}

function getResult(result: { status?: number, message?: string, result?: any }): any {
    // getLogs, getHistory have weird success responses
    if (result.status == 0 && (result.message === "No records found" || result.message === "No transactions found")) {
        return result.result;
    }

    if (result.status != 1 || typeof(result.message) !== "string" || !result.message.match(/^OK/)) {
        const error: any = new Error("invalid response");
        error.result = JSON.stringify(result);
        if ((result.result || "").toLowerCase().indexOf("rate limit") >= 0) {
            error.throttleRetry = true;
        }
        throw error;
    }

    return result.result;
}

function getJsonResult(result: { jsonrpc: string, result?: any, error?: { code?: number, data?: any, message?: string} } ): any {
    // This response indicates we are being throttled
    if (result && (<any>result).status == 0 && (<any>result).message == "NOTOK" && (result.result || "").toLowerCase().indexOf("rate limit") >= 0) {
        const error: any = new Error("throttled response");
        error.result = JSON.stringify(result);
        error.throttleRetry = true;
        throw error;
    }

    if (result.jsonrpc != "2.0") {
        // @TODO: not any
        const error: any = new Error("invalid response");
        error.result = JSON.stringify(result);
        throw error;
    }

    if (result.error) {
        // @TODO: not any
        const error: any = new Error(result.error.message || "unknown error");
        if (result.error.code) { error.code = result.error.code; }
        if (result.error.data) { error.data = result.error.data; }
        throw error;
    }

    return result.result;
}

// The blockTag was normalized as a string by the Provider pre-perform operations
function checkLogTag(blockTag: string): number | "latest" {
    if (blockTag === "pending") { throw new Error("pending not supported"); }
    if (blockTag === "latest") { return blockTag; }

    return parseInt(blockTag.substring(2), 16);
}


function checkError(method: string, error: any, transaction: any): any {
    // Undo the "convenience" some nodes are attempting to prevent backwards
    // incompatibility; maybe for v6 consider forwarding reverts as errors
    if (method === "call" && error.code === Logger.errors.SERVER_ERROR) {
        const e = error.error;

        // Etherscan keeps changing their string
        if (e && (e.message.match(/reverted/i) || e.message.match(/VM execution error/i))) {
            // Etherscan prefixes the data like "Reverted 0x1234"
            let data = e.data;
            if (data) { data = "0x" + data.replace(/^.*0x/i, ""); }

            if (isHexString(data)) { return data; }

            logger.throwError("missing revert data in call exception", Logger.errors.CALL_EXCEPTION, {
                error, data: "0x"
            });
        }
    }

    // Get the message from any nested error structure
    let message = error.message;
    if (error.code === Logger.errors.SERVER_ERROR) {
        if (error.error && typeof(error.error.message) === "string") {
            message = error.error.message;
        } else if (typeof(error.body) === "string") {
            message = error.body;
        } else if (typeof(error.responseText) === "string") {
            message = error.responseText;
        }
    }
    message = (message || "").toLowerCase();

    // "Insufficient funds. The account you tried to send transaction from does not have enough funds. Required 21464000000000 and got: 0"
    if (message.match(/insufficient funds/)) {
        logger.throwError("insufficient funds for intrinsic transaction cost", Logger.errors.INSUFFICIENT_FUNDS, {
           error, method, transaction
        });
    }

    // "Transaction with the same hash was already imported."
    if (message.match(/same hash was already imported|transaction nonce is too low|nonce too low/)) {
        logger.throwError("nonce has already been used", Logger.errors.NONCE_EXPIRED, {
           error, method, transaction
        });
    }

    // "Transaction gas price is too low. There is another transaction with same nonce in the queue. Try increasing the gas price or incrementing the nonce."
    if (message.match(/another transaction with same nonce/)) {
         logger.throwError("replacement fee too low", Logger.errors.REPLACEMENT_UNDERPRICED, {
            error, method, transaction
         });
    }

    if (message.match(/execution failed due to an exception|execution reverted/)) {
        logger.throwError("cannot estimate gas; transaction may fail or may require manual gas limit", Logger.errors.UNPREDICTABLE_GAS_LIMIT, {
            error, method, transaction
        });
    }

    throw error;
}

export class EtherscanProvider extends BaseProvider{
    readonly baseUrl: string;
    readonly apiKey: string | null;

    constructor(network?: Networkish, apiKey?: string) {
        super(network);

        defineReadOnly(this, "baseUrl", this.getBaseUrl());
        defineReadOnly(this, "apiKey", apiKey || null);
    }

    getBaseUrl(): string {
        switch(this.network ? this.network.name: "invalid") {
            case "homestead":
                return "https:/\/api.etherscan.io";
            case "goerli":
                return "https:/\/api-goerli.etherscan.io";
            case "sepolia":
                return "https:/\/api-sepolia.etherscan.io";
            case "matic":
                return "https:/\/api.polygonscan.com";
            case "maticmum":
                return "https:/\/api-testnet.polygonscan.com";
            case "arbitrum":
                return "https:/\/api.arbiscan.io";
            case "arbitrum-goerli":
                return "https:/\/api-goerli.arbiscan.io";
            case "optimism":
                return "https:/\/api-optimistic.etherscan.io";
            case "optimism-goerli":
                return "https:/\/api-goerli-optimistic.etherscan.io";
            default:
        }

        return logger.throwArgumentError("unsupported network", "network", this.network.name);
    }

    getUrl(module: string, params: Record<string, string>): string {
        const query = Object.keys(params).reduce((accum, key) => {
            const value = params[key];
            if (value != null) {
                accum += `&${ key }=${ value }`
            }
            return accum
        }, "");
        const apiKey = ((this.apiKey) ? `&apikey=${ this.apiKey }`: "");
        return `${ this.baseUrl }/api?module=${ module }${ query }${ apiKey }`;
    }

    getPostUrl(): string {
        return `${ this.baseUrl }/api`;
    }

    getPostData(module: string, params: Record<string, any>): Record<string, any> {
        params.module = module;
        params.apikey = this.apiKey;
        return params;
    }

    async fetch(module: string, params: Record<string, any>, post?: boolean): Promise<any> {
        const url = (post ? this.getPostUrl(): this.getUrl(module, params));
        const payload = (post ? this.getPostData(module, params): null);
        const procFunc = (module === "proxy") ? getJsonResult: getResult;

        this.emit("debug", {
            action: "request",
            request: url,
            provider: this
        });

        const connection: ConnectionInfo = {
            url: url,
            throttleSlotInterval: 1000,
            throttleCallback: (attempt: number, url: string) => {
                if (this.isCommunityResource()) {
                    showThrottleMessage();
                }
                return Promise.resolve(true);
            }
        };

        let payloadStr: string = null;
        if (payload) {
            connection.headers = { "content-type": "application/x-www-form-urlencoded; charset=UTF-8" };
            payloadStr = Object.keys(payload).map((key) => {
                return `${ key }=${ payload[key] }`
            }).join("&");
        }

        const result = await fetchJson(connection, payloadStr, procFunc || getJsonResult);

        this.emit("debug", {
            action: "response",
            request: url,
            response: deepCopy(result),
            provider: this
        });

        return result;
    }

    async detectNetwork(): Promise<Network> {
        return this.network;
    }

    async perform(method: string, params: any): Promise<any> {

        switch (method) {
            case "getBlockNumber":
                return this.fetch("proxy", { action: "eth_blockNumber" });

            case "getGasPrice":
                return this.fetch("proxy", { action: "eth_gasPrice" });

            case "getBalance":
                // Returns base-10 result
                return this.fetch("account", {
                    action: "balance",
                    address: params.address,
                    tag: params.blockTag
                });

            case "getTransactionCount":
                return this.fetch("proxy", {
                    action: "eth_getTransactionCount",
                    address: params.address,
                    tag: params.blockTag
                });

            case "getCode":
                return this.fetch("proxy", {
                    action: "eth_getCode",
                    address: params.address,
                    tag: params.blockTag
                });

            case "getStorageAt":
                return this.fetch("proxy", {
                    action: "eth_getStorageAt",
                    address: params.address,
                    position: params.position,
                    tag: params.blockTag
                });

            case "sendTransaction":
                return this.fetch("proxy", {
                    action: "eth_sendRawTransaction",
                    hex: params.signedTransaction
                }, true).catch((error) => {
                    return checkError("sendTransaction", error, params.signedTransaction);
                });

            case "getBlock":
                if (params.blockTag) {
                    return this.fetch("proxy", {
                        action: "eth_getBlockByNumber",
                        tag: params.blockTag,
                        boolean: (params.includeTransactions ? "true": "false")
                    });
                }
                throw new Error("getBlock by blockHash not implemented");

            case "getTransaction":
                return this.fetch("proxy", {
                    action: "eth_getTransactionByHash",
                    txhash: params.transactionHash
                });

            case "getTransactionReceipt":
                return this.fetch("proxy", {
                    action: "eth_getTransactionReceipt",
                    txhash: params.transactionHash
                });

            case "call": {
                if (params.blockTag !== "latest") {
                    throw new Error("EtherscanProvider does not support blockTag for call");
                }

                const postData = getTransactionPostData(params.transaction);
                postData.module = "proxy";
                postData.action = "eth_call";

                try {
                    return await this.fetch("proxy", postData, true);
                } catch (error) {
                    return checkError("call", error, params.transaction);
                }
            }

            case "estimateGas": {
                const postData = getTransactionPostData(params.transaction);
                postData.module = "proxy";
                postData.action = "eth_estimateGas";

                try {
                    return await this.fetch("proxy", postData, true);
                } catch (error) {
                    return checkError("estimateGas", error, params.transaction);
                }
            }

            case "getLogs": {
                const args: Record<string, any> = { action: "getLogs" }

                if (params.filter.fromBlock) {
                    args.fromBlock = checkLogTag(params.filter.fromBlock);
                }

                if (params.filter.toBlock) {
                    args.toBlock = checkLogTag(params.filter.toBlock);
                }

                if (params.filter.address) {
                    args.address = params.filter.address;
                }

                // @TODO: We can handle slightly more complicated logs using the logs API
                if (params.filter.topics && params.filter.topics.length > 0) {
                    if (params.filter.topics.length > 1) {
                        logger.throwError("unsupported topic count", Logger.errors.UNSUPPORTED_OPERATION, { topics: params.filter.topics });
                    }

                    if (params.filter.topics.length === 1) {
                        const topic0 = params.filter.topics[0];
                        if (typeof(topic0) !== "string" || topic0.length !== 66) {
                            logger.throwError("unsupported topic format", Logger.errors.UNSUPPORTED_OPERATION, { topic0: topic0 });
                        }
                        args.topic0 = topic0;
                    }
                }

                const logs: Array<any> = await this.fetch("logs", args);

                // Cache txHash => blockHash
                let blocks: { [tag: string]: string } = {};

                // Add any missing blockHash to the logs
                for (let i = 0; i < logs.length; i++) {
                    const log = logs[i];
                    if (log.blockHash != null) { continue; }
                    if (blocks[log.blockNumber] == null) {
                        const block = await this.getBlock(log.blockNumber);
                        if (block) {
                            blocks[log.blockNumber] = block.hash;
                        }
                    }
                    log.blockHash = blocks[log.blockNumber];
                }

                return logs;
            }

            case "getEtherPrice":
                if (this.network.name !== "homestead") { return 0.0; }
                return parseFloat((await this.fetch("stats", { action: "ethprice" })).ethusd);

            default:
                break;
         }

        return super.perform(method, params);
    }

    // Note: The `page` page parameter only allows pagination within the
    //       10,000 window available without a page and offset parameter
    //       Error: Result window is too large, PageNo x Offset size must
    //              be less than or equal to 10000
    async getHistory(addressOrName: string | Promise<string>, startBlock?: BlockTag, endBlock?: BlockTag): Promise<Array<TransactionResponse>> {
        const params = {
            action: "txlist",
            address: (await this.resolveName(addressOrName)),
            startblock: ((startBlock == null) ? 0: startBlock),
            endblock: ((endBlock == null) ? 99999999: endBlock),
            sort: "asc"
        };

        const result = await this.fetch("account", params);

        return result.map((tx: any) => {
            ["contractAddress", "to"].forEach(function(key) {
                if (tx[key] == "") { delete tx[key]; }
            });
            if (tx.creates == null && tx.contractAddress != null) {
                tx.creates = tx.contractAddress;
            }
            const item = this.formatter.transactionResponse(tx);
            if (tx.timeStamp) { item.timestamp = parseInt(tx.timeStamp); }
            return item;
        });
    }

    isCommunityResource(): boolean {
        return (this.apiKey == null);
    }
}
