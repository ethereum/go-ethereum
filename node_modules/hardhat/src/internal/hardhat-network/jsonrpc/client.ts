import {
  Address,
  bytesToHex as bufferToHex,
} from "@nomicfoundation/ethereumjs-util";
import fsExtra from "fs-extra";
import * as t from "io-ts";
import path from "path";

import {
  numberToRpcQuantity,
  rpcData,
  rpcQuantity,
} from "../../core/jsonrpc/types/base-types";
import {
  rpcBlock,
  RpcBlock,
  rpcBlockWithTransactions,
  RpcBlockWithTransactions,
} from "../../core/jsonrpc/types/output/block";
import { decodeJsonRpcResponse } from "../../core/jsonrpc/types/output/decodeJsonRpcResponse";
import { rpcLog } from "../../core/jsonrpc/types/output/log";
import { rpcTransactionReceipt } from "../../core/jsonrpc/types/output/receipt";
import { rpcTransaction } from "../../core/jsonrpc/types/output/transaction";
import { HttpProvider } from "../../core/providers/http";
import { createNonCryptographicHashBasedIdentifier } from "../../util/hash";
import { nullable } from "../../util/io-ts";

export class JsonRpcClient {
  private _cache: Map<string, any> = new Map();

  constructor(
    private _httpProvider: HttpProvider,
    private _networkId: number,
    private _latestBlockNumberOnCreation: bigint,
    private _maxReorg: bigint,
    private _forkCachePath?: string
  ) {}

  public getNetworkId(): number {
    return this._networkId;
  }

  public async getDebugTraceTransaction(transactionHash: Buffer): Promise<any> {
    return this._perform(
      "debug_traceTransaction",
      [bufferToHex(transactionHash)],
      t.object,
      () => undefined
    );
  }

  // Storage key must be 32 bytes long
  public async getStorageAt(
    address: Address,
    position: bigint,
    blockNumber: bigint
  ): Promise<Buffer> {
    return this._perform(
      "eth_getStorageAt",
      [
        address.toString(),
        numberToRpcQuantity(position),
        numberToRpcQuantity(blockNumber),
      ],
      rpcData,
      () => blockNumber
    );
  }

  public async getBlockByNumber(
    blockNumber: bigint,
    includeTransactions?: false
  ): Promise<RpcBlock | null>;

  public async getBlockByNumber(
    blockNumber: bigint,
    includeTransactions: true
  ): Promise<RpcBlockWithTransactions | null>;

  public async getBlockByNumber(
    blockNumber: bigint,
    includeTransactions = false
  ): Promise<RpcBlock | RpcBlockWithTransactions | null> {
    if (includeTransactions) {
      return this._perform(
        "eth_getBlockByNumber",
        [numberToRpcQuantity(blockNumber), true],
        nullable(rpcBlockWithTransactions),
        (block) => block?.number ?? undefined
      );
    }

    return this._perform(
      "eth_getBlockByNumber",
      [numberToRpcQuantity(blockNumber), false],
      nullable(rpcBlock),
      (block) => block?.number ?? undefined
    );
  }

  public async getBlockByHash(
    blockHash: Buffer,
    includeTransactions?: false
  ): Promise<RpcBlock | null>;

  public async getBlockByHash(
    blockHash: Buffer,
    includeTransactions: true
  ): Promise<RpcBlockWithTransactions | null>;

  public async getBlockByHash(
    blockHash: Buffer,
    includeTransactions = false
  ): Promise<RpcBlock | RpcBlockWithTransactions | null> {
    if (includeTransactions) {
      return this._perform(
        "eth_getBlockByHash",
        [bufferToHex(blockHash), true],
        nullable(rpcBlockWithTransactions),
        (block) => block?.number ?? undefined
      );
    }

    return this._perform(
      "eth_getBlockByHash",
      [bufferToHex(blockHash), false],
      nullable(rpcBlock),
      (block) => block?.number ?? undefined
    );
  }

  public async getTransactionByHash(transactionHash: Buffer) {
    return this._perform(
      "eth_getTransactionByHash",
      [bufferToHex(transactionHash)],
      nullable(rpcTransaction),
      (tx) => tx?.blockNumber ?? undefined
    );
  }

  public async getTransactionCount(address: Uint8Array, blockNumber: bigint) {
    return this._perform(
      "eth_getTransactionCount",
      [bufferToHex(address), numberToRpcQuantity(blockNumber)],
      rpcQuantity,
      () => blockNumber
    );
  }

  public async getTransactionReceipt(transactionHash: Buffer) {
    return this._perform(
      "eth_getTransactionReceipt",
      [bufferToHex(transactionHash)],
      nullable(rpcTransactionReceipt),
      (tx) => tx?.blockNumber ?? undefined
    );
  }

  public async getLogs(options: {
    fromBlock: bigint;
    toBlock: bigint;
    address?: Uint8Array | Uint8Array[];
    topics?: Array<Array<Uint8Array | null> | null>;
  }) {
    let address: string | string[] | undefined;
    if (options.address !== undefined) {
      address = Array.isArray(options.address)
        ? options.address.map((x) => bufferToHex(x))
        : bufferToHex(options.address);
    }
    let topics: Array<Array<string | null> | null> | undefined;
    if (options.topics !== undefined) {
      topics = options.topics.map((items) =>
        items !== null
          ? items.map((x) => (x !== null ? bufferToHex(x) : x))
          : null
      );
    }

    return this._perform(
      "eth_getLogs",
      [
        {
          fromBlock: numberToRpcQuantity(options.fromBlock),
          toBlock: numberToRpcQuantity(options.toBlock),
          address,
          topics,
        },
      ],
      t.array(rpcLog, "RpcLog Array"),
      () => options.toBlock
    );
  }

  public async getAccountData(
    address: Address,
    blockNumber: bigint
  ): Promise<{ code: Buffer; transactionCount: bigint; balance: bigint }> {
    const results = await this._performBatch(
      [
        {
          method: "eth_getCode",
          params: [address.toString(), numberToRpcQuantity(blockNumber)],
          tType: rpcData,
        },
        {
          method: "eth_getTransactionCount",
          params: [address.toString(), numberToRpcQuantity(blockNumber)],
          tType: rpcQuantity,
        },
        {
          method: "eth_getBalance",
          params: [address.toString(), numberToRpcQuantity(blockNumber)],
          tType: rpcQuantity,
        },
      ],
      () => blockNumber
    );

    return {
      code: results[0],
      transactionCount: results[1],
      balance: results[2],
    };
  }

  public async getLatestBlockNumber(): Promise<bigint> {
    return this._perform(
      "eth_blockNumber",
      [],
      rpcQuantity,
      (blockNumber) => blockNumber
    );
  }

  private async _perform<T>(
    method: string,
    params: any[],
    tType: t.Type<T>,
    getMaxAffectedBlockNumber: (decodedResult: T) => bigint | undefined
  ): Promise<T> {
    const cacheKey = this._getCacheKey(method, params);

    const cachedResult = this._getFromCache(cacheKey);
    if (cachedResult !== undefined) {
      return cachedResult;
    }

    if (this._forkCachePath !== undefined) {
      const diskCachedResult = await this._getFromDiskCache(
        this._forkCachePath,
        cacheKey,
        tType
      );
      if (diskCachedResult !== undefined) {
        this._storeInCache(cacheKey, diskCachedResult);
        return diskCachedResult;
      }
    }

    const rawResult = await this._send(method, params);
    const decodedResult = decodeJsonRpcResponse(rawResult, tType);

    const blockNumber = getMaxAffectedBlockNumber(decodedResult);
    if (this._canBeCached(blockNumber)) {
      this._storeInCache(cacheKey, decodedResult);

      if (this._forkCachePath !== undefined) {
        await this._storeInDiskCache(this._forkCachePath, cacheKey, rawResult);
      }
    }

    return decodedResult;
  }

  private async _performBatch(
    batch: Array<{
      method: string;
      params: any[];
      tType: t.Type<any>;
    }>,
    getMaxAffectedBlockNumber: (decodedResults: any[]) => bigint | undefined
  ): Promise<any[]> {
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
      const diskCachedResult = await this._getBatchFromDiskCache(
        this._forkCachePath,
        cacheKey,
        batch.map((b) => b.tType)
      );

      if (diskCachedResult !== undefined) {
        this._storeInCache(cacheKey, diskCachedResult);
        return diskCachedResult;
      }
    }

    const rawResults = await this._sendBatch(batch);
    const decodedResults = rawResults.map((result, i) =>
      decodeJsonRpcResponse(result, batch[i].tType)
    );

    const blockNumber = getMaxAffectedBlockNumber(decodedResults);
    if (this._canBeCached(blockNumber)) {
      this._storeInCache(cacheKey, decodedResults);

      if (this._forkCachePath !== undefined) {
        await this._storeInDiskCache(this._forkCachePath, cacheKey, rawResults);
      }
    }

    return decodedResults;
  }

  private async _send(
    method: string,
    params: any[],
    isRetryCall = false
  ): Promise<any> {
    try {
      return await this._httpProvider.request({ method, params });
    } catch (err: any) {
      if (this._shouldRetry(isRetryCall, err)) {
        return this._send(method, params, true);
      }

      // This is a workaround for this TurboGeth bug: https://github.com/ledgerwatch/turbo-geth/issues/1645
      const errMessage: string = err.message;
      if (err.code === -32000 && errMessage.includes("not found")) {
        return null;
      }

      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw err;
    }
  }

  private async _sendBatch(
    batch: Array<{ method: string; params: any[] }>,
    isRetryCall = false
  ): Promise<any[]> {
    try {
      return await this._httpProvider.sendBatch(batch);
    } catch (err) {
      if (this._shouldRetry(isRetryCall, err)) {
        return this._sendBatch(batch, true);
      }
      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw err;
    }
  }

  private _shouldRetry(isRetryCall: boolean, err: any): boolean {
    const errMessage: string = err.message;

    const isRetriableError =
      errMessage.includes("header not found") ||
      errMessage.includes("connect ETIMEDOUT");

    const isServiceUrl =
      this._httpProvider.url.includes("infura") ||
      this._httpProvider.url.includes("alchemyapi");

    return (
      !isRetryCall && isServiceUrl && err instanceof Error && isRetriableError
    );
  }

  private _getCacheKey(method: string, params: any[]) {
    const networkId = this.getNetworkId();
    const plaintextKey = `${networkId} ${method} ${JSON.stringify(params)}`;

    const hashed = createNonCryptographicHashBasedIdentifier(
      Buffer.from(plaintextKey, "utf8")
    );

    return hashed.toString("hex");
  }

  private _getBatchCacheKey(batch: Array<{ method: string; params: any[] }>) {
    let fakeMethod = "";
    const fakeParams = [];

    for (const entry of batch) {
      fakeMethod += entry.method;
      fakeParams.push(...entry.params);
    }

    return this._getCacheKey(fakeMethod, fakeParams);
  }

  private _getFromCache(cacheKey: string): any | undefined {
    return this._cache.get(cacheKey);
  }

  private _storeInCache(cacheKey: string, decodedResult: any) {
    this._cache.set(cacheKey, decodedResult);
  }

  private async _getFromDiskCache(
    forkCachePath: string,
    cacheKey: string,
    tType: t.Type<any>
  ): Promise<any | undefined> {
    const rawResult = await this._getRawFromDiskCache(forkCachePath, cacheKey);

    if (rawResult !== undefined) {
      return decodeJsonRpcResponse(rawResult, tType);
    }
  }

  private async _getBatchFromDiskCache(
    forkCachePath: string,
    cacheKey: string,
    tTypes: Array<t.Type<any>>
  ): Promise<any[] | undefined> {
    const rawResults = await this._getRawFromDiskCache(forkCachePath, cacheKey);

    if (!Array.isArray(rawResults)) {
      return undefined;
    }

    return rawResults.map((r, i) => decodeJsonRpcResponse(r, tTypes[i]));
  }

  private async _getRawFromDiskCache(
    forkCachePath: string,
    cacheKey: string
  ): Promise<unknown | undefined> {
    try {
      return await fsExtra.readJSON(
        this._getDiskCachePathForKey(forkCachePath, cacheKey),
        {
          encoding: "utf8",
        }
      );
    } catch (error: any) {
      if (error.code === "ENOENT") {
        return undefined;
      }

      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw error;
    }
  }

  private async _storeInDiskCache(
    forkCachePath: string,
    cacheKey: string,
    rawResult: any
  ) {
    const requestPath = this._getDiskCachePathForKey(forkCachePath, cacheKey);

    await fsExtra.ensureDir(path.dirname(requestPath));
    await fsExtra.writeJSON(requestPath, rawResult, {
      encoding: "utf8",
    });
  }

  private _getDiskCachePathForKey(forkCachePath: string, key: string): string {
    return path.join(
      forkCachePath,
      `network-${this._networkId!}`,
      `request-${key}.json`
    );
  }

  private _canBeCached(blockNumber: bigint | undefined) {
    if (blockNumber === undefined) {
      return false;
    }

    return !this._canBeReorgedOut(blockNumber);
  }

  private _canBeReorgedOut(blockNumber: bigint) {
    const maxSafeBlockNumber =
      this._latestBlockNumberOnCreation - this._maxReorg;
    return blockNumber > maxSafeBlockNumber;
  }
}
