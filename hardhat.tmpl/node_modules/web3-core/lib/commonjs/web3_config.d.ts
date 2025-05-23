import { Numbers, HexString, BlockNumberOrTag, Common, DataFormat } from 'web3-types';
import { CustomTransactionSchema, TransactionTypeParser } from './types.js';
import { TransactionBuilder } from './web3_context.js';
import { Web3EventEmitter } from './web3_event_emitter.js';
export interface Web3ConfigOptions {
    handleRevert: boolean;
    defaultAccount?: HexString;
    defaultBlock: BlockNumberOrTag;
    transactionSendTimeout: number;
    transactionBlockTimeout: number;
    transactionConfirmationBlocks: number;
    transactionPollingInterval: number;
    transactionPollingTimeout: number;
    transactionReceiptPollingInterval?: number;
    transactionConfirmationPollingInterval?: number;
    blockHeaderTimeout: number;
    maxListenersWarningThreshold: number;
    contractDataInputFill: 'data' | 'input' | 'both';
    defaultNetworkId?: Numbers;
    defaultChain: string;
    defaultHardfork: string;
    ignoreGasPricing: boolean;
    defaultCommon?: Common;
    defaultTransactionType: Numbers;
    defaultMaxPriorityFeePerGas: Numbers;
    enableExperimentalFeatures: {
        useSubscriptionWhenCheckingBlockTimeout: boolean;
        useRpcCallSpecification: boolean;
    };
    transactionBuilder?: TransactionBuilder;
    transactionTypeParser?: TransactionTypeParser;
    customTransactionSchema?: CustomTransactionSchema;
    defaultReturnFormat: DataFormat;
}
type ConfigEvent<T, P extends keyof T = keyof T> = P extends unknown ? {
    name: P;
    oldValue: T[P];
    newValue: T[P];
} : never;
export declare enum Web3ConfigEvent {
    CONFIG_CHANGE = "CONFIG_CHANGE"
}
export declare abstract class Web3Config extends Web3EventEmitter<{
    [Web3ConfigEvent.CONFIG_CHANGE]: ConfigEvent<Web3ConfigOptions>;
}> implements Web3ConfigOptions {
    config: Web3ConfigOptions;
    constructor(options?: Partial<Web3ConfigOptions>);
    setConfig(options: Partial<Web3ConfigOptions>): void;
    /**
     * The `handleRevert` options property returns the revert reason string if enabled for the following methods:
     * - web3.eth.sendTransaction()
     * - web3.eth.call()
     * - myContract.methods.myMethod().call()
     * - myContract.methods.myMethod().send()
     * Default is `false`.
     *
     * `Note`: At the moment `handleRevert` is only supported for `sendTransaction` and not for `sendSignedTransaction`
     */
    get handleRevert(): boolean;
    /**
     * Will set the handleRevert
     */
    set handleRevert(val: boolean);
    /**
     * The `contractDataInputFill` options property will allow you to set the hash of the method signature and encoded parameters to the property
     * either `data`, `input` or both within your contract.
     * This will affect the contracts send, call and estimateGas methods
     * Default is `data`.
     */
    get contractDataInputFill(): "input" | "data" | "both";
    /**
     * Will set the contractDataInputFill
     */
    set contractDataInputFill(val: "input" | "data" | "both");
    /**
     * This default address is used as the default `from` property, if no `from` property is specified in for the following methods:
     * - web3.eth.sendTransaction()
     * - web3.eth.call()
     * - myContract.methods.myMethod().call()
     * - myContract.methods.myMethod().send()
     */
    get defaultAccount(): string | undefined;
    /**
     * Will set the default account.
     */
    set defaultAccount(val: string | undefined);
    /**
     * The default block is used for certain methods. You can override it by passing in the defaultBlock as last parameter. The default value is `"latest"`.
     * - web3.eth.getBalance()
     * - web3.eth.getCode()
     * - web3.eth.getTransactionCount()
     * - web3.eth.getStorageAt()
     * - web3.eth.call()
     * - myContract.methods.myMethod().call()
     */
    get defaultBlock(): BlockNumberOrTag;
    /**
     * Will set the default block.
     *
     * - A block number
     * - `"earliest"` - String: The genesis block
     * - `"latest"` - String: The latest block (current head of the blockchain)
     * - `"pending"` - String: The currently mined block (including pending transactions)
     * - `"finalized"` - String: (For POS networks) The finalized block is one which has been accepted as canonical by greater than 2/3 of validators
     * - `"safe"` - String: (For POS networks) The safe head block is one which under normal network conditions, is expected to be included in the canonical chain. Under normal network conditions the safe head and the actual tip of the chain will be equivalent (with safe head trailing only by a few seconds). Safe heads will be less likely to be reorged than the proof of work network's latest blocks.
     */
    set defaultBlock(val: BlockNumberOrTag);
    /**
     * The time used to wait for Ethereum Node to return the sent transaction result.
     * Note: If the RPC call stuck at the Node and therefor timed-out, the transaction may still be pending or even mined by the Network. We recommend checking the pending transactions in such a case.
     * Default is `750` seconds (12.5 minutes).
     */
    get transactionSendTimeout(): number;
    /**
     * Will set the transactionSendTimeout.
     */
    set transactionSendTimeout(val: number);
    /**
     * The `transactionBlockTimeout` is used over socket-based connections. This option defines the amount of new blocks it should wait until the first confirmation happens, otherwise the PromiEvent rejects with a timeout error.
     * Default is `50`.
     */
    get transactionBlockTimeout(): number;
    /**
     * Will set the transactionBlockTimeout.
     */
    set transactionBlockTimeout(val: number);
    /**
     * This defines the number of blocks it requires until a transaction is considered confirmed.
     * Default is `24`.
     */
    get transactionConfirmationBlocks(): number;
    /**
     * Will set the transactionConfirmationBlocks.
     */
    set transactionConfirmationBlocks(val: number);
    /**
     * Used over HTTP connections. This option defines the number of seconds between Web3 calls for a receipt which confirms that a transaction was mined by the network.
     * Default is `1000` ms.
     */
    get transactionPollingInterval(): number;
    /**
     * Will set the transactionPollingInterval.
     */
    set transactionPollingInterval(val: number);
    /**
     * Used over HTTP connections. This option defines the number of seconds Web3 will wait for a receipt which confirms that a transaction was mined by the network. Note: If this method times out, the transaction may still be pending.
     * Default is `750` seconds (12.5 minutes).
     */
    get transactionPollingTimeout(): number;
    /**
     * Will set the transactionPollingTimeout.
     */
    set transactionPollingTimeout(val: number);
    /**
     * The `transactionPollingInterval` is used over HTTP connections. This option defines the number of seconds between Web3 calls for a receipt which confirms that a transaction was mined by the network.
     * Default is `undefined`
     */
    get transactionReceiptPollingInterval(): number | undefined;
    /**
     * Will set the transactionReceiptPollingInterval
     */
    set transactionReceiptPollingInterval(val: number | undefined);
    get transactionConfirmationPollingInterval(): number | undefined;
    set transactionConfirmationPollingInterval(val: number | undefined);
    /**
     * The blockHeaderTimeout is used over socket-based connections. This option defines the amount seconds it should wait for `'newBlockHeaders'` event before falling back to polling to fetch transaction receipt.
     * Default is `10` seconds.
     */
    get blockHeaderTimeout(): number;
    /**
     * Will set the blockHeaderTimeout
     */
    set blockHeaderTimeout(val: number);
    /**
     * The enableExperimentalFeatures is used to enable trying new experimental features that are still not fully implemented or not fully tested or still have some related issues.
     * Default is `false` for every feature.
     */
    get enableExperimentalFeatures(): {
        useSubscriptionWhenCheckingBlockTimeout: boolean;
        useRpcCallSpecification: boolean;
    };
    /**
     * Will set the enableExperimentalFeatures
     */
    set enableExperimentalFeatures(val: {
        useSubscriptionWhenCheckingBlockTimeout: boolean;
        useRpcCallSpecification: boolean;
    });
    get maxListenersWarningThreshold(): number;
    set maxListenersWarningThreshold(val: number);
    get defaultReturnFormat(): DataFormat;
    set defaultReturnFormat(val: DataFormat);
    get defaultNetworkId(): Numbers | undefined;
    set defaultNetworkId(val: Numbers | undefined);
    get defaultChain(): string;
    set defaultChain(val: string);
    /**
     * Will return the default hardfork. Default is `london`
     * The default hardfork property can be one of the following:
     * - `chainstart`
     * - `homestead`
     * - `dao`
     * - `tangerineWhistle`
     * - `spuriousDragon`
     * - `byzantium`
     * - `constantinople`
     * - `petersburg`
     * - `istanbul`
     * - `berlin`
     * - `london`
     * - 'arrowGlacier',
     * - 'tangerineWhistle',
     * - 'muirGlacier'
     *
     */
    get defaultHardfork(): string;
    /**
     * Will set the default hardfork.
     *
     */
    set defaultHardfork(val: string);
    /**
     *
     * Will get the default common property
     * The default common property does contain the following Common object:
     * - `customChain` - `Object`: The custom chain properties
     * 	- `name` - `string`: (optional) The name of the chain
     * 	- `networkId` - `number`: Network ID of the custom chain
     * 	- `chainId` - `number`: Chain ID of the custom chain
     * - `baseChain` - `string`: (optional) mainnet, goerli, kovan, rinkeby, or ropsten
     * - `hardfork` - `string`: (optional) chainstart, homestead, dao, tangerineWhistle, spuriousDragon, byzantium, constantinople, petersburg, istanbul, berlin, or london
     * Default is `undefined`.
     *
     */
    get defaultCommon(): Common | undefined;
    /**
     * Will set the default common property
     *
     */
    set defaultCommon(val: Common | undefined);
    /**
     *  Will get the ignoreGasPricing property. When true, the gasPrice, maxPriorityFeePerGas, and maxFeePerGas will not be autofilled in the transaction object.
     *  Useful when you want wallets to handle gas pricing.
     */
    get ignoreGasPricing(): boolean;
    set ignoreGasPricing(val: boolean);
    get defaultTransactionType(): Numbers;
    set defaultTransactionType(val: Numbers);
    get defaultMaxPriorityFeePerGas(): Numbers;
    set defaultMaxPriorityFeePerGas(val: Numbers);
    get transactionBuilder(): TransactionBuilder | undefined;
    set transactionBuilder(val: TransactionBuilder | undefined);
    get transactionTypeParser(): TransactionTypeParser | undefined;
    set transactionTypeParser(val: TransactionTypeParser | undefined);
    get customTransactionSchema(): CustomTransactionSchema | undefined;
    set customTransactionSchema(schema: CustomTransactionSchema | undefined);
    private _triggerConfigChange;
}
export {};
