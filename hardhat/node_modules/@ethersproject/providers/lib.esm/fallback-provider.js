"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
import { Provider } from "@ethersproject/abstract-provider";
import { BigNumber } from "@ethersproject/bignumber";
import { isHexString } from "@ethersproject/bytes";
import { deepCopy, defineReadOnly, shallowCopy } from "@ethersproject/properties";
import { shuffled } from "@ethersproject/random";
import { poll } from "@ethersproject/web";
import { BaseProvider } from "./base-provider";
import { isCommunityResource } from "./formatter";
import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);
function now() { return (new Date()).getTime(); }
// Returns to network as long as all agree, or null if any is null.
// Throws an error if any two networks do not match.
function checkNetworks(networks) {
    let result = null;
    for (let i = 0; i < networks.length; i++) {
        const network = networks[i];
        // Null! We do not know our network; bail.
        if (network == null) {
            return null;
        }
        if (result) {
            // Make sure the network matches the previous networks
            if (!(result.name === network.name && result.chainId === network.chainId &&
                ((result.ensAddress === network.ensAddress) || (result.ensAddress == null && network.ensAddress == null)))) {
                logger.throwArgumentError("provider mismatch", "networks", networks);
            }
        }
        else {
            result = network;
        }
    }
    return result;
}
function median(values, maxDelta) {
    values = values.slice().sort();
    const middle = Math.floor(values.length / 2);
    // Odd length; take the middle
    if (values.length % 2) {
        return values[middle];
    }
    // Even length; take the average of the two middle
    const a = values[middle - 1], b = values[middle];
    if (maxDelta != null && Math.abs(a - b) > maxDelta) {
        return null;
    }
    return (a + b) / 2;
}
function serialize(value) {
    if (value === null) {
        return "null";
    }
    else if (typeof (value) === "number" || typeof (value) === "boolean") {
        return JSON.stringify(value);
    }
    else if (typeof (value) === "string") {
        return value;
    }
    else if (BigNumber.isBigNumber(value)) {
        return value.toString();
    }
    else if (Array.isArray(value)) {
        return JSON.stringify(value.map((i) => serialize(i)));
    }
    else if (typeof (value) === "object") {
        const keys = Object.keys(value);
        keys.sort();
        return "{" + keys.map((key) => {
            let v = value[key];
            if (typeof (v) === "function") {
                v = "[function]";
            }
            else {
                v = serialize(v);
            }
            return JSON.stringify(key) + ":" + v;
        }).join(",") + "}";
    }
    throw new Error("unknown value type: " + typeof (value));
}
// Next request ID to use for emitting debug info
let nextRid = 1;
;
function stall(duration) {
    let cancel = null;
    let timer = null;
    let promise = (new Promise((resolve) => {
        cancel = function () {
            if (timer) {
                clearTimeout(timer);
                timer = null;
            }
            resolve();
        };
        timer = setTimeout(cancel, duration);
    }));
    const wait = (func) => {
        promise = promise.then(func);
        return promise;
    };
    function getPromise() {
        return promise;
    }
    return { cancel, getPromise, wait };
}
const ForwardErrors = [
    Logger.errors.CALL_EXCEPTION,
    Logger.errors.INSUFFICIENT_FUNDS,
    Logger.errors.NONCE_EXPIRED,
    Logger.errors.REPLACEMENT_UNDERPRICED,
    Logger.errors.UNPREDICTABLE_GAS_LIMIT
];
const ForwardProperties = [
    "address",
    "args",
    "errorArgs",
    "errorSignature",
    "method",
    "transaction",
];
;
function exposeDebugConfig(config, now) {
    const result = {
        weight: config.weight
    };
    Object.defineProperty(result, "provider", { get: () => config.provider });
    if (config.start) {
        result.start = config.start;
    }
    if (now) {
        result.duration = (now - config.start);
    }
    if (config.done) {
        if (config.error) {
            result.error = config.error;
        }
        else {
            result.result = config.result || null;
        }
    }
    return result;
}
function normalizedTally(normalize, quorum) {
    return function (configs) {
        // Count the votes for each result
        const tally = {};
        configs.forEach((c) => {
            const value = normalize(c.result);
            if (!tally[value]) {
                tally[value] = { count: 0, result: c.result };
            }
            tally[value].count++;
        });
        // Check for a quorum on any given result
        const keys = Object.keys(tally);
        for (let i = 0; i < keys.length; i++) {
            const check = tally[keys[i]];
            if (check.count >= quorum) {
                return check.result;
            }
        }
        // No quroum
        return undefined;
    };
}
function getProcessFunc(provider, method, params) {
    let normalize = serialize;
    switch (method) {
        case "getBlockNumber":
            // Return the median value, unless there is (median + 1) is also
            // present, in which case that is probably true and the median
            // is going to be stale soon. In the event of a malicious node,
            // the lie will be true soon enough.
            return function (configs) {
                const values = configs.map((c) => c.result);
                // Get the median block number
                let blockNumber = median(configs.map((c) => c.result), 2);
                if (blockNumber == null) {
                    return undefined;
                }
                blockNumber = Math.ceil(blockNumber);
                // If the next block height is present, its prolly safe to use
                if (values.indexOf(blockNumber + 1) >= 0) {
                    blockNumber++;
                }
                // Don't ever roll back the blockNumber
                if (blockNumber >= provider._highestBlockNumber) {
                    provider._highestBlockNumber = blockNumber;
                }
                return provider._highestBlockNumber;
            };
        case "getGasPrice":
            // Return the middle (round index up) value, similar to median
            // but do not average even entries and choose the higher.
            // Malicious actors must compromise 50% of the nodes to lie.
            return function (configs) {
                const values = configs.map((c) => c.result);
                values.sort();
                return values[Math.floor(values.length / 2)];
            };
        case "getEtherPrice":
            // Returns the median price. Malicious actors must compromise at
            // least 50% of the nodes to lie (in a meaningful way).
            return function (configs) {
                return median(configs.map((c) => c.result));
            };
        // No additional normalizing required; serialize is enough
        case "getBalance":
        case "getTransactionCount":
        case "getCode":
        case "getStorageAt":
        case "call":
        case "estimateGas":
        case "getLogs":
            break;
        // We drop the confirmations from transactions as it is approximate
        case "getTransaction":
        case "getTransactionReceipt":
            normalize = function (tx) {
                if (tx == null) {
                    return null;
                }
                tx = shallowCopy(tx);
                tx.confirmations = -1;
                return serialize(tx);
            };
            break;
        // We drop the confirmations from transactions as it is approximate
        case "getBlock":
            // We drop the confirmations from transactions as it is approximate
            if (params.includeTransactions) {
                normalize = function (block) {
                    if (block == null) {
                        return null;
                    }
                    block = shallowCopy(block);
                    block.transactions = block.transactions.map((tx) => {
                        tx = shallowCopy(tx);
                        tx.confirmations = -1;
                        return tx;
                    });
                    return serialize(block);
                };
            }
            else {
                normalize = function (block) {
                    if (block == null) {
                        return null;
                    }
                    return serialize(block);
                };
            }
            break;
        default:
            throw new Error("unknown method: " + method);
    }
    // Return the result if and only if the expected quorum is
    // satisfied and agreed upon for the final result.
    return normalizedTally(normalize, provider.quorum);
}
// If we are doing a blockTag query, we need to make sure the backend is
// caught up to the FallbackProvider, before sending a request to it.
function waitForSync(config, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        const provider = (config.provider);
        if ((provider.blockNumber != null && provider.blockNumber >= blockNumber) || blockNumber === -1) {
            return provider;
        }
        return poll(() => {
            return new Promise((resolve, reject) => {
                setTimeout(function () {
                    // We are synced
                    if (provider.blockNumber >= blockNumber) {
                        return resolve(provider);
                    }
                    // We're done; just quit
                    if (config.cancelled) {
                        return resolve(null);
                    }
                    // Try again, next block
                    return resolve(undefined);
                }, 0);
            });
        }, { oncePoll: provider });
    });
}
function getRunner(config, currentBlockNumber, method, params) {
    return __awaiter(this, void 0, void 0, function* () {
        let provider = config.provider;
        switch (method) {
            case "getBlockNumber":
            case "getGasPrice":
                return provider[method]();
            case "getEtherPrice":
                if (provider.getEtherPrice) {
                    return provider.getEtherPrice();
                }
                break;
            case "getBalance":
            case "getTransactionCount":
            case "getCode":
                if (params.blockTag && isHexString(params.blockTag)) {
                    provider = yield waitForSync(config, currentBlockNumber);
                }
                return provider[method](params.address, params.blockTag || "latest");
            case "getStorageAt":
                if (params.blockTag && isHexString(params.blockTag)) {
                    provider = yield waitForSync(config, currentBlockNumber);
                }
                return provider.getStorageAt(params.address, params.position, params.blockTag || "latest");
            case "getBlock":
                if (params.blockTag && isHexString(params.blockTag)) {
                    provider = yield waitForSync(config, currentBlockNumber);
                }
                return provider[(params.includeTransactions ? "getBlockWithTransactions" : "getBlock")](params.blockTag || params.blockHash);
            case "call":
            case "estimateGas":
                if (params.blockTag && isHexString(params.blockTag)) {
                    provider = yield waitForSync(config, currentBlockNumber);
                }
                if (method === "call" && params.blockTag) {
                    return provider[method](params.transaction, params.blockTag);
                }
                return provider[method](params.transaction);
            case "getTransaction":
            case "getTransactionReceipt":
                return provider[method](params.transactionHash);
            case "getLogs": {
                let filter = params.filter;
                if ((filter.fromBlock && isHexString(filter.fromBlock)) || (filter.toBlock && isHexString(filter.toBlock))) {
                    provider = yield waitForSync(config, currentBlockNumber);
                }
                return provider.getLogs(filter);
            }
        }
        return logger.throwError("unknown method error", Logger.errors.UNKNOWN_ERROR, {
            method: method,
            params: params
        });
    });
}
export class FallbackProvider extends BaseProvider {
    constructor(providers, quorum) {
        if (providers.length === 0) {
            logger.throwArgumentError("missing providers", "providers", providers);
        }
        const providerConfigs = providers.map((configOrProvider, index) => {
            if (Provider.isProvider(configOrProvider)) {
                const stallTimeout = isCommunityResource(configOrProvider) ? 2000 : 750;
                const priority = 1;
                return Object.freeze({ provider: configOrProvider, weight: 1, stallTimeout, priority });
            }
            const config = shallowCopy(configOrProvider);
            if (config.priority == null) {
                config.priority = 1;
            }
            if (config.stallTimeout == null) {
                config.stallTimeout = isCommunityResource(configOrProvider) ? 2000 : 750;
            }
            if (config.weight == null) {
                config.weight = 1;
            }
            const weight = config.weight;
            if (weight % 1 || weight > 512 || weight < 1) {
                logger.throwArgumentError("invalid weight; must be integer in [1, 512]", `providers[${index}].weight`, weight);
            }
            return Object.freeze(config);
        });
        const total = providerConfigs.reduce((accum, c) => (accum + c.weight), 0);
        if (quorum == null) {
            quorum = total / 2;
        }
        else if (quorum > total) {
            logger.throwArgumentError("quorum will always fail; larger than total weight", "quorum", quorum);
        }
        // Are all providers' networks are known
        let networkOrReady = checkNetworks(providerConfigs.map((c) => (c.provider).network));
        // Not all networks are known; we must stall
        if (networkOrReady == null) {
            networkOrReady = new Promise((resolve, reject) => {
                setTimeout(() => {
                    this.detectNetwork().then(resolve, reject);
                }, 0);
            });
        }
        super(networkOrReady);
        // Preserve a copy, so we do not get mutated
        defineReadOnly(this, "providerConfigs", Object.freeze(providerConfigs));
        defineReadOnly(this, "quorum", quorum);
        this._highestBlockNumber = -1;
    }
    detectNetwork() {
        return __awaiter(this, void 0, void 0, function* () {
            const networks = yield Promise.all(this.providerConfigs.map((c) => c.provider.getNetwork()));
            return checkNetworks(networks);
        });
    }
    perform(method, params) {
        return __awaiter(this, void 0, void 0, function* () {
            // Sending transactions is special; always broadcast it to all backends
            if (method === "sendTransaction") {
                const results = yield Promise.all(this.providerConfigs.map((c) => {
                    return c.provider.sendTransaction(params.signedTransaction).then((result) => {
                        return result.hash;
                    }, (error) => {
                        return error;
                    });
                }));
                // Any success is good enough (other errors are likely "already seen" errors
                for (let i = 0; i < results.length; i++) {
                    const result = results[i];
                    if (typeof (result) === "string") {
                        return result;
                    }
                }
                // They were all an error; pick the first error
                throw results[0];
            }
            // We need to make sure we are in sync with our backends, so we need
            // to know this before we can make a lot of calls
            if (this._highestBlockNumber === -1 && method !== "getBlockNumber") {
                yield this.getBlockNumber();
            }
            const processFunc = getProcessFunc(this, method, params);
            // Shuffle the providers and then sort them by their priority; we
            // shallowCopy them since we will store the result in them too
            const configs = shuffled(this.providerConfigs.map(shallowCopy));
            configs.sort((a, b) => (a.priority - b.priority));
            const currentBlockNumber = this._highestBlockNumber;
            let i = 0;
            let first = true;
            while (true) {
                const t0 = now();
                // Compute the inflight weight (exclude anything past)
                let inflightWeight = configs.filter((c) => (c.runner && ((t0 - c.start) < c.stallTimeout)))
                    .reduce((accum, c) => (accum + c.weight), 0);
                // Start running enough to meet quorum
                while (inflightWeight < this.quorum && i < configs.length) {
                    const config = configs[i++];
                    const rid = nextRid++;
                    config.start = now();
                    config.staller = stall(config.stallTimeout);
                    config.staller.wait(() => { config.staller = null; });
                    config.runner = getRunner(config, currentBlockNumber, method, params).then((result) => {
                        config.done = true;
                        config.result = result;
                        if (this.listenerCount("debug")) {
                            this.emit("debug", {
                                action: "request",
                                rid: rid,
                                backend: exposeDebugConfig(config, now()),
                                request: { method: method, params: deepCopy(params) },
                                provider: this
                            });
                        }
                    }, (error) => {
                        config.done = true;
                        config.error = error;
                        if (this.listenerCount("debug")) {
                            this.emit("debug", {
                                action: "request",
                                rid: rid,
                                backend: exposeDebugConfig(config, now()),
                                request: { method: method, params: deepCopy(params) },
                                provider: this
                            });
                        }
                    });
                    if (this.listenerCount("debug")) {
                        this.emit("debug", {
                            action: "request",
                            rid: rid,
                            backend: exposeDebugConfig(config, null),
                            request: { method: method, params: deepCopy(params) },
                            provider: this
                        });
                    }
                    inflightWeight += config.weight;
                }
                // Wait for anything meaningful to finish or stall out
                const waiting = [];
                configs.forEach((c) => {
                    if (c.done || !c.runner) {
                        return;
                    }
                    waiting.push(c.runner);
                    if (c.staller) {
                        waiting.push(c.staller.getPromise());
                    }
                });
                if (waiting.length) {
                    yield Promise.race(waiting);
                }
                // Check the quorum and process the results; the process function
                // may additionally decide the quorum is not met
                const results = configs.filter((c) => (c.done && c.error == null));
                if (results.length >= this.quorum) {
                    const result = processFunc(results);
                    if (result !== undefined) {
                        // Shut down any stallers
                        configs.forEach(c => {
                            if (c.staller) {
                                c.staller.cancel();
                            }
                            c.cancelled = true;
                        });
                        return result;
                    }
                    if (!first) {
                        yield stall(100).getPromise();
                    }
                    first = false;
                }
                // No result, check for errors that should be forwarded
                const errors = configs.reduce((accum, c) => {
                    if (!c.done || c.error == null) {
                        return accum;
                    }
                    const code = (c.error).code;
                    if (ForwardErrors.indexOf(code) >= 0) {
                        if (!accum[code]) {
                            accum[code] = { error: c.error, weight: 0 };
                        }
                        accum[code].weight += c.weight;
                    }
                    return accum;
                }, ({}));
                Object.keys(errors).forEach((errorCode) => {
                    const tally = errors[errorCode];
                    if (tally.weight < this.quorum) {
                        return;
                    }
                    // Shut down any stallers
                    configs.forEach(c => {
                        if (c.staller) {
                            c.staller.cancel();
                        }
                        c.cancelled = true;
                    });
                    const e = (tally.error);
                    const props = {};
                    ForwardProperties.forEach((name) => {
                        if (e[name] == null) {
                            return;
                        }
                        props[name] = e[name];
                    });
                    logger.throwError(e.reason || e.message, errorCode, props);
                });
                // All configs have run to completion; we will never get more data
                if (configs.filter((c) => !c.done).length === 0) {
                    break;
                }
            }
            // Shut down any stallers; shouldn't be any
            configs.forEach(c => {
                if (c.staller) {
                    c.staller.cancel();
                }
                c.cancelled = true;
            });
            return logger.throwError("failed to meet quorum", Logger.errors.SERVER_ERROR, {
                method: method,
                params: params,
                //results: configs.map((c) => c.result),
                //errors: configs.map((c) => c.error),
                results: configs.map((c) => exposeDebugConfig(c)),
                provider: this
            });
        });
    }
}
//# sourceMappingURL=fallback-provider.js.map