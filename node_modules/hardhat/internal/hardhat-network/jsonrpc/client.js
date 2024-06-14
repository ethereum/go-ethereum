"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.JsonRpcClient = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const fs_extra_1 = __importDefault(require("fs-extra"));
const t = __importStar(require("io-ts"));
const path_1 = __importDefault(require("path"));
const base_types_1 = require("../../core/jsonrpc/types/base-types");
const block_1 = require("../../core/jsonrpc/types/output/block");
const decodeJsonRpcResponse_1 = require("../../core/jsonrpc/types/output/decodeJsonRpcResponse");
const log_1 = require("../../core/jsonrpc/types/output/log");
const receipt_1 = require("../../core/jsonrpc/types/output/receipt");
const transaction_1 = require("../../core/jsonrpc/types/output/transaction");
const hash_1 = require("../../util/hash");
const io_ts_1 = require("../../util/io-ts");
class JsonRpcClient {
    constructor(_httpProvider, _networkId, _latestBlockNumberOnCreation, _maxReorg, _forkCachePath) {
        this._httpProvider = _httpProvider;
        this._networkId = _networkId;
        this._latestBlockNumberOnCreation = _latestBlockNumberOnCreation;
        this._maxReorg = _maxReorg;
        this._forkCachePath = _forkCachePath;
        this._cache = new Map();
    }
    getNetworkId() {
        return this._networkId;
    }
    async getDebugTraceTransaction(transactionHash) {
        return this._perform("debug_traceTransaction", [(0, ethereumjs_util_1.bytesToHex)(transactionHash)], t.object, () => undefined);
    }
    // Storage key must be 32 bytes long
    async getStorageAt(address, position, blockNumber) {
        return this._perform("eth_getStorageAt", [
            address.toString(),
            (0, base_types_1.numberToRpcQuantity)(position),
            (0, base_types_1.numberToRpcQuantity)(blockNumber),
        ], base_types_1.rpcData, () => blockNumber);
    }
    async getBlockByNumber(blockNumber, includeTransactions = false) {
        if (includeTransactions) {
            return this._perform("eth_getBlockByNumber", [(0, base_types_1.numberToRpcQuantity)(blockNumber), true], (0, io_ts_1.nullable)(block_1.rpcBlockWithTransactions), (block) => block?.number ?? undefined);
        }
        return this._perform("eth_getBlockByNumber", [(0, base_types_1.numberToRpcQuantity)(blockNumber), false], (0, io_ts_1.nullable)(block_1.rpcBlock), (block) => block?.number ?? undefined);
    }
    async getBlockByHash(blockHash, includeTransactions = false) {
        if (includeTransactions) {
            return this._perform("eth_getBlockByHash", [(0, ethereumjs_util_1.bytesToHex)(blockHash), true], (0, io_ts_1.nullable)(block_1.rpcBlockWithTransactions), (block) => block?.number ?? undefined);
        }
        return this._perform("eth_getBlockByHash", [(0, ethereumjs_util_1.bytesToHex)(blockHash), false], (0, io_ts_1.nullable)(block_1.rpcBlock), (block) => block?.number ?? undefined);
    }
    async getTransactionByHash(transactionHash) {
        return this._perform("eth_getTransactionByHash", [(0, ethereumjs_util_1.bytesToHex)(transactionHash)], (0, io_ts_1.nullable)(transaction_1.rpcTransaction), (tx) => tx?.blockNumber ?? undefined);
    }
    async getTransactionCount(address, blockNumber) {
        return this._perform("eth_getTransactionCount", [(0, ethereumjs_util_1.bytesToHex)(address), (0, base_types_1.numberToRpcQuantity)(blockNumber)], base_types_1.rpcQuantity, () => blockNumber);
    }
    async getTransactionReceipt(transactionHash) {
        return this._perform("eth_getTransactionReceipt", [(0, ethereumjs_util_1.bytesToHex)(transactionHash)], (0, io_ts_1.nullable)(receipt_1.rpcTransactionReceipt), (tx) => tx?.blockNumber ?? undefined);
    }
    async getLogs(options) {
        let address;
        if (options.address !== undefined) {
            address = Array.isArray(options.address)
                ? options.address.map((x) => (0, ethereumjs_util_1.bytesToHex)(x))
                : (0, ethereumjs_util_1.bytesToHex)(options.address);
        }
        let topics;
        if (options.topics !== undefined) {
            topics = options.topics.map((items) => items !== null
                ? items.map((x) => (x !== null ? (0, ethereumjs_util_1.bytesToHex)(x) : x))
                : null);
        }
        return this._perform("eth_getLogs", [
            {
                fromBlock: (0, base_types_1.numberToRpcQuantity)(options.fromBlock),
                toBlock: (0, base_types_1.numberToRpcQuantity)(options.toBlock),
                address,
                topics,
            },
        ], t.array(log_1.rpcLog, "RpcLog Array"), () => options.toBlock);
    }
    async getAccountData(address, blockNumber) {
        const results = await this._performBatch([
            {
                method: "eth_getCode",
                params: [address.toString(), (0, base_types_1.numberToRpcQuantity)(blockNumber)],
                tType: base_types_1.rpcData,
            },
            {
                method: "eth_getTransactionCount",
                params: [address.toString(), (0, base_types_1.numberToRpcQuantity)(blockNumber)],
                tType: base_types_1.rpcQuantity,
            },
            {
                method: "eth_getBalance",
                params: [address.toString(), (0, base_types_1.numberToRpcQuantity)(blockNumber)],
                tType: base_types_1.rpcQuantity,
            },
        ], () => blockNumber);
        return {
            code: results[0],
            transactionCount: results[1],
            balance: results[2],
        };
    }
    async getLatestBlockNumber() {
        return this._perform("eth_blockNumber", [], base_types_1.rpcQuantity, (blockNumber) => blockNumber);
    }
    async _perform(method, params, tType, getMaxAffectedBlockNumber) {
        const cacheKey = this._getCacheKey(method, params);
        const cachedResult = this._getFromCache(cacheKey);
        if (cachedResult !== undefined) {
            return cachedResult;
        }
        if (this._forkCachePath !== undefined) {
            const diskCachedResult = await this._getFromDiskCache(this._forkCachePath, cacheKey, tType);
            if (diskCachedResult !== undefined) {
                this._storeInCache(cacheKey, diskCachedResult);
                return diskCachedResult;
            }
        }
        const rawResult = await this._send(method, params);
        const decodedResult = (0, decodeJsonRpcResponse_1.decodeJsonRpcResponse)(rawResult, tType);
        const blockNumber = getMaxAffectedBlockNumber(decodedResult);
        if (this._canBeCached(blockNumber)) {
            this._storeInCache(cacheKey, decodedResult);
            if (this._forkCachePath !== undefined) {
                await this._storeInDiskCache(this._forkCachePath, cacheKey, rawResult);
            }
        }
        return decodedResult;
    }
    async _performBatch(batch, getMaxAffectedBlockNumber) {
        // Perform Batch caches the entire batch at once.
        // It could implement something more clever, like caching per request
        // but it's only used in one place, and those other requests aren't
        // used anywhere else.
        const cacheKey = this._getBatchCacheKey(batch);
        const cachedResult = this._getFromCache(cacheKey);
        if (cachedResult !== undefined) {
            return cachedResult;
        }
        if (this._forkCachePath !== undefined) {
            const diskCachedResult = await this._getBatchFromDiskCache(this._forkCachePath, cacheKey, batch.map((b) => b.tType));
            if (diskCachedResult !== undefined) {
                this._storeInCache(cacheKey, diskCachedResult);
                return diskCachedResult;
            }
        }
        const rawResults = await this._sendBatch(batch);
        const decodedResults = rawResults.map((result, i) => (0, decodeJsonRpcResponse_1.decodeJsonRpcResponse)(result, batch[i].tType));
        const blockNumber = getMaxAffectedBlockNumber(decodedResults);
        if (this._canBeCached(blockNumber)) {
            this._storeInCache(cacheKey, decodedResults);
            if (this._forkCachePath !== undefined) {
                await this._storeInDiskCache(this._forkCachePath, cacheKey, rawResults);
            }
        }
        return decodedResults;
    }
    async _send(method, params, isRetryCall = false) {
        try {
            return await this._httpProvider.request({ method, params });
        }
        catch (err) {
            if (this._shouldRetry(isRetryCall, err)) {
                return this._send(method, params, true);
            }
            // This is a workaround for this TurboGeth bug: https://github.com/ledgerwatch/turbo-geth/issues/1645
            const errMessage = err.message;
            if (err.code === -32000 && errMessage.includes("not found")) {
                return null;
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw err;
        }
    }
    async _sendBatch(batch, isRetryCall = false) {
        try {
            return await this._httpProvider.sendBatch(batch);
        }
        catch (err) {
            if (this._shouldRetry(isRetryCall, err)) {
                return this._sendBatch(batch, true);
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw err;
        }
    }
    _shouldRetry(isRetryCall, err) {
        const errMessage = err.message;
        const isRetriableError = errMessage.includes("header not found") ||
            errMessage.includes("connect ETIMEDOUT");
        const isServiceUrl = this._httpProvider.url.includes("infura") ||
            this._httpProvider.url.includes("alchemyapi");
        return (!isRetryCall && isServiceUrl && err instanceof Error && isRetriableError);
    }
    _getCacheKey(method, params) {
        const networkId = this.getNetworkId();
        const plaintextKey = `${networkId} ${method} ${JSON.stringify(params)}`;
        const hashed = (0, hash_1.createNonCryptographicHashBasedIdentifier)(Buffer.from(plaintextKey, "utf8"));
        return hashed.toString("hex");
    }
    _getBatchCacheKey(batch) {
        let fakeMethod = "";
        const fakeParams = [];
        for (const entry of batch) {
            fakeMethod += entry.method;
            fakeParams.push(...entry.params);
        }
        return this._getCacheKey(fakeMethod, fakeParams);
    }
    _getFromCache(cacheKey) {
        return this._cache.get(cacheKey);
    }
    _storeInCache(cacheKey, decodedResult) {
        this._cache.set(cacheKey, decodedResult);
    }
    async _getFromDiskCache(forkCachePath, cacheKey, tType) {
        const rawResult = await this._getRawFromDiskCache(forkCachePath, cacheKey);
        if (rawResult !== undefined) {
            return (0, decodeJsonRpcResponse_1.decodeJsonRpcResponse)(rawResult, tType);
        }
    }
    async _getBatchFromDiskCache(forkCachePath, cacheKey, tTypes) {
        const rawResults = await this._getRawFromDiskCache(forkCachePath, cacheKey);
        if (!Array.isArray(rawResults)) {
            return undefined;
        }
        return rawResults.map((r, i) => (0, decodeJsonRpcResponse_1.decodeJsonRpcResponse)(r, tTypes[i]));
    }
    async _getRawFromDiskCache(forkCachePath, cacheKey) {
        try {
            return await fs_extra_1.default.readJSON(this._getDiskCachePathForKey(forkCachePath, cacheKey), {
                encoding: "utf8",
            });
        }
        catch (error) {
            if (error.code === "ENOENT") {
                return undefined;
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw error;
        }
    }
    async _storeInDiskCache(forkCachePath, cacheKey, rawResult) {
        const requestPath = this._getDiskCachePathForKey(forkCachePath, cacheKey);
        await fs_extra_1.default.ensureDir(path_1.default.dirname(requestPath));
        await fs_extra_1.default.writeJSON(requestPath, rawResult, {
            encoding: "utf8",
        });
    }
    _getDiskCachePathForKey(forkCachePath, key) {
        return path_1.default.join(forkCachePath, `network-${this._networkId}`, `request-${key}.json`);
    }
    _canBeCached(blockNumber) {
        if (blockNumber === undefined) {
            return false;
        }
        return !this._canBeReorgedOut(blockNumber);
    }
    _canBeReorgedOut(blockNumber) {
        const maxSafeBlockNumber = this._latestBlockNumberOnCreation - this._maxReorg;
        return blockNumber > maxSafeBlockNumber;
    }
}
exports.JsonRpcClient = JsonRpcClient;
//# sourceMappingURL=client.js.map