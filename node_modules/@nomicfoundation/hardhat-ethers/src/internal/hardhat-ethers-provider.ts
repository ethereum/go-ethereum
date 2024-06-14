import type {
  AddressLike,
  BlockTag,
  TransactionRequest,
  Filter,
  FilterByBlockHash,
  Listener,
  ProviderEvent,
  PerformActionTransaction,
  TransactionResponseParams,
  BlockParams,
  TransactionReceiptParams,
  LogParams,
  PerformActionFilter,
  EventFilter,
} from "ethers";
import type LodashIsEqualT from "lodash.isequal";

import debug from "debug";
import {
  Block,
  FeeData,
  Log,
  Network as EthersNetwork,
  Transaction,
  TransactionReceipt,
  TransactionResponse,
  ethers,
  getBigInt,
  isHexString,
  resolveAddress,
  toQuantity,
} from "ethers";
import { EthereumProvider } from "hardhat/types";

import { HardhatEthersSigner } from "../signers";
import {
  copyRequest,
  formatBlock,
  formatLog,
  formatTransactionReceipt,
  formatTransactionResponse,
  getRpcTransaction,
} from "./ethers-utils";
import {
  AccountIndexOutOfRange,
  BroadcastedTxDifferentHash,
  HardhatEthersError,
  UnsupportedEventError,
  NotImplementedError,
} from "./errors";

const log = debug("hardhat:hardhat-ethers:provider");

interface ListenerItem {
  listener: Listener;
  once: boolean;
}

interface EventListenerItem {
  event: EventFilter;
  // map from the given listener to the block listener registered for that listener
  listenersMap: Map<Listener, Listener>;
}

// this type has a more explicit and type-safe list
// of the events that we support
type HardhatEthersProviderEvent =
  | {
      kind: "block";
    }
  | {
      kind: "transactionHash";
      txHash: string;
    }
  | {
      kind: "event";
      eventFilter: EventFilter;
    };

export class HardhatEthersProvider implements ethers.Provider {
  private _isHardhatNetworkCached: boolean | undefined;

  // event-emitter related fields
  private _latestBlockNumberPolled: number | undefined;
  private _blockListeners: ListenerItem[] = [];
  private _transactionHashListeners: Map<string, ListenerItem[]> = new Map();
  private _eventListeners: EventListenerItem[] = [];

  private _transactionHashPollingTimeout: NodeJS.Timeout | undefined;
  private _blockPollingTimeout: NodeJS.Timeout | undefined;

  constructor(
    private readonly _hardhatProvider: EthereumProvider,
    private readonly _networkName: string
  ) {}

  public get provider(): this {
    return this;
  }

  public destroy() {}

  public async send(method: string, params?: any[]): Promise<any> {
    return this._hardhatProvider.send(method, params);
  }

  public async getSigner(
    address?: number | string
  ): Promise<HardhatEthersSigner> {
    if (address === null || address === undefined) {
      address = 0;
    }

    const accountsPromise = this.send("eth_accounts", []);

    // Account index
    if (typeof address === "number") {
      const accounts: string[] = await accountsPromise;
      if (address >= accounts.length) {
        throw new AccountIndexOutOfRange(address, accounts.length);
      }
      return HardhatEthersSigner.create(this, accounts[address]);
    }

    if (typeof address === "string") {
      return HardhatEthersSigner.create(this, address);
    }

    throw new HardhatEthersError(`Couldn't get account ${address as any}`);
  }

  public async getBlockNumber(): Promise<number> {
    const blockNumber = await this._hardhatProvider.send("eth_blockNumber");

    return Number(blockNumber);
  }

  public async getNetwork(): Promise<EthersNetwork> {
    const chainId = await this._hardhatProvider.send("eth_chainId");
    return new EthersNetwork(this._networkName, Number(chainId));
  }

  public async getFeeData(): Promise<ethers.FeeData> {
    let gasPrice: bigint | undefined;
    let maxFeePerGas: bigint | undefined;
    let maxPriorityFeePerGas: bigint | undefined;

    try {
      gasPrice = BigInt(await this._hardhatProvider.send("eth_gasPrice"));
    } catch {}

    const latestBlock = await this.getBlock("latest");
    const baseFeePerGas = latestBlock?.baseFeePerGas;
    if (baseFeePerGas !== undefined && baseFeePerGas !== null) {
      try {
        maxPriorityFeePerGas = BigInt(
          await this._hardhatProvider.send("eth_maxPriorityFeePerGas")
        );
      } catch {
        // the max priority fee RPC call is not supported by
        // this chain
      }

      maxPriorityFeePerGas = maxPriorityFeePerGas ?? 1_000_000_000n;
      maxFeePerGas = 2n * baseFeePerGas + maxPriorityFeePerGas;
    }

    return new FeeData(gasPrice, maxFeePerGas, maxPriorityFeePerGas);
  }

  public async getBalance(
    address: AddressLike,
    blockTag?: BlockTag | undefined
  ): Promise<bigint> {
    const resolvedAddress = await this._getAddress(address);
    const resolvedBlockTag = await this._getBlockTag(blockTag);
    const rpcBlockTag = this._getRpcBlockTag(resolvedBlockTag);

    const balance = await this._hardhatProvider.send("eth_getBalance", [
      resolvedAddress,
      rpcBlockTag,
    ]);

    return BigInt(balance);
  }

  public async getTransactionCount(
    address: AddressLike,
    blockTag?: BlockTag | undefined
  ): Promise<number> {
    const resolvedAddress = await this._getAddress(address);
    const resolvedBlockTag = await this._getBlockTag(blockTag);
    const rpcBlockTag = this._getRpcBlockTag(resolvedBlockTag);

    const transactionCount = await this._hardhatProvider.send(
      "eth_getTransactionCount",
      [resolvedAddress, rpcBlockTag]
    );

    return Number(transactionCount);
  }

  public async getCode(
    address: AddressLike,
    blockTag?: BlockTag | undefined
  ): Promise<string> {
    const resolvedAddress = await this._getAddress(address);
    const resolvedBlockTag = await this._getBlockTag(blockTag);
    const rpcBlockTag = this._getRpcBlockTag(resolvedBlockTag);

    return this._hardhatProvider.send("eth_getCode", [
      resolvedAddress,
      rpcBlockTag,
    ]);
  }

  public async getStorage(
    address: AddressLike,
    position: ethers.BigNumberish,
    blockTag?: BlockTag | undefined
  ): Promise<string> {
    const resolvedAddress = await this._getAddress(address);
    const resolvedPosition = getBigInt(position, "position");
    const resolvedBlockTag = await this._getBlockTag(blockTag);
    const rpcBlockTag = this._getRpcBlockTag(resolvedBlockTag);

    return this._hardhatProvider.send("eth_getStorageAt", [
      resolvedAddress,
      `0x${resolvedPosition.toString(16)}`,
      rpcBlockTag,
    ]);
  }

  public async estimateGas(tx: TransactionRequest): Promise<bigint> {
    const blockTag =
      tx.blockTag === undefined ? "pending" : this._getBlockTag(tx.blockTag);
    const [resolvedTx, resolvedBlockTag] = await Promise.all([
      this._getTransactionRequest(tx),
      blockTag,
    ]);

    const rpcTransaction = getRpcTransaction(resolvedTx);
    const rpcBlockTag = this._getRpcBlockTag(resolvedBlockTag);

    const gasEstimation = await this._hardhatProvider.send("eth_estimateGas", [
      rpcTransaction,
      rpcBlockTag,
    ]);

    return BigInt(gasEstimation);
  }

  public async call(tx: TransactionRequest): Promise<string> {
    const [resolvedTx, resolvedBlockTag] = await Promise.all([
      this._getTransactionRequest(tx),
      this._getBlockTag(tx.blockTag),
    ]);
    const rpcTransaction = getRpcTransaction(resolvedTx);
    const rpcBlockTag = this._getRpcBlockTag(resolvedBlockTag);

    return this._hardhatProvider.send("eth_call", [
      rpcTransaction,
      rpcBlockTag,
    ]);
  }

  public async broadcastTransaction(
    signedTx: string
  ): Promise<ethers.TransactionResponse> {
    const hashPromise = this._hardhatProvider.send("eth_sendRawTransaction", [
      signedTx,
    ]);

    const [hash, blockNumber] = await Promise.all([
      hashPromise,
      this.getBlockNumber(),
    ]);

    const tx = Transaction.from(signedTx);
    if (tx.hash === null) {
      throw new HardhatEthersError(
        "Assertion error: hash of signed tx shouldn't be null"
      );
    }

    if (tx.hash !== hash) {
      throw new BroadcastedTxDifferentHash(tx.hash, hash);
    }

    return this._wrapTransactionResponse(tx as any).replaceableTransaction(
      blockNumber
    );
  }

  public async getBlock(
    blockHashOrBlockTag: BlockTag,
    prefetchTxs?: boolean | undefined
  ): Promise<ethers.Block | null> {
    const block = await this._getBlock(
      blockHashOrBlockTag,
      prefetchTxs ?? false
    );

    // eslint-disable-next-line eqeqeq
    if (block == null) {
      return null;
    }

    return this._wrapBlock(block);
  }

  public async getTransaction(
    hash: string
  ): Promise<ethers.TransactionResponse | null> {
    const transaction = await this._hardhatProvider.send(
      "eth_getTransactionByHash",
      [hash]
    );

    // eslint-disable-next-line eqeqeq
    if (transaction == null) {
      return null;
    }

    return this._wrapTransactionResponse(
      formatTransactionResponse(transaction)
    );
  }

  public async getTransactionReceipt(
    hash: string
  ): Promise<ethers.TransactionReceipt | null> {
    const receipt = await this._hardhatProvider.send(
      "eth_getTransactionReceipt",
      [hash]
    );

    // eslint-disable-next-line eqeqeq
    if (receipt == null) {
      return null;
    }

    return this._wrapTransactionReceipt(receipt);
  }

  public async getTransactionResult(_hash: string): Promise<string | null> {
    throw new NotImplementedError("HardhatEthersProvider.getTransactionResult");
  }

  public async getLogs(
    filter: Filter | FilterByBlockHash
  ): Promise<ethers.Log[]> {
    const resolvedFilter = await this._getFilter(filter);

    const logs = await this._hardhatProvider.send("eth_getLogs", [
      resolvedFilter,
    ]);

    return logs.map((l: any) => this._wrapLog(formatLog(l)));
  }

  public async resolveName(_ensName: string): Promise<string | null> {
    throw new NotImplementedError("HardhatEthersProvider.resolveName");
  }

  public async lookupAddress(_address: string): Promise<string | null> {
    throw new NotImplementedError("HardhatEthersProvider.lookupAddress");
  }

  public async waitForTransaction(
    _hash: string,
    _confirms?: number | undefined,
    _timeout?: number | undefined
  ): Promise<ethers.TransactionReceipt | null> {
    throw new NotImplementedError("HardhatEthersProvider.waitForTransaction");
  }

  public async waitForBlock(
    _blockTag?: BlockTag | undefined
  ): Promise<ethers.Block> {
    throw new NotImplementedError("HardhatEthersProvider.waitForBlock");
  }

  // -------------------------------------- //
  // event-emitter related public functions //
  // -------------------------------------- //

  public async on(
    ethersEvent: ProviderEvent,
    listener: Listener
  ): Promise<this> {
    const event = ethersToHardhatEvent(ethersEvent);

    if (event.kind === "block") {
      await this._onBlock(listener, { once: false });
    } else if (event.kind === "transactionHash") {
      await this._onTransactionHash(event.txHash, listener, { once: false });
    } else if (event.kind === "event") {
      const { eventFilter } = event;
      const blockListener = this._getBlockListenerForEvent(
        eventFilter,
        listener
      );

      this._addEventListener(eventFilter, listener, blockListener);

      await this.on("block", blockListener);
    } else {
      const _exhaustiveCheck: never = event;
    }

    return this;
  }

  public async once(
    ethersEvent: ProviderEvent,
    listener: Listener
  ): Promise<this> {
    const event = ethersToHardhatEvent(ethersEvent);

    if (event.kind === "block") {
      await this._onBlock(listener, { once: true });
    } else if (event.kind === "transactionHash") {
      await this._onTransactionHash(event.txHash, listener, { once: true });
    } else if (event.kind === "event") {
      const { eventFilter } = event;
      const blockListener = this._getBlockListenerForEvent(
        eventFilter,
        listener
      );

      this._addEventListener(eventFilter, listener, blockListener);

      await this.once("block", blockListener);
    } else {
      const _exhaustiveCheck: never = event;
    }

    return this;
  }

  public async emit(
    ethersEvent: ProviderEvent,
    ...args: any[]
  ): Promise<boolean> {
    const event = ethersToHardhatEvent(ethersEvent);

    if (event.kind === "block") {
      return this._emitBlock(...args);
    } else if (event.kind === "transactionHash") {
      return this._emitTransactionHash(event.txHash, ...args);
    } else if (event.kind === "event") {
      throw new NotImplementedError("emit(event)");
    } else {
      const _exhaustiveCheck: never = event;
      return _exhaustiveCheck;
    }
  }

  public async listenerCount(
    event?: ProviderEvent | undefined
  ): Promise<number> {
    const listeners = await this.listeners(event);

    return listeners.length;
  }

  public async listeners(
    ethersEvent?: ProviderEvent | undefined
  ): Promise<Listener[]> {
    if (ethersEvent === undefined) {
      throw new NotImplementedError("listeners()");
    }

    const event = ethersToHardhatEvent(ethersEvent);

    if (event.kind === "block") {
      return this._blockListeners.map(({ listener }) => listener);
    } else if (event.kind === "transactionHash") {
      return (
        this._transactionHashListeners
          .get(event.txHash)
          ?.map(({ listener }) => listener) ?? []
      );
    } else if (event.kind === "event") {
      const isEqual = require("lodash.isequal") as typeof LodashIsEqualT;

      const eventListener = this._eventListeners.find((item) =>
        isEqual(item.event, event)
      );
      if (eventListener === undefined) {
        return [];
      }
      return [...eventListener.listenersMap.keys()];
    } else {
      const _exhaustiveCheck: never = event;
      return _exhaustiveCheck;
    }
  }

  public async off(
    ethersEvent: ProviderEvent,
    listener?: Listener | undefined
  ): Promise<this> {
    const event = ethersToHardhatEvent(ethersEvent);

    if (event.kind === "block") {
      this._clearBlockListeners(listener);
    } else if (event.kind === "transactionHash") {
      this._clearTransactionHashListeners(event.txHash, listener);
    } else if (event.kind === "event") {
      const { eventFilter } = event;
      if (listener === undefined) {
        await this._clearEventListeners(eventFilter);
      } else {
        await this._removeEventListener(eventFilter, listener);
      }
    } else {
      const _exhaustiveCheck: never = event;
    }

    return this;
  }

  public async removeAllListeners(
    ethersEvent?: ProviderEvent | undefined
  ): Promise<this> {
    const event =
      ethersEvent !== undefined ? ethersToHardhatEvent(ethersEvent) : undefined;

    if (event === undefined || event.kind === "block") {
      this._clearBlockListeners();
    }
    if (event === undefined || event.kind === "transactionHash") {
      this._clearTransactionHashListeners(event?.txHash);
    }
    if (event === undefined || event.kind === "event") {
      await this._clearEventListeners(event?.eventFilter);
    }

    if (
      event !== undefined &&
      event.kind !== "block" &&
      event.kind !== "transactionHash" &&
      event.kind !== "event"
    ) {
      // this check is only to remember to add a proper if block
      // in this method's implementation if we add support for a
      // new kind of event
      const _exhaustiveCheck: never = event;
    }

    return this;
  }

  public async addListener(
    event: ProviderEvent,
    listener: Listener
  ): Promise<this> {
    return this.on(event, listener);
  }

  public async removeListener(
    event: ProviderEvent,
    listener: Listener
  ): Promise<this> {
    return this.off(event, listener);
  }

  public toJSON() {
    return "<EthersHardhatProvider>";
  }

  private _getAddress(address: AddressLike): string | Promise<string> {
    return resolveAddress(address, this);
  }

  private _getBlockTag(blockTag?: BlockTag): string | Promise<string> {
    // eslint-disable-next-line eqeqeq
    if (blockTag == null) {
      return "latest";
    }

    switch (blockTag) {
      case "earliest":
        return "0x0";
      case "latest":
      case "pending":
      case "safe":
      case "finalized":
        return blockTag;
    }

    if (isHexString(blockTag)) {
      if (isHexString(blockTag, 32)) {
        return blockTag;
      }
      return toQuantity(blockTag);
    }

    if (typeof blockTag === "number") {
      if (blockTag >= 0) {
        return toQuantity(blockTag);
      }
      return this.getBlockNumber().then((b) => toQuantity(b + blockTag));
    }

    if (typeof blockTag === "bigint") {
      if (blockTag >= 0n) {
        return toQuantity(blockTag);
      }
      return this.getBlockNumber().then((b) =>
        toQuantity(b + Number(blockTag))
      );
    }

    throw new HardhatEthersError(`Invalid blockTag: ${blockTag}`);
  }

  private _getTransactionRequest(
    _request: TransactionRequest
  ): PerformActionTransaction | Promise<PerformActionTransaction> {
    const request = copyRequest(_request) as PerformActionTransaction;

    const promises: Array<Promise<void>> = [];
    ["to", "from"].forEach((key) => {
      if (
        (request as any)[key] === null ||
        (request as any)[key] === undefined
      ) {
        return;
      }

      const addr = resolveAddress((request as any)[key]);
      if (isPromise(addr)) {
        promises.push(
          (async function () {
            (request as any)[key] = await addr;
          })()
        );
      } else {
        (request as any)[key] = addr;
      }
    });

    if (request.blockTag !== null && request.blockTag !== undefined) {
      const blockTag = this._getBlockTag(request.blockTag);
      if (isPromise(blockTag)) {
        promises.push(
          (async function () {
            request.blockTag = await blockTag;
          })()
        );
      } else {
        request.blockTag = blockTag;
      }
    }

    if (promises.length > 0) {
      return (async function () {
        await Promise.all(promises);
        return request;
      })();
    }

    return request;
  }

  private _wrapTransactionResponse(
    tx: TransactionResponseParams
  ): TransactionResponse {
    return new TransactionResponse(tx, this);
  }

  private async _getBlock(
    block: BlockTag | string,
    includeTransactions: boolean
  ): Promise<any> {
    if (isHexString(block, 32)) {
      return this._hardhatProvider.send("eth_getBlockByHash", [
        block,
        includeTransactions,
      ]);
    }

    let blockTag = this._getBlockTag(block);
    if (typeof blockTag !== "string") {
      blockTag = await blockTag;
    }

    return this._hardhatProvider.send("eth_getBlockByNumber", [
      blockTag,
      includeTransactions,
    ]);
  }

  private _wrapBlock(value: BlockParams): Block {
    return new Block(formatBlock(value), this);
  }

  private _wrapTransactionReceipt(
    value: TransactionReceiptParams
  ): TransactionReceipt {
    return new TransactionReceipt(formatTransactionReceipt(value), this);
  }

  private _getFilter(
    filter: Filter | FilterByBlockHash
  ): PerformActionFilter | Promise<PerformActionFilter> {
    // Create a canonical representation of the topics
    const topics = (filter.topics ?? []).map((topic) => {
      // eslint-disable-next-line eqeqeq
      if (topic == null) {
        return null;
      }
      if (Array.isArray(topic)) {
        return concisify(topic.map((t) => t.toLowerCase()));
      }
      return topic.toLowerCase();
    });

    const blockHash = "blockHash" in filter ? filter.blockHash : undefined;

    const resolve = (
      _address: string[],
      fromBlock?: string,
      toBlock?: string
    ) => {
      let resolvedAddress: undefined | string | string[];
      switch (_address.length) {
        case 0:
          break;
        case 1:
          resolvedAddress = _address[0];
          break;
        default:
          _address.sort();
          resolvedAddress = _address;
      }

      if (blockHash !== undefined) {
        // eslint-disable-next-line eqeqeq
        if (fromBlock != null || toBlock != null) {
          throw new HardhatEthersError("invalid filter");
        }
      }

      const resolvedFilter: any = {};
      if (resolvedAddress !== undefined) {
        resolvedFilter.address = resolvedAddress;
      }
      if (topics.length > 0) {
        resolvedFilter.topics = topics;
      }
      if (fromBlock !== undefined) {
        resolvedFilter.fromBlock = fromBlock;
      }
      if (toBlock !== undefined) {
        resolvedFilter.toBlock = toBlock;
      }
      if (blockHash !== undefined) {
        resolvedFilter.blockHash = blockHash;
      }

      return resolvedFilter;
    };

    // Addresses could be async (ENS names or Addressables)
    const address: Array<string | Promise<string>> = [];
    if (filter.address !== undefined) {
      if (Array.isArray(filter.address)) {
        for (const addr of filter.address) {
          address.push(this._getAddress(addr));
        }
      } else {
        address.push(this._getAddress(filter.address));
      }
    }

    let resolvedFromBlock: undefined | string | Promise<string>;
    if ("fromBlock" in filter) {
      resolvedFromBlock = this._getBlockTag(filter.fromBlock);
    }

    let resolvedToBlock: undefined | string | Promise<string>;
    if ("toBlock" in filter) {
      resolvedToBlock = this._getBlockTag(filter.toBlock);
    }

    if (
      address.filter((a) => typeof a !== "string").length > 0 ||
      // eslint-disable-next-line eqeqeq
      (resolvedFromBlock != null && typeof resolvedFromBlock !== "string") ||
      // eslint-disable-next-line eqeqeq
      (resolvedToBlock != null && typeof resolvedToBlock !== "string")
    ) {
      return Promise.all([
        Promise.all(address),
        resolvedFromBlock,
        resolvedToBlock,
      ]).then((result) => {
        return resolve(result[0], result[1], result[2]);
      });
    }

    return resolve(address as string[], resolvedFromBlock, resolvedToBlock);
  }

  private _wrapLog(value: LogParams): Log {
    return new Log(formatLog(value), this);
  }

  private _getRpcBlockTag(blockTag: string): string | { blockHash: string } {
    if (isHexString(blockTag, 32)) {
      return { blockHash: blockTag };
    }

    return blockTag;
  }

  private async _isHardhatNetwork(): Promise<boolean> {
    if (this._isHardhatNetworkCached === undefined) {
      this._isHardhatNetworkCached = false;
      try {
        await this._hardhatProvider.send("hardhat_metadata");
        this._isHardhatNetworkCached = true;
      } catch {}
    }

    return this._isHardhatNetworkCached;
  }

  // ------------------------------------- //
  // event-emitter related private helpers //
  // ------------------------------------- //

  private async _onTransactionHash(
    transactionHash: string,
    listener: Listener,
    { once }: { once: boolean }
  ): Promise<void> {
    const listeners = this._transactionHashListeners.get(transactionHash) ?? [];
    listeners.push({ listener, once });
    this._transactionHashListeners.set(transactionHash, listeners);
    await this._startTransactionHashPolling();
  }

  private _clearTransactionHashListeners(
    transactionHash?: string,
    listener?: Listener
  ): void {
    if (transactionHash === undefined) {
      this._transactionHashListeners = new Map();
    } else if (listener === undefined) {
      this._transactionHashListeners.delete(transactionHash);
    } else {
      const listeners = this._transactionHashListeners.get(transactionHash);
      if (listeners !== undefined) {
        const listenerIndex = listeners.findIndex(
          (item) => item.listener === listener
        );

        if (listenerIndex >= 0) {
          listeners.splice(listenerIndex, 1);
        }

        if (listeners.length === 0) {
          this._transactionHashListeners.delete(transactionHash);
        }
      }
    }

    if (this._transactionHashListeners.size === 0) {
      this._stopTransactionHashPolling();
    }
  }

  private async _startTransactionHashPolling() {
    await this._pollTransactionHashes();
  }

  private _stopTransactionHashPolling() {
    clearTimeout(this._transactionHashPollingTimeout);
    this._transactionHashPollingTimeout = undefined;
  }

  /**
   * Traverse all the registered transaction hashes and check if they were mined.
   *
   * This function should NOT throw.
   */
  private async _pollTransactionHashes() {
    try {
      const listenersToRemove: Array<[string, Listener]> = [];

      for (const [
        transactionHash,
        listeners,
      ] of this._transactionHashListeners.entries()) {
        const receipt = await this.getTransactionReceipt(transactionHash);

        if (receipt !== null) {
          for (const { listener, once } of listeners) {
            listener(receipt);
            if (once) {
              listenersToRemove.push([transactionHash, listener]);
            }
          }
        }
      }

      for (const [transactionHash, listener] of listenersToRemove) {
        this._clearTransactionHashListeners(transactionHash, listener);
      }
    } catch (e: any) {
      log(`Error during transaction hash polling: ${e.message}`);
    } finally {
      // it's possible that the first poll cleans all the listeners,
      // in that case we don't set the timeout
      if (this._transactionHashListeners.size > 0) {
        const _isHardhatNetwork = await this._isHardhatNetwork();
        const timeout = _isHardhatNetwork ? 50 : 500;

        clearTimeout(this._transactionHashPollingTimeout);
        this._transactionHashPollingTimeout = setTimeout(async () => {
          await this._pollTransactionHashes();
        }, timeout);
      }
    }
  }

  private async _startBlockPolling() {
    this._latestBlockNumberPolled = await this.getBlockNumber();
    await this._pollBlocks();
  }

  private _stopBlockPolling() {
    clearInterval(this._blockPollingTimeout);
    this._blockPollingTimeout = undefined;
  }

  private async _pollBlocks() {
    try {
      const currentBlockNumber = await this.getBlockNumber();
      const previousBlockNumber = this._latestBlockNumberPolled ?? 0;

      if (currentBlockNumber === previousBlockNumber) {
        // Don't do anything, there are no new blocks
        return;
      } else if (currentBlockNumber < previousBlockNumber) {
        // This can happen if there was a reset or a snapshot was reverted.
        // We don't know which number the network was reset to, so we update
        // the latest block number seen and do nothing else.
        this._latestBlockNumberPolled = currentBlockNumber;
        return;
      }

      this._latestBlockNumberPolled = currentBlockNumber;

      for (
        let blockNumber = previousBlockNumber + 1;
        blockNumber <= this._latestBlockNumberPolled;
        blockNumber++
      ) {
        const listenersToRemove: Listener[] = [];

        for (const { listener, once } of this._blockListeners) {
          listener(blockNumber);
          if (once) {
            listenersToRemove.push(listener);
          }
        }

        for (const listener of listenersToRemove) {
          this._clearBlockListeners(listener);
        }
      }
    } catch (e: any) {
      log(`Error during block polling: ${e.message}`);
    } finally {
      // it's possible that the first poll cleans all the listeners,
      // in that case we don't set the timeout
      if (this._blockListeners.length > 0) {
        const _isHardhatNetwork = await this._isHardhatNetwork();
        const timeout = _isHardhatNetwork ? 50 : 500;

        clearTimeout(this._blockPollingTimeout);
        this._blockPollingTimeout = setTimeout(async () => {
          await this._pollBlocks();
        }, timeout);
      }
    }
  }

  private _emitTransactionHash(
    transactionHash: string,
    ...args: any[]
  ): boolean {
    const listeners = this._transactionHashListeners.get(transactionHash);
    const listenersToRemove: Listener[] = [];

    if (listeners === undefined) {
      return false;
    }

    for (const { listener, once } of listeners) {
      listener(...args);
      if (once) {
        listenersToRemove.push(listener);
      }
    }

    for (const listener of listenersToRemove) {
      this._clearTransactionHashListeners(transactionHash, listener);
    }

    return true;
  }

  private _emitBlock(...args: any[]): boolean {
    const listeners = this._blockListeners;
    const listenersToRemove: Listener[] = [];

    for (const { listener, once } of listeners) {
      listener(...args);
      if (once) {
        listenersToRemove.push(listener);
      }
    }

    for (const listener of listenersToRemove) {
      this._clearBlockListeners(listener);
    }

    return true;
  }

  private async _onBlock(
    listener: Listener,
    { once }: { once: boolean }
  ): Promise<void> {
    const listeners = this._blockListeners;
    listeners.push({ listener, once });
    this._blockListeners = listeners;
    await this._startBlockPolling();
  }

  private _clearBlockListeners(listener?: Listener): void {
    if (listener === undefined) {
      this._blockListeners = [];
      this._stopBlockPolling();
    } else {
      const listenerIndex = this._blockListeners.findIndex(
        (item) => item.listener === listener
      );

      if (listenerIndex >= 0) {
        this._blockListeners.splice(listenerIndex, 1);
      }

      if (this._blockListeners.length === 0) {
        this._stopBlockPolling();
      }
    }
  }

  private _getBlockListenerForEvent(event: EventFilter, listener: Listener) {
    return async (blockNumber: number) => {
      const eventLogs = await this.getLogs({
        fromBlock: blockNumber,
        toBlock: blockNumber,
      });

      const matchingLogs = eventLogs.filter((e) => {
        if (event.address !== undefined && e.address !== event.address) {
          return false;
        }
        if (event.topics !== undefined) {
          const topicsToMatch = event.topics;
          // the array of topics to match can be smaller than the actual
          // array of topics; in that case only those first topics are
          // checked
          const topics = e.topics.slice(0, topicsToMatch.length);

          const topicsMatch = topics.every((topic, i) => {
            const topicToMatch = topicsToMatch[i];
            if (topicToMatch === null) {
              return true;
            }

            if (typeof topicToMatch === "string") {
              return topic === topicsToMatch[i];
            }

            return topicToMatch.includes(topic);
          });

          return topicsMatch;
        }

        return true;
      });

      for (const matchingLog of matchingLogs) {
        listener(matchingLog);
      }
    };
  }

  private _addEventListener(
    event: EventFilter,
    listener: Listener,
    blockListener: Listener
  ) {
    const isEqual = require("lodash.isequal") as typeof LodashIsEqualT;

    const eventListener = this._eventListeners.find((item) =>
      isEqual(item.event, event)
    );

    if (eventListener === undefined) {
      const listenersMap = new Map();
      listenersMap.set(listener, blockListener);
      this._eventListeners.push({ event, listenersMap });
    } else {
      eventListener.listenersMap.set(listener, blockListener);
    }
  }

  private async _clearEventListeners(event?: EventFilter) {
    const isEqual = require("lodash.isequal") as typeof LodashIsEqualT;

    const blockListenersToRemove: Listener[] = [];

    if (event === undefined) {
      for (const eventListener of this._eventListeners) {
        for (const blockListener of eventListener.listenersMap.values()) {
          blockListenersToRemove.push(blockListener);
        }
      }

      this._eventListeners = [];
    } else {
      const index = this._eventListeners.findIndex((item) =>
        isEqual(item.event, event)
      );
      if (index === -1) {
        const { listenersMap } = this._eventListeners[index];
        this._eventListeners.splice(index, 1);
        for (const blockListener of listenersMap.values()) {
          blockListenersToRemove.push(blockListener);
        }
      }
    }

    for (const blockListener of blockListenersToRemove) {
      await this.off("block", blockListener);
    }
  }

  private async _removeEventListener(event: EventFilter, listener: Listener) {
    const isEqual = require("lodash.isequal") as typeof LodashIsEqualT;

    const index = this._eventListeners.findIndex((item) =>
      isEqual(item.event, event)
    );
    if (index === -1) {
      // nothing to do
      return;
    }

    const { listenersMap } = this._eventListeners[index];

    const blockListener = listenersMap.get(listener);
    listenersMap.delete(listener);
    if (blockListener === undefined) {
      // nothing to do
      return;
    }

    await this.off("block", blockListener);
  }
}

function isPromise<T = any>(value: any): value is Promise<T> {
  return Boolean(value) && typeof value.then === "function";
}

function concisify(items: string[]): string[] {
  items = Array.from(new Set(items).values());
  items.sort();
  return items;
}

function isTransactionHash(x: string): boolean {
  return x.startsWith("0x") && x.length === 66;
}

function isEventFilter(x: ProviderEvent): x is EventFilter {
  if (typeof x !== "string" && !Array.isArray(x) && !("orphan" in x)) {
    return true;
  }

  return false;
}

function ethersToHardhatEvent(
  event: ProviderEvent
): HardhatEthersProviderEvent {
  if (typeof event === "string") {
    if (event === "block") {
      return { kind: "block" };
    } else if (isTransactionHash(event)) {
      return { kind: "transactionHash", txHash: event };
    } else {
      throw new UnsupportedEventError(event);
    }
  } else if (isEventFilter(event)) {
    return { kind: "event", eventFilter: event };
  } else {
    throw new UnsupportedEventError(event);
  }
}
