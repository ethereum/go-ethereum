import type { AddressLike, BlockTag, TransactionRequest, Filter, FilterByBlockHash, Listener, ProviderEvent } from "ethers";
import { Network as EthersNetwork, ethers } from "ethers";
import { EthereumProvider } from "hardhat/types";
import { HardhatEthersSigner } from "../signers";
export declare class HardhatEthersProvider implements ethers.Provider {
    private readonly _hardhatProvider;
    private readonly _networkName;
    private _isHardhatNetworkCached;
    private _latestBlockNumberPolled;
    private _blockListeners;
    private _transactionHashListeners;
    private _eventListeners;
    private _transactionHashPollingTimeout;
    private _blockPollingTimeout;
    constructor(_hardhatProvider: EthereumProvider, _networkName: string);
    get provider(): this;
    destroy(): void;
    send(method: string, params?: any[]): Promise<any>;
    getSigner(address?: number | string): Promise<HardhatEthersSigner>;
    getBlockNumber(): Promise<number>;
    getNetwork(): Promise<EthersNetwork>;
    getFeeData(): Promise<ethers.FeeData>;
    getBalance(address: AddressLike, blockTag?: BlockTag | undefined): Promise<bigint>;
    getTransactionCount(address: AddressLike, blockTag?: BlockTag | undefined): Promise<number>;
    getCode(address: AddressLike, blockTag?: BlockTag | undefined): Promise<string>;
    getStorage(address: AddressLike, position: ethers.BigNumberish, blockTag?: BlockTag | undefined): Promise<string>;
    estimateGas(tx: TransactionRequest): Promise<bigint>;
    call(tx: TransactionRequest): Promise<string>;
    broadcastTransaction(signedTx: string): Promise<ethers.TransactionResponse>;
    getBlock(blockHashOrBlockTag: BlockTag, prefetchTxs?: boolean | undefined): Promise<ethers.Block | null>;
    getTransaction(hash: string): Promise<ethers.TransactionResponse | null>;
    getTransactionReceipt(hash: string): Promise<ethers.TransactionReceipt | null>;
    getTransactionResult(_hash: string): Promise<string | null>;
    getLogs(filter: Filter | FilterByBlockHash): Promise<ethers.Log[]>;
    resolveName(_ensName: string): Promise<string | null>;
    lookupAddress(_address: string): Promise<string | null>;
    waitForTransaction(_hash: string, _confirms?: number | undefined, _timeout?: number | undefined): Promise<ethers.TransactionReceipt | null>;
    waitForBlock(_blockTag?: BlockTag | undefined): Promise<ethers.Block>;
    on(ethersEvent: ProviderEvent, listener: Listener): Promise<this>;
    once(ethersEvent: ProviderEvent, listener: Listener): Promise<this>;
    emit(ethersEvent: ProviderEvent, ...args: any[]): Promise<boolean>;
    listenerCount(event?: ProviderEvent | undefined): Promise<number>;
    listeners(ethersEvent?: ProviderEvent | undefined): Promise<Listener[]>;
    off(ethersEvent: ProviderEvent, listener?: Listener | undefined): Promise<this>;
    removeAllListeners(ethersEvent?: ProviderEvent | undefined): Promise<this>;
    addListener(event: ProviderEvent, listener: Listener): Promise<this>;
    removeListener(event: ProviderEvent, listener: Listener): Promise<this>;
    toJSON(): string;
    private _getAddress;
    private _getBlockTag;
    private _getTransactionRequest;
    private _wrapTransactionResponse;
    private _getBlock;
    private _wrapBlock;
    private _wrapTransactionReceipt;
    private _getFilter;
    private _wrapLog;
    private _getRpcBlockTag;
    private _isHardhatNetwork;
    private _onTransactionHash;
    private _clearTransactionHashListeners;
    private _startTransactionHashPolling;
    private _stopTransactionHashPolling;
    /**
     * Traverse all the registered transaction hashes and check if they were mined.
     *
     * This function should NOT throw.
     */
    private _pollTransactionHashes;
    private _startBlockPolling;
    private _stopBlockPolling;
    private _pollBlocks;
    private _emitTransactionHash;
    private _emitBlock;
    private _onBlock;
    private _clearBlockListeners;
    private _getBlockListenerForEvent;
    private _addEventListener;
    private _clearEventListeners;
    private _removeEventListener;
}
//# sourceMappingURL=hardhat-ethers-provider.d.ts.map