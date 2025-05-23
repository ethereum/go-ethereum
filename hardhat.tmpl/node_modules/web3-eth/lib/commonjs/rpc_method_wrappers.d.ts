import { FormatType, DataFormat, EthExecutionAPI, SignedTransactionInfoAPI, Address, BlockNumberOrTag, Bytes, Filter, HexString, Numbers, HexStringBytes, TransactionReceipt, Transaction, TransactionCall, Web3EthExecutionAPI, TransactionWithFromLocalWalletIndex, TransactionWithToLocalWalletIndex, TransactionWithFromAndToLocalWalletIndex, TransactionForAccessList, Eip712TypedData } from 'web3-types';
import { Web3Context, Web3PromiEvent } from 'web3-core';
import { SendSignedTransactionEvents, SendSignedTransactionOptions, SendTransactionEvents, SendTransactionOptions, TransactionMiddleware } from './types.js';
/**
 * View additional documentations here: {@link Web3Eth.getProtocolVersion}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare const getProtocolVersion: (web3Context: Web3Context<EthExecutionAPI>) => Promise<string>;
/**
 * View additional documentations here: {@link Web3Eth.isSyncing}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare const isSyncing: (web3Context: Web3Context<EthExecutionAPI>) => Promise<import("web3-types").SyncingStatusAPI>;
/**
 * View additional documentations here: {@link Web3Eth.getCoinbase}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare const getCoinbase: (web3Context: Web3Context<EthExecutionAPI>) => Promise<string>;
/**
 * View additional documentations here: {@link Web3Eth.isMining}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare const isMining: (web3Context: Web3Context<EthExecutionAPI>) => Promise<boolean>;
/**
 * View additional documentations here: {@link Web3Eth.getHashRate}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getHashRate<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.getGasPrice}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getGasPrice<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.getMaxPriorityFeePerGas}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getMaxPriorityFeePerGas<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.getBlockNumber}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getBlockNumber<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.getBalance}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getBalance<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, address: Address, blockNumber: BlockNumberOrTag | undefined, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.getStorageAt}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getStorageAt<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, address: Address, storageSlot: Numbers, blockNumber: BlockNumberOrTag | undefined, returnFormat: ReturnFormat): Promise<import("web3-types").ByteTypes[ReturnFormat["bytes"]]>;
/**
 * View additional documentations here: {@link Web3Eth.getCode}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getCode<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, address: Address, blockNumber: BlockNumberOrTag | undefined, returnFormat: ReturnFormat): Promise<import("web3-types").ByteTypes[ReturnFormat["bytes"]]>;
/**
 * View additional documentations here: {@link Web3Eth.getBlock}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getBlock<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, block: (Bytes | BlockNumberOrTag) | undefined, hydrated: boolean | undefined, returnFormat: ReturnFormat): Promise<{
    transactions: string[] | {
        readonly blockHash?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
        readonly blockNumber?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        readonly from: Address;
        readonly hash: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
        readonly transactionIndex?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        to?: (Address | null) | undefined;
        value?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        accessList?: {
            readonly address?: Address | undefined;
            readonly storageKeys?: import("web3-types").HexString32Bytes[] | undefined;
        }[] | undefined;
        common?: {
            customChain: {
                name?: string | undefined;
                networkId: import("web3-types").NumberTypes[ReturnFormat["number"]];
                chainId: import("web3-types").NumberTypes[ReturnFormat["number"]];
            };
            baseChain?: import("web3-types").ValidChains | undefined;
            hardfork?: import("web3-types").Hardfork | undefined;
        } | undefined;
        gas?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        gasPrice?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        type?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        maxFeePerGas?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        maxPriorityFeePerGas?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        data?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
        input?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
        nonce?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        chain?: import("web3-types").ValidChains | undefined;
        hardfork?: import("web3-types").Hardfork | undefined;
        chainId?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        networkId?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        gasLimit?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        yParity?: import("web3-types").Uint | undefined;
        v?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        r?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
        s?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    }[];
    parentHash: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    sha3Uncles: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    miner: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    stateRoot: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    transactionsRoot: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    receiptsRoot: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    logsBloom?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    difficulty?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    number: import("web3-types").NumberTypes[ReturnFormat["number"]];
    gasLimit: import("web3-types").NumberTypes[ReturnFormat["number"]];
    gasUsed: import("web3-types").NumberTypes[ReturnFormat["number"]];
    timestamp: import("web3-types").NumberTypes[ReturnFormat["number"]];
    extraData: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    mixHash: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    nonce: import("web3-types").NumberTypes[ReturnFormat["number"]];
    totalDifficulty: import("web3-types").NumberTypes[ReturnFormat["number"]];
    baseFeePerGas?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    size: import("web3-types").NumberTypes[ReturnFormat["number"]];
    uncles: string[];
    hash?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
}>;
/**
 * View additional documentations here: {@link Web3Eth.getBlockTransactionCount}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getBlockTransactionCount<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, block: (Bytes | BlockNumberOrTag) | undefined, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.getBlockUncleCount}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getBlockUncleCount<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, block: (Bytes | BlockNumberOrTag) | undefined, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.getUncle}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getUncle<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, block: (Bytes | BlockNumberOrTag) | undefined, uncleIndex: Numbers, returnFormat: ReturnFormat): Promise<{
    readonly parentHash: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly sha3Uncles: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly miner: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly stateRoot: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly transactionsRoot: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly receiptsRoot: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly logsBloom?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    readonly difficulty?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    readonly number: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly gasLimit: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly gasUsed: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly timestamp: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly extraData: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly mixHash: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly nonce: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly totalDifficulty: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly baseFeePerGas?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    readonly size: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly transactions: string[] | {
        readonly blockHash?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
        readonly blockNumber?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        readonly from: Address;
        readonly hash: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
        readonly transactionIndex?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        to?: (Address | null) | undefined;
        value?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        accessList?: {
            readonly address?: Address | undefined;
            readonly storageKeys?: import("web3-types").HexString32Bytes[] | undefined;
        }[] | undefined;
        common?: {
            customChain: {
                name?: string | undefined;
                networkId: import("web3-types").NumberTypes[ReturnFormat["number"]];
                chainId: import("web3-types").NumberTypes[ReturnFormat["number"]];
            };
            baseChain?: import("web3-types").ValidChains | undefined;
            hardfork?: import("web3-types").Hardfork | undefined;
        } | undefined;
        gas?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        gasPrice?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        type?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        maxFeePerGas?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        maxPriorityFeePerGas?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        data?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
        input?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
        nonce?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        chain?: import("web3-types").ValidChains | undefined;
        hardfork?: import("web3-types").Hardfork | undefined;
        chainId?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        networkId?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        gasLimit?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        yParity?: import("web3-types").Uint | undefined;
        v?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        r?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
        s?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    }[];
    readonly uncles: string[];
    readonly hash?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
}>;
/**
 * View additional documentations here: {@link Web3Eth.getTransaction}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getTransaction<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, transactionHash: Bytes, returnFormat?: ReturnFormat): Promise<{
    readonly yParity: import("web3-types").Uint;
    readonly r: import("web3-types").Uint;
    readonly s: import("web3-types").Uint;
    readonly v?: never | undefined;
    readonly maxFeePerGas: import("web3-types").Uint;
    readonly maxPriorityFeePerGas: import("web3-types").Uint;
    readonly accessList: {
        readonly address?: Address | undefined;
        readonly storageKeys?: import("web3-types").HexString32Bytes[] | undefined;
    }[];
    readonly gasPrice: import("web3-types").Uint;
    readonly to?: (Address | null) | undefined;
    readonly type: import("web3-types").HexStringSingleByte;
    readonly nonce: import("web3-types").Uint;
    readonly gas: import("web3-types").Uint;
    readonly value: import("web3-types").Uint;
    readonly input: HexStringBytes;
    readonly data?: HexStringBytes | undefined;
    readonly chainId?: import("web3-types").Uint | undefined;
    readonly hash: import("web3-types").HexString32Bytes;
    readonly blockHash?: import("web3-types").HexString32Bytes | undefined;
    readonly blockNumber?: import("web3-types").Uint | undefined;
    readonly from: Address;
    readonly transactionIndex?: import("web3-types").Uint | undefined;
} | {
    readonly yParity: import("web3-types").Uint;
    readonly r: import("web3-types").Uint;
    readonly s: import("web3-types").Uint;
    readonly v?: never | undefined;
    readonly gasPrice: import("web3-types").Uint;
    readonly accessList: {
        readonly address?: Address | undefined;
        readonly storageKeys?: import("web3-types").HexString32Bytes[] | undefined;
    }[];
    readonly maxFeePerGas?: never | undefined;
    readonly maxPriorityFeePerGas?: never | undefined;
    readonly to?: (Address | null) | undefined;
    readonly type: import("web3-types").HexStringSingleByte;
    readonly nonce: import("web3-types").Uint;
    readonly gas: import("web3-types").Uint;
    readonly value: import("web3-types").Uint;
    readonly input: HexStringBytes;
    readonly data?: HexStringBytes | undefined;
    readonly chainId?: import("web3-types").Uint | undefined;
    readonly hash: import("web3-types").HexString32Bytes;
    readonly blockHash?: import("web3-types").HexString32Bytes | undefined;
    readonly blockNumber?: import("web3-types").Uint | undefined;
    readonly from: Address;
    readonly transactionIndex?: import("web3-types").Uint | undefined;
} | {
    readonly v: import("web3-types").Uint;
    readonly r: import("web3-types").Uint;
    readonly s: import("web3-types").Uint;
    readonly gasPrice: import("web3-types").Uint;
    readonly accessList?: never | undefined;
    readonly maxFeePerGas?: never | undefined;
    readonly maxPriorityFeePerGas?: never | undefined;
    readonly to?: (Address | null) | undefined;
    readonly type: import("web3-types").HexStringSingleByte;
    readonly nonce: import("web3-types").Uint;
    readonly gas: import("web3-types").Uint;
    readonly value: import("web3-types").Uint;
    readonly input: HexStringBytes;
    readonly data?: HexStringBytes | undefined;
    readonly chainId?: import("web3-types").Uint | undefined;
    readonly hash: import("web3-types").HexString32Bytes;
    readonly blockHash?: import("web3-types").HexString32Bytes | undefined;
    readonly blockNumber?: import("web3-types").Uint | undefined;
    readonly from: Address;
    readonly transactionIndex?: import("web3-types").Uint | undefined;
} | undefined>;
/**
 * View additional documentations here: {@link Web3Eth.getPendingTransactions}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getPendingTransactions<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, returnFormat: ReturnFormat): Promise<{
    from?: Address | undefined;
    to?: (Address | null) | undefined;
    value?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    accessList?: {
        readonly address?: Address | undefined;
        readonly storageKeys?: import("web3-types").HexString32Bytes[] | undefined;
    }[] | undefined;
    common?: {
        customChain: {
            name?: string | undefined;
            networkId: import("web3-types").NumberTypes[ReturnFormat["number"]];
            chainId: import("web3-types").NumberTypes[ReturnFormat["number"]];
        };
        baseChain?: import("web3-types").ValidChains | undefined;
        hardfork?: import("web3-types").Hardfork | undefined;
    } | undefined;
    gas?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    gasPrice?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    type?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    maxFeePerGas?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    maxPriorityFeePerGas?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    data?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    input?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    nonce?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    chain?: import("web3-types").ValidChains | undefined;
    hardfork?: import("web3-types").Hardfork | undefined;
    chainId?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    networkId?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    gasLimit?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    yParity?: import("web3-types").Uint | undefined;
    v?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    r?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    s?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
}[]>;
/**
 * View additional documentations here: {@link Web3Eth.getTransactionFromBlock}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getTransactionFromBlock<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, block: (Bytes | BlockNumberOrTag) | undefined, transactionIndex: Numbers, returnFormat: ReturnFormat): Promise<{
    readonly yParity: import("web3-types").Uint;
    readonly r: import("web3-types").Uint;
    readonly s: import("web3-types").Uint;
    readonly v?: never | undefined;
    readonly maxFeePerGas: import("web3-types").Uint;
    readonly maxPriorityFeePerGas: import("web3-types").Uint;
    readonly accessList: {
        readonly address?: Address | undefined;
        readonly storageKeys?: import("web3-types").HexString32Bytes[] | undefined;
    }[];
    readonly gasPrice: import("web3-types").Uint;
    readonly to?: (Address | null) | undefined;
    readonly type: import("web3-types").HexStringSingleByte;
    readonly nonce: import("web3-types").Uint;
    readonly gas: import("web3-types").Uint;
    readonly value: import("web3-types").Uint;
    readonly input: HexStringBytes;
    readonly data?: HexStringBytes | undefined;
    readonly chainId?: import("web3-types").Uint | undefined;
    readonly hash: import("web3-types").HexString32Bytes;
    readonly blockHash?: import("web3-types").HexString32Bytes | undefined;
    readonly blockNumber?: import("web3-types").Uint | undefined;
    readonly from: Address;
    readonly transactionIndex?: import("web3-types").Uint | undefined;
} | {
    readonly yParity: import("web3-types").Uint;
    readonly r: import("web3-types").Uint;
    readonly s: import("web3-types").Uint;
    readonly v?: never | undefined;
    readonly gasPrice: import("web3-types").Uint;
    readonly accessList: {
        readonly address?: Address | undefined;
        readonly storageKeys?: import("web3-types").HexString32Bytes[] | undefined;
    }[];
    readonly maxFeePerGas?: never | undefined;
    readonly maxPriorityFeePerGas?: never | undefined;
    readonly to?: (Address | null) | undefined;
    readonly type: import("web3-types").HexStringSingleByte;
    readonly nonce: import("web3-types").Uint;
    readonly gas: import("web3-types").Uint;
    readonly value: import("web3-types").Uint;
    readonly input: HexStringBytes;
    readonly data?: HexStringBytes | undefined;
    readonly chainId?: import("web3-types").Uint | undefined;
    readonly hash: import("web3-types").HexString32Bytes;
    readonly blockHash?: import("web3-types").HexString32Bytes | undefined;
    readonly blockNumber?: import("web3-types").Uint | undefined;
    readonly from: Address;
    readonly transactionIndex?: import("web3-types").Uint | undefined;
} | {
    readonly v: import("web3-types").Uint;
    readonly r: import("web3-types").Uint;
    readonly s: import("web3-types").Uint;
    readonly gasPrice: import("web3-types").Uint;
    readonly accessList?: never | undefined;
    readonly maxFeePerGas?: never | undefined;
    readonly maxPriorityFeePerGas?: never | undefined;
    readonly to?: (Address | null) | undefined;
    readonly type: import("web3-types").HexStringSingleByte;
    readonly nonce: import("web3-types").Uint;
    readonly gas: import("web3-types").Uint;
    readonly value: import("web3-types").Uint;
    readonly input: HexStringBytes;
    readonly data?: HexStringBytes | undefined;
    readonly chainId?: import("web3-types").Uint | undefined;
    readonly hash: import("web3-types").HexString32Bytes;
    readonly blockHash?: import("web3-types").HexString32Bytes | undefined;
    readonly blockNumber?: import("web3-types").Uint | undefined;
    readonly from: Address;
    readonly transactionIndex?: import("web3-types").Uint | undefined;
} | undefined>;
/**
 * View additional documentations here: {@link Web3Eth.getTransactionReceipt}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getTransactionReceipt<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, transactionHash: Bytes, returnFormat: ReturnFormat): Promise<{
    readonly transactionHash: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly transactionIndex: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly blockHash: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly blockNumber: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly from: Address;
    readonly to: Address;
    readonly cumulativeGasUsed: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly gasUsed: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly effectiveGasPrice?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    readonly contractAddress?: Address | undefined;
    readonly logs: {
        readonly id?: string | undefined;
        readonly removed?: boolean | undefined;
        readonly logIndex?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        readonly transactionIndex?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        readonly transactionHash?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
        readonly blockHash?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
        readonly blockNumber?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
        readonly address?: Address | undefined;
        readonly data?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
        readonly topics?: import("web3-types").ByteTypes[ReturnFormat["bytes"]][] | undefined;
    }[];
    readonly logsBloom: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly root: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly status: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly type?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    events?: {
        [x: string]: {
            readonly event: string;
            readonly id?: string | undefined;
            readonly logIndex?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
            readonly transactionIndex?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
            readonly transactionHash?: import("web3-types").HexString32Bytes | undefined;
            readonly blockHash?: import("web3-types").HexString32Bytes | undefined;
            readonly blockNumber?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
            readonly address: string;
            readonly topics: HexString[];
            readonly data: HexString;
            readonly raw?: {
                data: string;
                topics: unknown[];
            } | undefined;
            readonly returnValues: {
                [x: string]: unknown;
            };
            readonly signature?: HexString | undefined;
        };
    } | undefined;
} | undefined>;
/**
 * View additional documentations here: {@link Web3Eth.getTransactionCount}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getTransactionCount<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, address: Address, blockNumber: BlockNumberOrTag | undefined, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.sendTransaction}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function sendTransaction<ReturnFormat extends DataFormat, ResolveType = FormatType<TransactionReceipt, ReturnFormat>>(web3Context: Web3Context<EthExecutionAPI>, transactionObj: Transaction | TransactionWithFromLocalWalletIndex | TransactionWithToLocalWalletIndex | TransactionWithFromAndToLocalWalletIndex, returnFormat: ReturnFormat, options?: SendTransactionOptions<ResolveType>, transactionMiddleware?: TransactionMiddleware): Web3PromiEvent<ResolveType, SendTransactionEvents<ReturnFormat>>;
/**
 * View additional documentations here: {@link Web3Eth.sendSignedTransaction}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function sendSignedTransaction<ReturnFormat extends DataFormat, ResolveType = FormatType<TransactionReceipt, ReturnFormat>>(web3Context: Web3Context<EthExecutionAPI>, signedTransaction: Bytes, returnFormat: ReturnFormat, options?: SendSignedTransactionOptions<ResolveType>): Web3PromiEvent<ResolveType, SendSignedTransactionEvents<ReturnFormat>>;
/**
 * View additional documentations here: {@link Web3Eth.sign}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function sign<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, message: Bytes, addressOrIndex: Address | number, returnFormat?: ReturnFormat): Promise<{
    messageHash: string;
    r: string;
    s: string;
    v: string;
    message?: string | undefined;
    signature: string;
} | import("web3-types").ByteTypes[ReturnFormat["bytes"]]>;
/**
 * View additional documentations here: {@link Web3Eth.signTransaction}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function signTransaction<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, transaction: Transaction, returnFormat?: ReturnFormat): Promise<SignedTransactionInfoAPI>;
/**
 * View additional documentations here: {@link Web3Eth.call}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function call<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, transaction: TransactionCall, blockNumber?: BlockNumberOrTag, returnFormat?: ReturnFormat): Promise<import("web3-types").ByteTypes[ReturnFormat["bytes"]]>;
/**
 * View additional documentations here: {@link Web3Eth.estimateGas}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function estimateGas<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, transaction: Transaction, blockNumber: BlockNumberOrTag | undefined, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.getPastLogs}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getLogs<ReturnFormat extends DataFormat>(web3Context: Web3Context<Web3EthExecutionAPI>, filter: Filter, returnFormat: ReturnFormat): Promise<(string | {
    readonly id?: string | undefined;
    readonly removed?: boolean | undefined;
    readonly logIndex?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    readonly transactionIndex?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    readonly transactionHash?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    readonly blockHash?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    readonly blockNumber?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    readonly address?: Address | undefined;
    readonly data?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    readonly topics?: import("web3-types").ByteTypes[ReturnFormat["bytes"]][] | undefined;
})[]>;
/**
 * View additional documentations here: {@link Web3Eth.getChainId}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getChainId<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.getProof}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getProof<ReturnFormat extends DataFormat>(web3Context: Web3Context<Web3EthExecutionAPI>, address: Address, storageKeys: Bytes[], blockNumber: BlockNumberOrTag | undefined, returnFormat: ReturnFormat): Promise<{
    readonly balance: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly codeHash: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly nonce: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly storageHash: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
    readonly accountProof: import("web3-types").ByteTypes[ReturnFormat["bytes"]][];
    readonly storageProof: {
        readonly key: import("web3-types").ByteTypes[ReturnFormat["bytes"]];
        readonly value: import("web3-types").NumberTypes[ReturnFormat["number"]];
        readonly proof: import("web3-types").ByteTypes[ReturnFormat["bytes"]][];
    }[];
}>;
/**
 * View additional documentations here: {@link Web3Eth.getFeeHistory}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function getFeeHistory<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, blockCount: Numbers, newestBlock: BlockNumberOrTag | undefined, rewardPercentiles: Numbers[], returnFormat: ReturnFormat): Promise<{
    readonly oldestBlock: import("web3-types").NumberTypes[ReturnFormat["number"]];
    readonly baseFeePerGas: import("web3-types").NumberTypes[ReturnFormat["number"]][];
    readonly reward: import("web3-types").NumberTypes[ReturnFormat["number"]][][];
    readonly gasUsedRatio: import("web3-types").NumberTypes[ReturnFormat["number"]][];
}>;
/**
 * View additional documentations here: {@link Web3Eth.createAccessList}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function createAccessList<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, transaction: TransactionForAccessList, blockNumber: BlockNumberOrTag | undefined, returnFormat: ReturnFormat): Promise<{
    readonly accessList?: {
        readonly address?: Address | undefined;
        readonly storageKeys?: import("web3-types").HexString32Bytes[] | undefined;
    }[] | undefined;
    readonly gasUsed?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
}>;
/**
 * View additional documentations here: {@link Web3Eth.signTypedData}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export declare function signTypedData<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, address: Address, typedData: Eip712TypedData, useLegacy: boolean, returnFormat: ReturnFormat): Promise<string>;
