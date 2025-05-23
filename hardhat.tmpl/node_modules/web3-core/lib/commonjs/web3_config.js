"use strict";
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
Object.defineProperty(exports, "__esModule", { value: true });
exports.Web3Config = exports.Web3ConfigEvent = void 0;
const web3_types_1 = require("web3-types");
const web3_errors_1 = require("web3-errors");
const web3_utils_1 = require("web3-utils");
const web3_event_emitter_js_1 = require("./web3_event_emitter.js");
var Web3ConfigEvent;
(function (Web3ConfigEvent) {
    Web3ConfigEvent["CONFIG_CHANGE"] = "CONFIG_CHANGE";
})(Web3ConfigEvent || (exports.Web3ConfigEvent = Web3ConfigEvent = {}));
class Web3Config extends web3_event_emitter_js_1.Web3EventEmitter {
    constructor(options) {
        super();
        this.config = {
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
            defaultMaxPriorityFeePerGas: (0, web3_utils_1.toHex)(2500000000),
            enableExperimentalFeatures: {
                useSubscriptionWhenCheckingBlockTimeout: false,
                useRpcCallSpecification: false,
            },
            transactionBuilder: undefined,
            transactionTypeParser: undefined,
            customTransactionSchema: undefined,
            defaultReturnFormat: web3_types_1.DEFAULT_RETURN_FORMAT,
            ignoreGasPricing: false,
        };
        this.setConfig(options !== null && options !== void 0 ? options : {});
    }
    setConfig(options) {
        // TODO: Improve and add key check
        const keys = Object.keys(options);
        for (const key of keys) {
            this._triggerConfigChange(key, options[key]);
            if (!(0, web3_utils_1.isNullish)(options[key]) &&
                typeof options[key] === 'number' &&
                key === 'maxListenersWarningThreshold') {
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
    get handleRevert() {
        return this.config.handleRevert;
    }
    /**
     * Will set the handleRevert
     */
    set handleRevert(val) {
        this._triggerConfigChange('handleRevert', val);
        this.config.handleRevert = val;
    }
    /**
     * The `contractDataInputFill` options property will allow you to set the hash of the method signature and encoded parameters to the property
     * either `data`, `input` or both within your contract.
     * This will affect the contracts send, call and estimateGas methods
     * Default is `data`.
     */
    get contractDataInputFill() {
        return this.config.contractDataInputFill;
    }
    /**
     * Will set the contractDataInputFill
     */
    set contractDataInputFill(val) {
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
    get defaultAccount() {
        return this.config.defaultAccount;
    }
    /**
     * Will set the default account.
     */
    set defaultAccount(val) {
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
    get defaultBlock() {
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
    set defaultBlock(val) {
        this._triggerConfigChange('defaultBlock', val);
        this.config.defaultBlock = val;
    }
    /**
     * The time used to wait for Ethereum Node to return the sent transaction result.
     * Note: If the RPC call stuck at the Node and therefor timed-out, the transaction may still be pending or even mined by the Network. We recommend checking the pending transactions in such a case.
     * Default is `750` seconds (12.5 minutes).
     */
    get transactionSendTimeout() {
        return this.config.transactionSendTimeout;
    }
    /**
     * Will set the transactionSendTimeout.
     */
    set transactionSendTimeout(val) {
        this._triggerConfigChange('transactionSendTimeout', val);
        this.config.transactionSendTimeout = val;
    }
    /**
     * The `transactionBlockTimeout` is used over socket-based connections. This option defines the amount of new blocks it should wait until the first confirmation happens, otherwise the PromiEvent rejects with a timeout error.
     * Default is `50`.
     */
    get transactionBlockTimeout() {
        return this.config.transactionBlockTimeout;
    }
    /**
     * Will set the transactionBlockTimeout.
     */
    set transactionBlockTimeout(val) {
        this._triggerConfigChange('transactionBlockTimeout', val);
        this.config.transactionBlockTimeout = val;
    }
    /**
     * This defines the number of blocks it requires until a transaction is considered confirmed.
     * Default is `24`.
     */
    get transactionConfirmationBlocks() {
        return this.config.transactionConfirmationBlocks;
    }
    /**
     * Will set the transactionConfirmationBlocks.
     */
    set transactionConfirmationBlocks(val) {
        this._triggerConfigChange('transactionConfirmationBlocks', val);
        this.config.transactionConfirmationBlocks = val;
    }
    /**
     * Used over HTTP connections. This option defines the number of seconds between Web3 calls for a receipt which confirms that a transaction was mined by the network.
     * Default is `1000` ms.
     */
    get transactionPollingInterval() {
        return this.config.transactionPollingInterval;
    }
    /**
     * Will set the transactionPollingInterval.
     */
    set transactionPollingInterval(val) {
        this._triggerConfigChange('transactionPollingInterval', val);
        this.config.transactionPollingInterval = val;
        this.transactionReceiptPollingInterval = val;
        this.transactionConfirmationPollingInterval = val;
    }
    /**
     * Used over HTTP connections. This option defines the number of seconds Web3 will wait for a receipt which confirms that a transaction was mined by the network. Note: If this method times out, the transaction may still be pending.
     * Default is `750` seconds (12.5 minutes).
     */
    get transactionPollingTimeout() {
        return this.config.transactionPollingTimeout;
    }
    /**
     * Will set the transactionPollingTimeout.
     */
    set transactionPollingTimeout(val) {
        this._triggerConfigChange('transactionPollingTimeout', val);
        this.config.transactionPollingTimeout = val;
    }
    /**
     * The `transactionPollingInterval` is used over HTTP connections. This option defines the number of seconds between Web3 calls for a receipt which confirms that a transaction was mined by the network.
     * Default is `undefined`
     */
    get transactionReceiptPollingInterval() {
        return this.config.transactionReceiptPollingInterval;
    }
    /**
     * Will set the transactionReceiptPollingInterval
     */
    set transactionReceiptPollingInterval(val) {
        this._triggerConfigChange('transactionReceiptPollingInterval', val);
        this.config.transactionReceiptPollingInterval = val;
    }
    get transactionConfirmationPollingInterval() {
        return this.config.transactionConfirmationPollingInterval;
    }
    set transactionConfirmationPollingInterval(val) {
        this._triggerConfigChange('transactionConfirmationPollingInterval', val);
        this.config.transactionConfirmationPollingInterval = val;
    }
    /**
     * The blockHeaderTimeout is used over socket-based connections. This option defines the amount seconds it should wait for `'newBlockHeaders'` event before falling back to polling to fetch transaction receipt.
     * Default is `10` seconds.
     */
    get blockHeaderTimeout() {
        return this.config.blockHeaderTimeout;
    }
    /**
     * Will set the blockHeaderTimeout
     */
    set blockHeaderTimeout(val) {
        this._triggerConfigChange('blockHeaderTimeout', val);
        this.config.blockHeaderTimeout = val;
    }
    /**
     * The enableExperimentalFeatures is used to enable trying new experimental features that are still not fully implemented or not fully tested or still have some related issues.
     * Default is `false` for every feature.
     */
    get enableExperimentalFeatures() {
        return this.config.enableExperimentalFeatures;
    }
    /**
     * Will set the enableExperimentalFeatures
     */
    set enableExperimentalFeatures(val) {
        this._triggerConfigChange('enableExperimentalFeatures', val);
        this.config.enableExperimentalFeatures = val;
    }
    get maxListenersWarningThreshold() {
        return this.config.maxListenersWarningThreshold;
    }
    set maxListenersWarningThreshold(val) {
        this._triggerConfigChange('maxListenersWarningThreshold', val);
        this.setMaxListenerWarningThreshold(val);
        this.config.maxListenersWarningThreshold = val;
    }
    get defaultReturnFormat() {
        return this.config.defaultReturnFormat;
    }
    set defaultReturnFormat(val) {
        this._triggerConfigChange('defaultReturnFormat', val);
        this.config.defaultReturnFormat = val;
    }
    get defaultNetworkId() {
        return this.config.defaultNetworkId;
    }
    set defaultNetworkId(val) {
        this._triggerConfigChange('defaultNetworkId', val);
        this.config.defaultNetworkId = val;
    }
    get defaultChain() {
        return this.config.defaultChain;
    }
    set defaultChain(val) {
        if (!(0, web3_utils_1.isNullish)(this.config.defaultCommon) &&
            !(0, web3_utils_1.isNullish)(this.config.defaultCommon.baseChain) &&
            val !== this.config.defaultCommon.baseChain)
            throw new web3_errors_1.ConfigChainMismatchError(this.config.defaultChain, val);
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
    get defaultHardfork() {
        return this.config.defaultHardfork;
    }
    /**
     * Will set the default hardfork.
     *
     */
    set defaultHardfork(val) {
        if (!(0, web3_utils_1.isNullish)(this.config.defaultCommon) &&
            !(0, web3_utils_1.isNullish)(this.config.defaultCommon.hardfork) &&
            val !== this.config.defaultCommon.hardfork)
            throw new web3_errors_1.ConfigHardforkMismatchError(this.config.defaultCommon.hardfork, val);
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
    get defaultCommon() {
        return this.config.defaultCommon;
    }
    /**
     * Will set the default common property
     *
     */
    set defaultCommon(val) {
        // validation check if default hardfork is set and matches defaultCommon hardfork
        if (!(0, web3_utils_1.isNullish)(this.config.defaultHardfork) &&
            !(0, web3_utils_1.isNullish)(val) &&
            !(0, web3_utils_1.isNullish)(val.hardfork) &&
            this.config.defaultHardfork !== val.hardfork)
            throw new web3_errors_1.ConfigHardforkMismatchError(this.config.defaultHardfork, val.hardfork);
        if (!(0, web3_utils_1.isNullish)(this.config.defaultChain) &&
            !(0, web3_utils_1.isNullish)(val) &&
            !(0, web3_utils_1.isNullish)(val.baseChain) &&
            this.config.defaultChain !== val.baseChain)
            throw new web3_errors_1.ConfigChainMismatchError(this.config.defaultChain, val.baseChain);
        this._triggerConfigChange('defaultCommon', val);
        this.config.defaultCommon = val;
    }
    /**
     *  Will get the ignoreGasPricing property. When true, the gasPrice, maxPriorityFeePerGas, and maxFeePerGas will not be autofilled in the transaction object.
     *  Useful when you want wallets to handle gas pricing.
     */
    get ignoreGasPricing() {
        return this.config.ignoreGasPricing;
    }
    set ignoreGasPricing(val) {
        this._triggerConfigChange('ignoreGasPricing', val);
        this.config.ignoreGasPricing = val;
    }
    get defaultTransactionType() {
        return this.config.defaultTransactionType;
    }
    set defaultTransactionType(val) {
        this._triggerConfigChange('defaultTransactionType', val);
        this.config.defaultTransactionType = val;
    }
    get defaultMaxPriorityFeePerGas() {
        return this.config.defaultMaxPriorityFeePerGas;
    }
    set defaultMaxPriorityFeePerGas(val) {
        this._triggerConfigChange('defaultMaxPriorityFeePerGas', val);
        this.config.defaultMaxPriorityFeePerGas = val;
    }
    get transactionBuilder() {
        return this.config.transactionBuilder;
    }
    set transactionBuilder(val) {
        this._triggerConfigChange('transactionBuilder', val);
        this.config.transactionBuilder = val;
    }
    get transactionTypeParser() {
        return this.config.transactionTypeParser;
    }
    set transactionTypeParser(val) {
        this._triggerConfigChange('transactionTypeParser', val);
        this.config.transactionTypeParser = val;
    }
    get customTransactionSchema() {
        return this.config.customTransactionSchema;
    }
    set customTransactionSchema(schema) {
        this._triggerConfigChange('customTransactionSchema', schema);
        this.config.customTransactionSchema = schema;
    }
    _triggerConfigChange(config, newValue) {
        this.emit(Web3ConfigEvent.CONFIG_CHANGE, {
            name: config,
            oldValue: this.config[config],
            newValue,
        });
    }
}
exports.Web3Config = Web3Config;
//# sourceMappingURL=web3_config.js.map