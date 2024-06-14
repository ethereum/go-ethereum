"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Common = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const events_1 = require("events");
const chains_js_1 = require("./chains.js");
const crc_js_1 = require("./crc.js");
const eips_js_1 = require("./eips.js");
const enums_js_1 = require("./enums.js");
const hardforks_js_1 = require("./hardforks.js");
const utils_js_1 = require("./utils.js");
/**
 * Common class to access chain and hardfork parameters and to provide
 * a unified and shared view on the network and hardfork state.
 *
 * Use the {@link Common.custom} static constructor for creating simple
 * custom chain {@link Common} objects (more complete custom chain setups
 * can be created via the main constructor and the {@link CommonOpts.customChains} parameter).
 */
class Common {
    constructor(opts) {
        this._eips = [];
        this._paramsCache = {};
        this._activatedEIPsCache = [];
        this.events = new events_1.EventEmitter();
        this._customChains = opts.customChains ?? [];
        this._chainParams = this.setChain(opts.chain);
        this.DEFAULT_HARDFORK = this._chainParams.defaultHardfork ?? enums_js_1.Hardfork.Shanghai;
        // Assign hardfork changes in the sequence of the applied hardforks
        this.HARDFORK_CHANGES = this.hardforks().map((hf) => [
            hf.name,
            hardforks_js_1.hardforks[hf.name],
        ]);
        this._hardfork = this.DEFAULT_HARDFORK;
        if (opts.hardfork !== undefined) {
            this.setHardfork(opts.hardfork);
        }
        if (opts.eips) {
            this.setEIPs(opts.eips);
        }
        this.customCrypto = opts.customCrypto ?? {};
        if (Object.keys(this._paramsCache).length === 0) {
            this._buildParamsCache();
            this._buildActivatedEIPsCache();
        }
    }
    /**
     * Creates a {@link Common} object for a custom chain, based on a standard one.
     *
     * It uses all the {@link Chain} parameters from the {@link baseChain} option except the ones overridden
     * in a provided {@link chainParamsOrName} dictionary. Some usage example:
     *
     * ```javascript
     * Common.custom({chainId: 123})
     * ```
     *
     * There are also selected supported custom chains which can be initialized by using one of the
     * {@link CustomChains} for {@link chainParamsOrName}, e.g.:
     *
     * ```javascript
     * Common.custom(CustomChains.MaticMumbai)
     * ```
     *
     * Note that these supported custom chains only provide some base parameters (usually the chain and
     * network ID and a name) and can only be used for selected use cases (e.g. sending a tx with
     * the `@ethereumjs/tx` library to a Layer-2 chain).
     *
     * @param chainParamsOrName Custom parameter dict (`name` will default to `custom-chain`) or string with name of a supported custom chain
     * @param opts Custom chain options to set the {@link CustomCommonOpts.baseChain}, selected {@link CustomCommonOpts.hardfork} and others
     */
    static custom(chainParamsOrName, opts = {}) {
        const baseChain = opts.baseChain ?? 'mainnet';
        const standardChainParams = { ...Common._getChainParams(baseChain) };
        standardChainParams['name'] = 'custom-chain';
        if (typeof chainParamsOrName !== 'string') {
            return new Common({
                chain: {
                    ...standardChainParams,
                    ...chainParamsOrName,
                },
                ...opts,
            });
        }
        else {
            if (chainParamsOrName === enums_js_1.CustomChain.PolygonMainnet) {
                return Common.custom({
                    name: enums_js_1.CustomChain.PolygonMainnet,
                    chainId: 137,
                    networkId: 137,
                }, opts);
            }
            if (chainParamsOrName === enums_js_1.CustomChain.PolygonMumbai) {
                return Common.custom({
                    name: enums_js_1.CustomChain.PolygonMumbai,
                    chainId: 80001,
                    networkId: 80001,
                }, opts);
            }
            if (chainParamsOrName === enums_js_1.CustomChain.ArbitrumOne) {
                return Common.custom({
                    name: enums_js_1.CustomChain.ArbitrumOne,
                    chainId: 42161,
                    networkId: 42161,
                }, opts);
            }
            if (chainParamsOrName === enums_js_1.CustomChain.xDaiChain) {
                return Common.custom({
                    name: enums_js_1.CustomChain.xDaiChain,
                    chainId: 100,
                    networkId: 100,
                }, opts);
            }
            if (chainParamsOrName === enums_js_1.CustomChain.OptimisticKovan) {
                return Common.custom({
                    name: enums_js_1.CustomChain.OptimisticKovan,
                    chainId: 69,
                    networkId: 69,
                }, 
                // Optimism has not implemented the London hardfork yet (targeting Q1.22)
                { hardfork: enums_js_1.Hardfork.Berlin, ...opts });
            }
            if (chainParamsOrName === enums_js_1.CustomChain.OptimisticEthereum) {
                return Common.custom({
                    name: enums_js_1.CustomChain.OptimisticEthereum,
                    chainId: 10,
                    networkId: 10,
                }, 
                // Optimism has not implemented the London hardfork yet (targeting Q1.22)
                { hardfork: enums_js_1.Hardfork.Berlin, ...opts });
            }
            throw new Error(`Custom chain ${chainParamsOrName} not supported`);
        }
    }
    /**
     * Static method to load and set common from a geth genesis json
     * @param genesisJson json of geth configuration
     * @param { chain, eips, genesisHash, hardfork, mergeForkIdPostMerge } to further configure the common instance
     * @returns Common
     */
    static fromGethGenesis(genesisJson, { chain, eips, genesisHash, hardfork, mergeForkIdPostMerge }) {
        const genesisParams = (0, utils_js_1.parseGethGenesis)(genesisJson, chain, mergeForkIdPostMerge);
        const common = new Common({
            chain: genesisParams.name ?? 'custom',
            customChains: [genesisParams],
            eips,
            hardfork: hardfork ?? genesisParams.hardfork,
        });
        if (genesisHash !== undefined) {
            common.setForkHashes(genesisHash);
        }
        return common;
    }
    /**
     * Static method to determine if a {@link chainId} is supported as a standard chain
     * @param chainId bigint id (`1`) of a standard chain
     * @returns boolean
     */
    static isSupportedChainId(chainId) {
        const initializedChains = this.getInitializedChains();
        return Boolean(initializedChains['names'][chainId.toString()]);
    }
    static _getChainParams(chain, customChains) {
        const initializedChains = this.getInitializedChains(customChains);
        if (typeof chain === 'number' || typeof chain === 'bigint') {
            chain = chain.toString();
            if (initializedChains['names'][chain]) {
                const name = initializedChains['names'][chain];
                return initializedChains[name];
            }
            throw new Error(`Chain with ID ${chain} not supported`);
        }
        if (initializedChains[chain] !== undefined) {
            return initializedChains[chain];
        }
        throw new Error(`Chain with name ${chain} not supported`);
    }
    /**
     * Sets the chain
     * @param chain String ('mainnet') or Number (1) chain representation.
     *              Or, a Dictionary of chain parameters for a private network.
     * @returns The dictionary with parameters set as chain
     */
    setChain(chain) {
        if (typeof chain === 'number' || typeof chain === 'bigint' || typeof chain === 'string') {
            this._chainParams = Common._getChainParams(chain, this._customChains);
        }
        else if (typeof chain === 'object') {
            if (this._customChains.length > 0) {
                throw new Error('Chain must be a string, number, or bigint when initialized with customChains passed in');
            }
            const required = ['networkId', 'genesis', 'hardforks', 'bootstrapNodes'];
            for (const param of required) {
                if (!(param in chain)) {
                    throw new Error(`Missing required chain parameter: ${param}`);
                }
            }
            this._chainParams = chain;
        }
        else {
            throw new Error('Wrong input format');
        }
        for (const hf of this.hardforks()) {
            if (hf.block === undefined) {
                throw new Error(`Hardfork cannot have undefined block number`);
            }
        }
        return this._chainParams;
    }
    /**
     * Sets the hardfork to get params for
     * @param hardfork String identifier (e.g. 'byzantium') or {@link Hardfork} enum
     */
    setHardfork(hardfork) {
        let existing = false;
        for (const hfChanges of this.HARDFORK_CHANGES) {
            if (hfChanges[0] === hardfork) {
                if (this._hardfork !== hardfork) {
                    this._hardfork = hardfork;
                    this._buildParamsCache();
                    this._buildActivatedEIPsCache();
                    this.events.emit('hardforkChanged', hardfork);
                }
                existing = true;
            }
        }
        if (!existing) {
            throw new Error(`Hardfork with name ${hardfork} not supported`);
        }
    }
    /**
     * Returns the hardfork either based on block numer (older HFs) or
     * timestamp (Shanghai upwards).
     *
     * An optional TD takes precedence in case the corresponding HF block
     * is set to `null` or otherwise needs to match (if not an error
     * will be thrown).
     *
     * @param Opts Block number, timestamp or TD (all optional)
     * @returns The name of the HF
     */
    getHardforkBy(opts) {
        let { blockNumber, timestamp, td } = opts;
        blockNumber = (0, ethereumjs_util_1.toType)(blockNumber, ethereumjs_util_1.TypeOutput.BigInt);
        td = (0, ethereumjs_util_1.toType)(td, ethereumjs_util_1.TypeOutput.BigInt);
        timestamp = (0, ethereumjs_util_1.toType)(timestamp, ethereumjs_util_1.TypeOutput.BigInt);
        // Filter out hardforks with no block number, no ttd or no timestamp (i.e. unapplied hardforks)
        const hfs = this.hardforks().filter((hf) => hf.block !== null || (hf.ttd !== null && hf.ttd !== undefined) || hf.timestamp !== undefined);
        const mergeIndex = hfs.findIndex((hf) => hf.ttd !== null && hf.ttd !== undefined);
        const doubleTTDHF = hfs
            .slice(mergeIndex + 1)
            .findIndex((hf) => hf.ttd !== null && hf.ttd !== undefined);
        if (doubleTTDHF >= 0) {
            throw Error(`More than one merge hardforks found with ttd specified`);
        }
        // Find the first hardfork that has a block number greater than `blockNumber`
        // (skips the merge hardfork since it cannot have a block number specified).
        // If timestamp is not provided, it also skips timestamps hardforks to continue
        // discovering/checking number hardforks.
        let hfIndex = hfs.findIndex((hf) => (blockNumber !== undefined &&
            hf.block !== null &&
            BigInt(hf.block) > blockNumber) ||
            (timestamp !== undefined && hf.timestamp !== undefined && hf.timestamp > timestamp));
        if (hfIndex === -1) {
            // all hardforks apply, set hfIndex to the last one as that's the candidate
            hfIndex = hfs.length;
        }
        else if (hfIndex === 0) {
            // cannot have a case where a block number is before all applied hardforks
            // since the chain has to start with a hardfork
            throw Error('Must have at least one hardfork at block 0');
        }
        // If timestamp is not provided, we need to rollback to the last hf with block or ttd
        if (timestamp === undefined) {
            const stepBack = hfs
                .slice(0, hfIndex)
                .reverse()
                .findIndex((hf) => hf.block !== null || hf.ttd !== undefined);
            hfIndex = hfIndex - stepBack;
        }
        // Move hfIndex one back to arrive at candidate hardfork
        hfIndex = hfIndex - 1;
        // If the timestamp was not provided, we could have skipped timestamp hardforks to look for number
        // hardforks. so it will now be needed to rollback
        if (hfs[hfIndex].block === null && hfs[hfIndex].timestamp === undefined) {
            // We're on the merge hardfork.  Let's check the TTD
            if (td === undefined || td === null || BigInt(hfs[hfIndex].ttd) > td) {
                // Merge ttd greater than current td so we're on hardfork before merge
                hfIndex -= 1;
            }
        }
        else {
            if (mergeIndex >= 0 && td !== undefined && td !== null) {
                if (hfIndex >= mergeIndex && BigInt(hfs[mergeIndex].ttd) > td) {
                    throw Error('Maximum HF determined by total difficulty is lower than the block number HF');
                }
                else if (hfIndex < mergeIndex && BigInt(hfs[mergeIndex].ttd) < td) {
                    throw Error('HF determined by block number is lower than the minimum total difficulty HF');
                }
            }
        }
        const hfStartIndex = hfIndex;
        // Move the hfIndex to the end of the hardforks that might be scheduled on the same block/timestamp
        // This won't anyway be the case with Merge hfs
        for (; hfIndex < hfs.length - 1; hfIndex++) {
            // break out if hfIndex + 1 is not scheduled at hfIndex
            if (hfs[hfIndex].block !== hfs[hfIndex + 1].block ||
                hfs[hfIndex].timestamp !== hfs[hfIndex + 1].timestamp) {
                break;
            }
        }
        if (timestamp !== undefined) {
            const minTimeStamp = hfs
                .slice(0, hfStartIndex)
                .reduce((acc, hf) => Math.max(Number(hf.timestamp ?? '0'), acc), 0);
            if (minTimeStamp > timestamp) {
                throw Error(`Maximum HF determined by timestamp is lower than the block number/ttd HF`);
            }
            const maxTimeStamp = hfs
                .slice(hfIndex + 1)
                .reduce((acc, hf) => Math.min(Number(hf.timestamp ?? timestamp), acc), Number(timestamp));
            if (maxTimeStamp < timestamp) {
                throw Error(`Maximum HF determined by block number/ttd is lower than timestamp HF`);
            }
        }
        const hardfork = hfs[hfIndex];
        return hardfork.name;
    }
    /**
     * Sets a new hardfork either based on block numer (older HFs) or
     * timestamp (Shanghai upwards).
     *
     * An optional TD takes precedence in case the corresponding HF block
     * is set to `null` or otherwise needs to match (if not an error
     * will be thrown).
     *
     * @param Opts Block number, timestamp or TD (all optional)
     * @returns The name of the HF set
     */
    setHardforkBy(opts) {
        const hardfork = this.getHardforkBy(opts);
        this.setHardfork(hardfork);
        return hardfork;
    }
    /**
     * Internal helper function, returns the params for the given hardfork for the chain set
     * @param hardfork Hardfork name
     * @returns Dictionary with hardfork params or null if hardfork not on chain
     */
    _getHardfork(hardfork) {
        const hfs = this.hardforks();
        for (const hf of hfs) {
            if (hf['name'] === hardfork)
                return hf;
        }
        return null;
    }
    /**
     * Sets the active EIPs
     * @param eips
     */
    setEIPs(eips = []) {
        for (const eip of eips) {
            if (!(eip in eips_js_1.EIPs)) {
                throw new Error(`${eip} not supported`);
            }
            const minHF = this.gteHardfork(eips_js_1.EIPs[eip]['minimumHardfork']);
            if (!minHF) {
                throw new Error(`${eip} cannot be activated on hardfork ${this.hardfork()}, minimumHardfork: ${minHF}`);
            }
        }
        this._eips = eips;
        this._buildParamsCache();
        this._buildActivatedEIPsCache();
        for (const eip of eips) {
            if (eips_js_1.EIPs[eip].requiredEIPs !== undefined) {
                for (const elem of eips_js_1.EIPs[eip].requiredEIPs) {
                    if (!(eips.includes(elem) || this.isActivatedEIP(elem))) {
                        throw new Error(`${eip} requires EIP ${elem}, but is not included in the EIP list`);
                    }
                }
            }
        }
    }
    /**
     * Internal helper for _buildParamsCache()
     */
    _mergeWithParamsCache(params) {
        this._paramsCache['gasConfig'] = {
            ...this._paramsCache['gasConfig'],
            ...params['gasConfig'],
        };
        this._paramsCache['gasPrices'] = {
            ...this._paramsCache['gasPrices'],
            ...params['gasPrices'],
        };
        this._paramsCache['pow'] = {
            ...this._paramsCache['pow'],
            ...params['pow'],
        };
        this._paramsCache['sharding'] = {
            ...this._paramsCache['sharding'],
            ...params['sharding'],
        };
        this._paramsCache['vm'] = {
            ...this._paramsCache['vm'],
            ...params['vm'],
        };
    }
    /**
     * Build up a cache for all parameter values for the current HF and all activated EIPs
     */
    _buildParamsCache() {
        this._paramsCache = {};
        // Iterate through all hardforks up to hardfork set
        const hardfork = this.hardfork();
        for (const hfChanges of this.HARDFORK_CHANGES) {
            // EIP-referencing HF config (e.g. for berlin)
            if ('eips' in hfChanges[1]) {
                const hfEIPs = hfChanges[1]['eips'];
                for (const eip of hfEIPs) {
                    if (!(eip in eips_js_1.EIPs)) {
                        throw new Error(`${eip} not supported`);
                    }
                    this._mergeWithParamsCache(eips_js_1.EIPs[eip]);
                }
                // Parameter-inlining HF config (e.g. for istanbul)
            }
            else {
                this._mergeWithParamsCache(hfChanges[1]);
            }
            if (hfChanges[0] === hardfork)
                break;
        }
        // Iterate through all additionally activated EIPs
        for (const eip of this._eips) {
            if (!(eip in eips_js_1.EIPs)) {
                throw new Error(`${eip} not supported`);
            }
            this._mergeWithParamsCache(eips_js_1.EIPs[eip]);
        }
    }
    _buildActivatedEIPsCache() {
        this._activatedEIPsCache = [];
        for (const hfChanges of this.HARDFORK_CHANGES) {
            const hf = hfChanges[1];
            if (this.gteHardfork(hf['name']) && 'eips' in hf) {
                this._activatedEIPsCache = this._activatedEIPsCache.concat(hf['eips']);
            }
        }
        this._activatedEIPsCache = this._activatedEIPsCache.concat(this._eips);
    }
    /**
     * Returns a parameter for the current chain setup
     *
     * If the parameter is present in an EIP, the EIP always takes precedence.
     * Otherwise the parameter is taken from the latest applied HF with
     * a change on the respective parameter.
     *
     * @param topic Parameter topic ('gasConfig', 'gasPrices', 'vm', 'pow')
     * @param name Parameter name (e.g. 'minGasLimit' for 'gasConfig' topic)
     * @returns The value requested or `BigInt(0)` if not found
     */
    param(topic, name) {
        // TODO: consider the case that different active EIPs
        // can change the same parameter
        let value = null;
        if (this._paramsCache[topic] !== undefined &&
            this._paramsCache[topic][name] !== undefined) {
            value = this._paramsCache[topic][name].v;
        }
        return BigInt(value ?? 0);
    }
    /**
     * Returns the parameter corresponding to a hardfork
     * @param topic Parameter topic ('gasConfig', 'gasPrices', 'vm', 'pow')
     * @param name Parameter name (e.g. 'minGasLimit' for 'gasConfig' topic)
     * @param hardfork Hardfork name
     * @returns The value requested or `BigInt(0)` if not found
     */
    paramByHardfork(topic, name, hardfork) {
        let value = null;
        for (const hfChanges of this.HARDFORK_CHANGES) {
            // EIP-referencing HF config (e.g. for berlin)
            if ('eips' in hfChanges[1]) {
                const hfEIPs = hfChanges[1]['eips'];
                for (const eip of hfEIPs) {
                    const valueEIP = this.paramByEIP(topic, name, eip);
                    value = typeof valueEIP === 'bigint' ? valueEIP : value;
                }
                // Parameter-inlining HF config (e.g. for istanbul)
            }
            else {
                if (hfChanges[1][topic] !== undefined &&
                    hfChanges[1][topic][name] !== undefined) {
                    value = hfChanges[1][topic][name].v;
                }
            }
            if (hfChanges[0] === hardfork)
                break;
        }
        return BigInt(value ?? 0);
    }
    /**
     * Returns a parameter corresponding to an EIP
     * @param topic Parameter topic ('gasConfig', 'gasPrices', 'vm', 'pow')
     * @param name Parameter name (e.g. 'minGasLimit' for 'gasConfig' topic)
     * @param eip Number of the EIP
     * @returns The value requested or `undefined` if not found
     */
    paramByEIP(topic, name, eip) {
        if (!(eip in eips_js_1.EIPs)) {
            throw new Error(`${eip} not supported`);
        }
        const eipParams = eips_js_1.EIPs[eip];
        if (!(topic in eipParams)) {
            return undefined;
        }
        if (eipParams[topic][name] === undefined) {
            return undefined;
        }
        const value = eipParams[topic][name].v;
        return BigInt(value);
    }
    /**
     * Returns a parameter for the hardfork active on block number or
     * optional provided total difficulty (Merge HF)
     * @param topic Parameter topic
     * @param name Parameter name
     * @param blockNumber Block number
     * @param td Total difficulty
     *    * @returns The value requested or `BigInt(0)` if not found
     */
    paramByBlock(topic, name, blockNumber, td, timestamp) {
        const hardfork = this.getHardforkBy({ blockNumber, td, timestamp });
        return this.paramByHardfork(topic, name, hardfork);
    }
    /**
     * Checks if an EIP is activated by either being included in the EIPs
     * manually passed in with the {@link CommonOpts.eips} or in a
     * hardfork currently being active
     *
     * Note: this method only works for EIPs being supported
     * by the {@link CommonOpts.eips} constructor option
     * @param eip
     */
    isActivatedEIP(eip) {
        if (this._activatedEIPsCache.includes(eip)) {
            return true;
        }
        return false;
    }
    /**
     * Checks if set or provided hardfork is active on block number
     * @param hardfork Hardfork name or null (for HF set)
     * @param blockNumber
     * @returns True if HF is active on block number
     */
    hardforkIsActiveOnBlock(hardfork, blockNumber) {
        blockNumber = (0, ethereumjs_util_1.toType)(blockNumber, ethereumjs_util_1.TypeOutput.BigInt);
        hardfork = hardfork ?? this._hardfork;
        const hfBlock = this.hardforkBlock(hardfork);
        if (typeof hfBlock === 'bigint' && hfBlock !== ethereumjs_util_1.BIGINT_0 && blockNumber >= hfBlock) {
            return true;
        }
        return false;
    }
    /**
     * Alias to hardforkIsActiveOnBlock when hardfork is set
     * @param blockNumber
     * @returns True if HF is active on block number
     */
    activeOnBlock(blockNumber) {
        return this.hardforkIsActiveOnBlock(null, blockNumber);
    }
    /**
     * Sequence based check if given or set HF1 is greater than or equal HF2
     * @param hardfork1 Hardfork name or null (if set)
     * @param hardfork2 Hardfork name
     * @param opts Hardfork options
     * @returns True if HF1 gte HF2
     */
    hardforkGteHardfork(hardfork1, hardfork2) {
        hardfork1 = hardfork1 ?? this._hardfork;
        const hardforks = this.hardforks();
        let posHf1 = -1, posHf2 = -1;
        let index = 0;
        for (const hf of hardforks) {
            if (hf['name'] === hardfork1)
                posHf1 = index;
            if (hf['name'] === hardfork2)
                posHf2 = index;
            index += 1;
        }
        return posHf1 >= posHf2 && posHf2 !== -1;
    }
    /**
     * Alias to hardforkGteHardfork when hardfork is set
     * @param hardfork Hardfork name
     * @returns True if hardfork set is greater than hardfork provided
     */
    gteHardfork(hardfork) {
        return this.hardforkGteHardfork(null, hardfork);
    }
    /**
     * Returns the hardfork change block for hardfork provided or set
     * @param hardfork Hardfork name, optional if HF set
     * @returns Block number or null if unscheduled
     */
    hardforkBlock(hardfork) {
        hardfork = hardfork ?? this._hardfork;
        const block = this._getHardfork(hardfork)?.['block'];
        if (block === undefined || block === null) {
            return null;
        }
        return BigInt(block);
    }
    hardforkTimestamp(hardfork) {
        hardfork = hardfork ?? this._hardfork;
        const timestamp = this._getHardfork(hardfork)?.['timestamp'];
        if (timestamp === undefined || timestamp === null) {
            return null;
        }
        return BigInt(timestamp);
    }
    /**
     * Returns the hardfork change block for eip
     * @param eip EIP number
     * @returns Block number or null if unscheduled
     */
    eipBlock(eip) {
        for (const hfChanges of this.HARDFORK_CHANGES) {
            const hf = hfChanges[1];
            if ('eips' in hf) {
                // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
                if (hf['eips'].includes(eip)) {
                    return this.hardforkBlock(hfChanges[0]);
                }
            }
        }
        return null;
    }
    /**
     * Returns the hardfork change total difficulty (Merge HF) for hardfork provided or set
     * @param hardfork Hardfork name, optional if HF set
     * @returns Total difficulty or null if no set
     */
    hardforkTTD(hardfork) {
        hardfork = hardfork ?? this._hardfork;
        const ttd = this._getHardfork(hardfork)?.['ttd'];
        if (ttd === undefined || ttd === null) {
            return null;
        }
        return BigInt(ttd);
    }
    /**
     * Returns the change block for the next hardfork after the hardfork provided or set
     * @param hardfork Hardfork name, optional if HF set
     * @returns Block timestamp, number or null if not available
     */
    nextHardforkBlockOrTimestamp(hardfork) {
        hardfork = hardfork ?? this._hardfork;
        const hfs = this.hardforks();
        let hfIndex = hfs.findIndex((hf) => hf.name === hardfork);
        // If the current hardfork is merge, go one behind as merge hf is not part of these
        // calcs even if the merge hf block is set
        if (hardfork === enums_js_1.Hardfork.Paris) {
            hfIndex -= 1;
        }
        // Hardfork not found
        if (hfIndex < 0) {
            return null;
        }
        let currHfTimeOrBlock = hfs[hfIndex].timestamp ?? hfs[hfIndex].block;
        currHfTimeOrBlock =
            currHfTimeOrBlock !== null && currHfTimeOrBlock !== undefined
                ? Number(currHfTimeOrBlock)
                : null;
        const nextHf = hfs.slice(hfIndex + 1).find((hf) => {
            let hfTimeOrBlock = hf.timestamp ?? hf.block;
            hfTimeOrBlock =
                hfTimeOrBlock !== null && hfTimeOrBlock !== undefined ? Number(hfTimeOrBlock) : null;
            return (hf.name !== enums_js_1.Hardfork.Paris &&
                hfTimeOrBlock !== null &&
                hfTimeOrBlock !== undefined &&
                hfTimeOrBlock !== currHfTimeOrBlock);
        });
        // If no next hf found with valid block or timestamp return null
        if (nextHf === undefined) {
            return null;
        }
        const nextHfBlock = nextHf.timestamp ?? nextHf.block;
        if (nextHfBlock === null || nextHfBlock === undefined) {
            return null;
        }
        return BigInt(nextHfBlock);
    }
    /**
     * Internal helper function to calculate a fork hash
     * @param hardfork Hardfork name
     * @param genesisHash Genesis block hash of the chain
     * @returns Fork hash as hex string
     */
    _calcForkHash(hardfork, genesisHash) {
        let hfBytes = new Uint8Array(0);
        let prevBlockOrTime = 0;
        for (const hf of this.hardforks()) {
            const { block, timestamp, name } = hf;
            // Timestamp to be used for timestamp based hfs even if we may bundle
            // block number with them retrospectively
            let blockOrTime = timestamp ?? block;
            blockOrTime = blockOrTime !== null ? Number(blockOrTime) : null;
            // Skip for chainstart (0), not applied HFs (null) and
            // when already applied on same blockOrTime HFs
            // and on the merge since forkhash doesn't change on merge hf
            if (typeof blockOrTime === 'number' &&
                blockOrTime !== 0 &&
                blockOrTime !== prevBlockOrTime &&
                name !== enums_js_1.Hardfork.Paris) {
                const hfBlockBytes = (0, ethereumjs_util_1.hexToBytes)('0x' + blockOrTime.toString(16).padStart(16, '0'));
                hfBytes = (0, ethereumjs_util_1.concatBytes)(hfBytes, hfBlockBytes);
                prevBlockOrTime = blockOrTime;
            }
            if (hf.name === hardfork)
                break;
        }
        const inputBytes = (0, ethereumjs_util_1.concatBytes)(genesisHash, hfBytes);
        // CRC32 delivers result as signed (negative) 32-bit integer,
        // convert to hex string
        const forkhash = (0, ethereumjs_util_1.bytesToHex)((0, ethereumjs_util_1.intToBytes)((0, crc_js_1.crc32)(inputBytes) >>> 0));
        return forkhash;
    }
    /**
     * Returns an eth/64 compliant fork hash (EIP-2124)
     * @param hardfork Hardfork name, optional if HF set
     * @param genesisHash Genesis block hash of the chain, optional if already defined and not needed to be calculated
     */
    forkHash(hardfork, genesisHash) {
        hardfork = hardfork ?? this._hardfork;
        const data = this._getHardfork(hardfork);
        if (data === null ||
            (data?.block === null && data?.timestamp === undefined && data?.ttd === undefined)) {
            const msg = 'No fork hash calculation possible for future hardfork';
            throw new Error(msg);
        }
        if (data?.forkHash !== null && data?.forkHash !== undefined) {
            return data.forkHash;
        }
        if (!genesisHash)
            throw new Error('genesisHash required for forkHash calculation');
        return this._calcForkHash(hardfork, genesisHash);
    }
    /**
     *
     * @param forkHash Fork hash as a hex string
     * @returns Array with hardfork data (name, block, forkHash)
     */
    hardforkForForkHash(forkHash) {
        const resArray = this.hardforks().filter((hf) => {
            return hf.forkHash === forkHash;
        });
        return resArray.length >= 1 ? resArray[resArray.length - 1] : null;
    }
    /**
     * Sets any missing forkHashes on the passed-in {@link Common} instance
     * @param common The {@link Common} to set the forkHashes for
     * @param genesisHash The genesis block hash
     */
    setForkHashes(genesisHash) {
        for (const hf of this.hardforks()) {
            const blockOrTime = hf.timestamp ?? hf.block;
            if ((hf.forkHash === null || hf.forkHash === undefined) &&
                ((blockOrTime !== null && blockOrTime !== undefined) || typeof hf.ttd !== 'undefined')) {
                hf.forkHash = this.forkHash(hf.name, genesisHash);
            }
        }
    }
    /**
     * Returns the Genesis parameters of the current chain
     * @returns Genesis dictionary
     */
    genesis() {
        return this._chainParams.genesis;
    }
    /**
     * Returns the hardforks for current chain
     * @returns {Array} Array with arrays of hardforks
     */
    hardforks() {
        return this._chainParams.hardforks;
    }
    /**
     * Returns bootstrap nodes for the current chain
     * @returns {Dictionary} Dict with bootstrap nodes
     */
    bootstrapNodes() {
        return this._chainParams.bootstrapNodes;
    }
    /**
     * Returns DNS networks for the current chain
     * @returns {String[]} Array of DNS ENR urls
     */
    dnsNetworks() {
        return this._chainParams.dnsNetworks;
    }
    /**
     * Returns the hardfork set
     * @returns Hardfork name
     */
    hardfork() {
        return this._hardfork;
    }
    /**
     * Returns the Id of current chain
     * @returns chain Id
     */
    chainId() {
        return BigInt(this._chainParams.chainId);
    }
    /**
     * Returns the name of current chain
     * @returns chain name (lower case)
     */
    chainName() {
        return this._chainParams.name;
    }
    /**
     * Returns the Id of current network
     * @returns network Id
     */
    networkId() {
        return BigInt(this._chainParams.networkId);
    }
    /**
     * Returns the additionally activated EIPs
     * (by using the `eips` constructor option)
     * @returns List of EIPs
     */
    eips() {
        return this._eips;
    }
    /**
     * Returns the consensus type of the network
     * Possible values: "pow"|"poa"|"pos"
     *
     * Note: This value can update along a Hardfork.
     */
    consensusType() {
        const hardfork = this.hardfork();
        let value;
        for (const hfChanges of this.HARDFORK_CHANGES) {
            if ('consensus' in hfChanges[1]) {
                value = hfChanges[1]['consensus']['type'];
            }
            if (hfChanges[0] === hardfork)
                break;
        }
        return value ?? this._chainParams['consensus']['type'];
    }
    /**
     * Returns the concrete consensus implementation
     * algorithm or protocol for the network
     * e.g. "ethash" for "pow" consensus type,
     * "clique" for "poa" consensus type or
     * "casper" for "pos" consensus type.
     *
     * Note: This value can update along a Hardfork.
     */
    consensusAlgorithm() {
        const hardfork = this.hardfork();
        let value;
        for (const hfChanges of this.HARDFORK_CHANGES) {
            if ('consensus' in hfChanges[1]) {
                value = hfChanges[1]['consensus']['algorithm'];
            }
            if (hfChanges[0] === hardfork)
                break;
        }
        return value ?? this._chainParams['consensus']['algorithm'];
    }
    /**
     * Returns a dictionary with consensus configuration
     * parameters based on the consensus algorithm
     *
     * Expected returns (parameters must be present in
     * the respective chain json files):
     *
     * ethash: empty object
     * clique: period, epoch
     * casper: empty object
     *
     * Note: This value can update along a Hardfork.
     */
    consensusConfig() {
        const hardfork = this.hardfork();
        let value;
        for (const hfChanges of this.HARDFORK_CHANGES) {
            if ('consensus' in hfChanges[1]) {
                // The config parameter is named after the respective consensus algorithm
                const config = hfChanges[1];
                const algorithm = config['consensus']['algorithm'];
                value = config['consensus'][algorithm];
            }
            if (hfChanges[0] === hardfork)
                break;
        }
        return (value ?? this._chainParams['consensus'][this.consensusAlgorithm()] ?? {});
    }
    /**
     * Returns a deep copy of this {@link Common} instance.
     */
    copy() {
        const copy = Object.assign(Object.create(Object.getPrototypeOf(this)), this);
        copy.events = new events_1.EventEmitter();
        return copy;
    }
    static getInitializedChains(customChains) {
        const names = {};
        for (const [name, id] of Object.entries(enums_js_1.Chain)) {
            names[id] = name.toLowerCase();
        }
        const chains = { ...chains_js_1.chains };
        if (customChains) {
            for (const chain of customChains) {
                const { name } = chain;
                names[chain.chainId.toString()] = name;
                chains[name] = chain;
            }
        }
        chains.names = names;
        return chains;
    }
}
exports.Common = Common;
//# sourceMappingURL=common.js.map