/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/

import {
	Numbers,
	HexString,
	BlockNumberOrTag,
	Common,
	DEFAULT_RETURN_FORMAT,
	DataFormat,
} from 'web3-types';
import { ConfigHardforkMismatchError, ConfigChainMismatchError } from 'web3-errors';
import { isNullish, toHex } from 'web3-utils';
import { CustomTransactionSchema, TransactionTypeParser } from './types.js';
// eslint-disable-next-line import/no-cycle
import { TransactionBuilder } from './web3_context.js';
import { Web3EventEmitter } from './web3_event_emitter.js';

// To avoid cycle dependency declare this
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
		useRpcCallSpecification: boolean; // EIP-1474 https://eips.ethereum.org/EIPS/eip-1474
		// other experimental features...
	};
	transactionBuilder?: TransactionBuilder;
	transactionTypeParser?: TransactionTypeParser;
	customTransactionSchema?: CustomTransactionSchema;
	defaultReturnFormat: DataFormat;
}

type ConfigEvent<T, P extends keyof T = keyof T> = P extends unknown
	? { name: P; oldValue: T[P]; newValue: T[P] }
	: never;

export enum Web3ConfigEvent {
	CONFIG_CHANGE = 'CONFIG_CHANGE',
}

export abstract class Web3Config
	extends Web3EventEmitter<{ [Web3ConfigEvent.CONFIG_CHANGE]: ConfigEvent<Web3ConfigOptions> }>
	implements Web3ConfigOptions
{
	public config: Web3ConfigOptions = {
		handleRevert: false,
		defaultAccount: undefined,
		defaultBlock: 'latest',
		transactionBlockTimeout: 50,
		transactionConfirmationBlocks: 24,
		transactionPollingInterval: 1000,
		transactionPollingTimeout: 750 * 1000,
		transactionReceiptPollingInterval: undefined,
		transactionSendTimeout: 750 * 1000,
		transactionConfirmationPollingInterval: undefined,
		blockHeaderTimeout: 10,
		maxListenersWarningThreshold: 100,
		contractDataInputFill: 'data',
		defaultNetworkId: undefined,
		defaultChain: 'mainnet',
		defaultHardfork: 'london',
		// TODO - Check if there is a default Common
		defaultCommon: undefined,
		defaultTransactionType: '0x2',
		defaultMaxPriorityFeePerGas: toHex(2500000000),
		enableExperimentalFeatures: {
			useSubscriptionWhenCheckingBlockTimeout: false,
			useRpcCallSpecification: false,
		},
		transactionBuilder: undefined,
		transactionTypeParser: undefined,
		customTransactionSchema: undefined,
		defaultReturnFormat: DEFAULT_RETURN_FORMAT,
		ignoreGasPricing: false,
	};

	public constructor(options?: Partial<Web3ConfigOptions>) {
		super();
		this.setConfig(options ?? {});
	}

	public setConfig(options: Partial<Web3ConfigOptions>) {
		// TODO: Improve and add key check
		const keys = Object.keys(options) as (keyof Web3ConfigOptions)[];
		for (const key of keys) {
			this._triggerConfigChange(key, options[key]);

			if (
				!isNullish(options[key]) &&
				typeof options[key] === 'number' &&
				key === 'maxListenersWarningThreshold'
			) {
				// additionally set in event emitter
				this.setMaxListenerWarningThreshold(Number(options[key]));
			}
		}
		Object.assign(this.config, options);
	}

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
	public get handleRevert() {
		return this.config.handleRevert;
	}

	/**
	 * Will set the handleRevert
	 */
	public set handleRevert(val) {
		this._triggerConfigChange('handleRevert', val);
		this.config.handleRevert = val;
	}

	/**
	 * The `contractDataInputFill` options property will allow you to set the hash of the method signature and encoded parameters to the property
	 * either `data`, `input` or both within your contract.
	 * This will affect the contracts send, call and estimateGas methods
	 * Default is `data`.
	 */
	public get contractDataInputFill() {
		return this.config.contractDataInputFill;
	}

	/**
	 * Will set the contractDataInputFill
	 */
	public set contractDataInputFill(val) {
		this._triggerConfigChange('contractDataInputFill', val);
		this.config.contractDataInputFill = val;
	}

	/**
	 * This default address is used as the default `from` property, if no `from` property is specified in for the following methods:
	 * - web3.eth.sendTransaction()
	 * - web3.eth.call()
	 * - myContract.methods.myMethod().call()
	 * - myContract.methods.myMethod().send()
	 */
	public get defaultAccount() {
		return this.config.defaultAccount;
	}
	/**
	 * Will set the default account.
	 */
	public set defaultAccount(val) {
		this._triggerConfigChange('defaultAccount', val);
		this.config.defaultAccount = val;
	}

	/**
	 * The default block is used for certain methods. You can override it by passing in the defaultBlock as last parameter. The default value is `"latest"`.
	 * - web3.eth.getBalance()
	 * - web3.eth.getCode()
	 * - web3.eth.getTransactionCount()
	 * - web3.eth.getStorageAt()
	 * - web3.eth.call()
	 * - myContract.methods.myMethod().call()
	 */
	public get defaultBlock() {
		return this.config.defaultBlock;
	}

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
	public set defaultBlock(val) {
		this._triggerConfigChange('defaultBlock', val);
		this.config.defaultBlock = val;
	}

	/**
	 * The time used to wait for Ethereum Node to return the sent transaction result.
	 * Note: If the RPC call stuck at the Node and therefor timed-out, the transaction may still be pending or even mined by the Network. We recommend checking the pending transactions in such a case.
	 * Default is `750` seconds (12.5 minutes).
	 */
	public get transactionSendTimeout() {
		return this.config.transactionSendTimeout;
	}

	/**
	 * Will set the transactionSendTimeout.
	 */
	public set transactionSendTimeout(val) {
		this._triggerConfigChange('transactionSendTimeout', val);
		this.config.transactionSendTimeout = val;
	}

	/**
	 * The `transactionBlockTimeout` is used over socket-based connections. This option defines the amount of new blocks it should wait until the first confirmation happens, otherwise the PromiEvent rejects with a timeout error.
	 * Default is `50`.
	 */
	public get transactionBlockTimeout() {
		return this.config.transactionBlockTimeout;
	}

	/**
	 * Will set the transactionBlockTimeout.
	 */
	public set transactionBlockTimeout(val) {
		this._triggerConfigChange('transactionBlockTimeout', val);
		this.config.transactionBlockTimeout = val;
	}

	/**
	 * This defines the number of blocks it requires until a transaction is considered confirmed.
	 * Default is `24`.
	 */
	public get transactionConfirmationBlocks() {
		return this.config.transactionConfirmationBlocks;
	}

	/**
	 * Will set the transactionConfirmationBlocks.
	 */
	public set transactionConfirmationBlocks(val) {
		this._triggerConfigChange('transactionConfirmationBlocks', val);
		this.config.transactionConfirmationBlocks = val;
	}

	/**
	 * Used over HTTP connections. This option defines the number of seconds between Web3 calls for a receipt which confirms that a transaction was mined by the network.
	 * Default is `1000` ms.
	 */
	public get transactionPollingInterval() {
		return this.config.transactionPollingInterval;
	}

	/**
	 * Will set the transactionPollingInterval.
	 */
	public set transactionPollingInterval(val) {
		this._triggerConfigChange('transactionPollingInterval', val);
		this.config.transactionPollingInterval = val;

		this.transactionReceiptPollingInterval = val;
		this.transactionConfirmationPollingInterval = val;
	}
	/**
	 * Used over HTTP connections. This option defines the number of seconds Web3 will wait for a receipt which confirms that a transaction was mined by the network. Note: If this method times out, the transaction may still be pending.
	 * Default is `750` seconds (12.5 minutes).
	 */
	public get transactionPollingTimeout() {
		return this.config.transactionPollingTimeout;
	}

	/**
	 * Will set the transactionPollingTimeout.
	 */
	public set transactionPollingTimeout(val) {
		this._triggerConfigChange('transactionPollingTimeout', val);

		this.config.transactionPollingTimeout = val;
	}

	/**
	 * The `transactionPollingInterval` is used over HTTP connections. This option defines the number of seconds between Web3 calls for a receipt which confirms that a transaction was mined by the network.
	 * Default is `undefined`
	 */
	public get transactionReceiptPollingInterval() {
		return this.config.transactionReceiptPollingInterval;
	}

	/**
	 * Will set the transactionReceiptPollingInterval
	 */
	public set transactionReceiptPollingInterval(val) {
		this._triggerConfigChange('transactionReceiptPollingInterval', val);
		this.config.transactionReceiptPollingInterval = val;
	}

	public get transactionConfirmationPollingInterval() {
		return this.config.transactionConfirmationPollingInterval;
	}

	public set transactionConfirmationPollingInterval(val) {
		this._triggerConfigChange('transactionConfirmationPollingInterval', val);
		this.config.transactionConfirmationPollingInterval = val;
	}

	/**
	 * The blockHeaderTimeout is used over socket-based connections. This option defines the amount seconds it should wait for `'newBlockHeaders'` event before falling back to polling to fetch transaction receipt.
	 * Default is `10` seconds.
	 */
	public get blockHeaderTimeout() {
		return this.config.blockHeaderTimeout;
	}

	/**
	 * Will set the blockHeaderTimeout
	 */
	public set blockHeaderTimeout(val) {
		this._triggerConfigChange('blockHeaderTimeout', val);

		this.config.blockHeaderTimeout = val;
	}

	/**
	 * The enableExperimentalFeatures is used to enable trying new experimental features that are still not fully implemented or not fully tested or still have some related issues.
	 * Default is `false` for every feature.
	 */
	public get enableExperimentalFeatures() {
		return this.config.enableExperimentalFeatures;
	}

	/**
	 * Will set the enableExperimentalFeatures
	 */
	public set enableExperimentalFeatures(val) {
		this._triggerConfigChange('enableExperimentalFeatures', val);

		this.config.enableExperimentalFeatures = val;
	}

	public get maxListenersWarningThreshold() {
		return this.config.maxListenersWarningThreshold;
	}

	public set maxListenersWarningThreshold(val) {
		this._triggerConfigChange('maxListenersWarningThreshold', val);
		this.setMaxListenerWarningThreshold(val);
		this.config.maxListenersWarningThreshold = val;
	}

	public get defaultReturnFormat() {
		return this.config.defaultReturnFormat;
	}
	public set defaultReturnFormat(val) {
		this._triggerConfigChange('defaultReturnFormat', val);

		this.config.defaultReturnFormat = val;
	}

	public get defaultNetworkId() {
		return this.config.defaultNetworkId;
	}

	public set defaultNetworkId(val) {
		this._triggerConfigChange('defaultNetworkId', val);

		this.config.defaultNetworkId = val;
	}

	public get defaultChain() {
		return this.config.defaultChain;
	}

	public set defaultChain(val) {
		if (
			!isNullish(this.config.defaultCommon) &&
			!isNullish(this.config.defaultCommon.baseChain) &&
			val !== this.config.defaultCommon.baseChain
		)
			throw new ConfigChainMismatchError(this.config.defaultChain, val);

		this._triggerConfigChange('defaultChain', val);

		this.config.defaultChain = val;
	}

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
	public get defaultHardfork() {
		return this.config.defaultHardfork;
	}

	/**
	 * Will set the default hardfork.
	 *
	 */
	public set defaultHardfork(val) {
		if (
			!isNullish(this.config.defaultCommon) &&
			!isNullish(this.config.defaultCommon.hardfork) &&
			val !== this.config.defaultCommon.hardfork
		)
			throw new ConfigHardforkMismatchError(this.config.defaultCommon.hardfork, val);
		this._triggerConfigChange('defaultHardfork', val);

		this.config.defaultHardfork = val;
	}

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
	public get defaultCommon() {
		return this.config.defaultCommon;
	}

	/**
	 * Will set the default common property
	 *
	 */
	public set defaultCommon(val: Common | undefined) {
		// validation check if default hardfork is set and matches defaultCommon hardfork
		if (
			!isNullish(this.config.defaultHardfork) &&
			!isNullish(val) &&
			!isNullish(val.hardfork) &&
			this.config.defaultHardfork !== val.hardfork
		)
			throw new ConfigHardforkMismatchError(this.config.defaultHardfork, val.hardfork);
		if (
			!isNullish(this.config.defaultChain) &&
			!isNullish(val) &&
			!isNullish(val.baseChain) &&
			this.config.defaultChain !== val.baseChain
		)
			throw new ConfigChainMismatchError(this.config.defaultChain, val.baseChain);
		this._triggerConfigChange('defaultCommon', val);

		this.config.defaultCommon = val;
	}

	/**
	 *  Will get the ignoreGasPricing property. When true, the gasPrice, maxPriorityFeePerGas, and maxFeePerGas will not be autofilled in the transaction object.
	 *  Useful when you want wallets to handle gas pricing.
	 */
	public get ignoreGasPricing() {
		return this.config.ignoreGasPricing;
	}
	public set ignoreGasPricing(val) {
		this._triggerConfigChange('ignoreGasPricing', val);
		this.config.ignoreGasPricing = val;
	}
	public get defaultTransactionType() {
		return this.config.defaultTransactionType;
	}

	public set defaultTransactionType(val) {
		this._triggerConfigChange('defaultTransactionType', val);

		this.config.defaultTransactionType = val;
	}

	public get defaultMaxPriorityFeePerGas() {
		return this.config.defaultMaxPriorityFeePerGas;
	}

	public set defaultMaxPriorityFeePerGas(val) {
		this._triggerConfigChange('defaultMaxPriorityFeePerGas', val);
		this.config.defaultMaxPriorityFeePerGas = val;
	}

	public get transactionBuilder() {
		return this.config.transactionBuilder;
	}

	public set transactionBuilder(val) {
		this._triggerConfigChange('transactionBuilder', val);
		this.config.transactionBuilder = val;
	}

	public get transactionTypeParser() {
		return this.config.transactionTypeParser;
	}

	public set transactionTypeParser(val) {
		this._triggerConfigChange('transactionTypeParser', val);
		this.config.transactionTypeParser = val;
	}

	public get customTransactionSchema(): CustomTransactionSchema | undefined {
		return this.config.customTransactionSchema;
	}

	public set customTransactionSchema(schema: CustomTransactionSchema | undefined) {
		this._triggerConfigChange('customTransactionSchema', schema);
		this.config.customTransactionSchema = schema;
	}

	private _triggerConfigChange<K extends keyof Web3ConfigOptions>(
		config: K,
		newValue: Web3ConfigOptions[K],
	) {
		this.emit(Web3ConfigEvent.CONFIG_CHANGE, {
			name: config,
			oldValue: this.config[config],
			newValue,
		} as ConfigEvent<Web3ConfigOptions>);
	}
}
