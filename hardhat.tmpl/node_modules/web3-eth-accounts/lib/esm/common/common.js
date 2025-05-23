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
import pkg from 'crc-32';
import { EventEmitter, bytesToHex, hexToBytes, uint8ArrayConcat } from 'web3-utils';
import { TypeOutput } from './types.js';
import { intToUint8Array, toType, parseGethGenesis } from './utils.js';
import goerli from './chains/goerli.js';
import mainnet from './chains/mainnet.js';
import sepolia from './chains/sepolia.js';
import { EIPs } from './eips/index.js';
import { Chain, CustomChain, Hardfork } from './enums.js';
import { hardforks as HARDFORK_SPECS } from './hardforks/index.js';
const { buf: crc32Uint8Array } = pkg;
/**
 * Common class to access chain and hardfork parameters and to provide
 * a unified and shared view on the network and hardfork state.
 *
 * Use the {@link Common.custom} static constructor for creating simple
 * custom chain {@link Common} objects (more complete custom chain setups
 * can be created via the main constructor and the {@link CommonOpts.customChains} parameter).
 */
export class Common extends EventEmitter {
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
     * the `web3-utils/tx` library to a Layer-2 chain).
     *
     * @param chainParamsOrName Custom parameter dict (`name` will default to `custom-chain`) or string with name of a supported custom chain
     * @param opts Custom chain options to set the {@link CustomCommonOpts.baseChain}, selected {@link CustomCommonOpts.hardfork} and others
     */
    static custom(chainParamsOrName, opts = {}) {
        var _a;
        const baseChain = (_a = opts.baseChain) !== null && _a !== void 0 ? _a : 'mainnet';
        const standardChainParams = Object.assign({}, Common._getChainParams(baseChain));
        standardChainParams.name = 'custom-chain';
        if (typeof chainParamsOrName !== 'string') {
            return new Common(Object.assign({ chain: Object.assign(Object.assign({}, standardChainParams), chainParamsOrName) }, opts));
        }
        if (chainParamsOrName === CustomChain.PolygonMainnet) {
            return Common.custom({
                name: CustomChain.PolygonMainnet,
                chainId: 137,
                networkId: 137,
            }, opts);
        }
        if (chainParamsOrName === CustomChain.PolygonMumbai) {
            return Common.custom({
                name: CustomChain.PolygonMumbai,
                chainId: 80001,
                networkId: 80001,
            }, opts);
        }
        if (chainParamsOrName === CustomChain.ArbitrumRinkebyTestnet) {
            return Common.custom({
                name: CustomChain.ArbitrumRinkebyTestnet,
                chainId: 421611,
                networkId: 421611,
            }, opts);
        }
        if (chainParamsOrName === CustomChain.ArbitrumOne) {
            return Common.custom({
                name: CustomChain.ArbitrumOne,
                chainId: 42161,
                networkId: 42161,
            }, opts);
        }
        if (chainParamsOrName === CustomChain.xDaiChain) {
            return Common.custom({
                name: CustomChain.xDaiChain,
                chainId: 100,
                networkId: 100,
            }, opts);
        }
        if (chainParamsOrName === CustomChain.OptimisticKovan) {
            return Common.custom({
                name: CustomChain.OptimisticKovan,
                chainId: 69,
                networkId: 69,
            }, Object.assign({ hardfork: Hardfork.Berlin }, opts));
        }
        if (chainParamsOrName === CustomChain.OptimisticEthereum) {
            return Common.custom({
                name: CustomChain.OptimisticEthereum,
                chainId: 10,
                networkId: 10,
            }, Object.assign({ hardfork: Hardfork.Berlin }, opts));
        }
        // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
        throw new Error(`Custom chain ${chainParamsOrName} not supported`);
    }
    /**
     * Static method to load and set common from a geth genesis json
     * @param genesisJson json of geth configuration
     * @param { chain, eips, genesisHash, hardfork, mergeForkIdPostMerge } to further configure the common instance
     * @returns Common
     */
    static fromGethGenesis(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    genesisJson, { chain, eips, genesisHash, hardfork, mergeForkIdPostMerge }) {
        var _a;
        const genesisParams = parseGethGenesis(genesisJson, chain, mergeForkIdPostMerge);
        const common = new Common({
            chain: (_a = genesisParams.name) !== null && _a !== void 0 ? _a : 'custom',
            customChains: [genesisParams],
            eips,
            hardfork: hardfork !== null && hardfork !== void 0 ? hardfork : genesisParams.hardfork,
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
        const initializedChains = this._getInitializedChains();
        return Boolean(initializedChains.names[chainId.toString()]);
    }
    static _getChainParams(_chain, customChains) {
        let chain = _chain;
        const initializedChains = this._getInitializedChains(customChains);
        if (typeof chain === 'number' || typeof chain === 'bigint') {
            chain = chain.toString();
            if (initializedChains.names[chain]) {
                const name = initializedChains.names[chain];
                return initializedChains[name];
            }
            throw new Error(`Chain with ID ${chain} not supported`);
        }
        if (initializedChains[chain] !== undefined) {
            return initializedChains[chain];
        }
        throw new Error(`Chain with name ${chain} not supported`);
    }
    constructor(opts) {
        var _a, _b;
        super();
        this._eips = [];
        this._customChains = (_a = opts.customChains) !== null && _a !== void 0 ? _a : [];
        this._chainParams = this.setChain(opts.chain);
        this.DEFAULT_HARDFORK = (_b = this._chainParams.defaultHardfork) !== null && _b !== void 0 ? _b : Hardfork.Merge;
        // Assign hardfork changes in the sequence of the applied hardforks
        this.HARDFORK_CHANGES = this.hardforks().map(hf => [
            hf.name,
            HARDFORK_SPECS[hf.name],
        ]);
        this._hardfork = this.DEFAULT_HARDFORK;
        if (opts.hardfork !== undefined) {
            this.setHardfork(opts.hardfork);
        }
        if (opts.eips) {
            this.setEIPs(opts.eips);
        }
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
                    this.emit('hardforkChanged', hardfork);
                }
                existing = true;
            }
        }
        if (!existing) {
            throw new Error(`Hardfork with name ${hardfork} not supported`);
        }
    }
    /**
     * Returns the hardfork based on the block number or an optional
     * total difficulty (Merge HF) provided.
     *
     * An optional TD takes precedence in case the corresponding HF block
     * is set to `null` or otherwise needs to match (if not an error
     * will be thrown).
     *
     * @param blockNumber
     * @param td : total difficulty of the parent block (for block hf) OR of the chain latest (for chain hf)
     * @param timestamp: timestamp in seconds at which block was/is to be minted
     * @returns The name of the HF
     */
    getHardforkByBlockNumber(_blockNumber, _td, _timestamp) {
        const blockNumber = toType(_blockNumber, TypeOutput.BigInt);
        const td = toType(_td, TypeOutput.BigInt);
        const timestamp = toType(_timestamp, TypeOutput.Number);
        // Filter out hardforks with no block number, no ttd or no timestamp (i.e. unapplied hardforks)
        const hfs = this.hardforks().filter(hf => 
        // eslint-disable-next-line no-null/no-null
        hf.block !== null ||
            // eslint-disable-next-line no-null/no-null
            (hf.ttd !== null && hf.ttd !== undefined) ||
            hf.timestamp !== undefined);
        // eslint-disable-next-line no-null/no-null
        const mergeIndex = hfs.findIndex(hf => hf.ttd !== null && hf.ttd !== undefined);
        const doubleTTDHF = hfs
            .slice(mergeIndex + 1)
            // eslint-disable-next-line no-null/no-null
            .findIndex(hf => hf.ttd !== null && hf.ttd !== undefined);
        if (doubleTTDHF >= 0) {
            throw Error(`More than one merge hardforks found with ttd specified`);
        }
        // Find the first hardfork that has a block number greater than `blockNumber`
        // (skips the merge hardfork since it cannot have a block number specified).
        // If timestamp is not provided, it also skips timestamps hardforks to continue
        // discovering/checking number hardforks.
        let hfIndex = hfs.findIndex(hf => 
        // eslint-disable-next-line no-null/no-null
        (hf.block !== null && hf.block > blockNumber) ||
            (timestamp !== undefined && Number(hf.timestamp) > timestamp));
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
                // eslint-disable-next-line no-null/no-null
                .findIndex(hf => hf.block !== null || hf.ttd !== undefined);
            hfIndex -= stepBack;
        }
        // Move hfIndex one back to arrive at candidate hardfork
        hfIndex -= 1;
        // If the timestamp was not provided, we could have skipped timestamp hardforks to look for number
        // hardforks. so it will now be needed to rollback
        // eslint-disable-next-line no-null/no-null
        if (hfs[hfIndex].block === null && hfs[hfIndex].timestamp === undefined) {
            // We're on the merge hardfork.  Let's check the TTD
            // eslint-disable-next-line no-null/no-null
            if (td === undefined || td === null || BigInt(hfs[hfIndex].ttd) > td) {
                // Merge ttd greater than current td so we're on hardfork before merge
                hfIndex -= 1;
            }
            // eslint-disable-next-line no-null/no-null
        }
        else if (mergeIndex >= 0 && td !== undefined && td !== null) {
            if (hfIndex >= mergeIndex && BigInt(hfs[mergeIndex].ttd) > td) {
                throw Error('Maximum HF determined by total difficulty is lower than the block number HF');
            }
            else if (hfIndex < mergeIndex && BigInt(hfs[mergeIndex].ttd) <= td) {
                throw Error('HF determined by block number is lower than the minimum total difficulty HF');
            }
        }
        const hfStartIndex = hfIndex;
        // Move the hfIndex to the end of the hardforks that might be scheduled on the same block/timestamp
        // This won't anyway be the case with Merge hfs
        for (; hfIndex < hfs.length - 1; hfIndex += 1) {
            // break out if hfIndex + 1 is not scheduled at hfIndex
            if (hfs[hfIndex].block !== hfs[hfIndex + 1].block ||
                hfs[hfIndex].timestamp !== hfs[hfIndex + 1].timestamp) {
                break;
            }
        }
        if (timestamp) {
            const minTimeStamp = hfs
                .slice(0, hfStartIndex)
                .reduce((acc, hf) => { var _a; return Math.max(Number((_a = hf.timestamp) !== null && _a !== void 0 ? _a : '0'), acc); }, 0);
            if (minTimeStamp > timestamp) {
                throw Error(`Maximum HF determined by timestamp is lower than the block number/ttd HF`);
            }
            const maxTimeStamp = hfs
                .slice(hfIndex + 1)
                .reduce((acc, hf) => { var _a; return Math.min(Number((_a = hf.timestamp) !== null && _a !== void 0 ? _a : timestamp), acc); }, timestamp);
            if (maxTimeStamp < timestamp) {
                throw Error(`Maximum HF determined by block number/ttd is lower than timestamp HF`);
            }
        }
        const hardfork = hfs[hfIndex];
        return hardfork.name;
    }
    /**
     * Sets a new hardfork based on the block number or an optional
     * total difficulty (Merge HF) provided.
     *
     * An optional TD takes precedence in case the corresponding HF block
     * is set to `null` or otherwise needs to match (if not an error
     * will be thrown).
     *
     * @param blockNumber
     * @param td
     * @param timestamp
     * @returns The name of the HF set
     */
    setHardforkByBlockNumber(blockNumber, td, timestamp) {
        const hardfork = this.getHardforkByBlockNumber(blockNumber, td, timestamp);
        this.setHardfork(hardfork);
        return hardfork;
    }
    /**
     * Internal helper function, returns the params for the given hardfork for the chain set
     * @param hardfork Hardfork name
     * @returns Dictionary with hardfork params or null if hardfork not on chain
     */
    // eslint-disable-next-line @typescript-eslint/ban-types
    _getHardfork(hardfork) {
        const hfs = this.hardforks();
        for (const hf of hfs) {
            if (hf.name === hardfork)
                return hf;
        }
        // eslint-disable-next-line no-null/no-null
        return null;
    }
    /**
     * Sets the active EIPs
     * @param eips
     */
    setEIPs(eips = []) {
        for (const eip of eips) {
            if (!(eip in EIPs)) {
                throw new Error(`${eip} not supported`);
            }
            // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-argument
            const minHF = this.gteHardfork(EIPs[eip].minimumHardfork);
            if (!minHF) {
                throw new Error(
                // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
                `${eip} cannot be activated on hardfork ${this.hardfork()}, minimumHardfork: ${minHF}`);
            }
            // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
            if (EIPs[eip].requiredEIPs !== undefined) {
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                for (const elem of EIPs[eip].requiredEIPs) {
                    // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
                    if (!(eips.includes(elem) || this.isActivatedEIP(elem))) {
                        throw new Error(
                        // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
                        `${eip} requires EIP ${elem}, but is not included in the EIP list`);
                    }
                }
            }
        }
        this._eips = eips;
    }
    /**
     * Returns a parameter for the current chain setup
     *
     * If the parameter is present in an EIP, the EIP always takes precedence.
     * Otherwise the parameter if taken from the latest applied HF with
     * a change on the respective parameter.
     *
     * @param topic Parameter topic ('gasConfig', 'gasPrices', 'vm', 'pow')
     * @param name Parameter name (e.g. 'minGasLimit' for 'gasConfig' topic)
     * @returns The value requested or `BigInt(0)` if not found
     */
    param(topic, name) {
        // TODO: consider the case that different active EIPs
        // can change the same parameter
        let value;
        for (const eip of this._eips) {
            value = this.paramByEIP(topic, name, eip);
            if (value !== undefined)
                return value;
        }
        return this.paramByHardfork(topic, name, this._hardfork);
    }
    /**
     * Returns the parameter corresponding to a hardfork
     * @param topic Parameter topic ('gasConfig', 'gasPrices', 'vm', 'pow')
     * @param name Parameter name (e.g. 'minGasLimit' for 'gasConfig' topic)
     * @param hardfork Hardfork name
     * @returns The value requested or `BigInt(0)` if not found
     */
    paramByHardfork(topic, name, hardfork) {
        // eslint-disable-next-line no-null/no-null
        let value = null;
        for (const hfChanges of this.HARDFORK_CHANGES) {
            // EIP-referencing HF file (e.g. berlin.json)
            if ('eips' in hfChanges[1]) {
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment
                const hfEIPs = hfChanges[1].eips;
                for (const eip of hfEIPs) {
                    // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
                    const valueEIP = this.paramByEIP(topic, name, eip);
                    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
                    value = typeof valueEIP === 'bigint' ? valueEIP : value;
                }
                // Parameter-inlining HF file (e.g. istanbul.json)
            }
            else {
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                if (hfChanges[1][topic] === undefined) {
                    throw new Error(`Topic ${topic} not defined`);
                }
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                if (hfChanges[1][topic][name] !== undefined) {
                    // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment
                    value = hfChanges[1][topic][name].v;
                }
            }
            if (hfChanges[0] === hardfork)
                break;
        }
        // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
        return BigInt(value !== null && value !== void 0 ? value : 0);
    }
    /**
     * Returns a parameter corresponding to an EIP
     * @param topic Parameter topic ('gasConfig', 'gasPrices', 'vm', 'pow')
     * @param name Parameter name (e.g. 'minGasLimit' for 'gasConfig' topic)
     * @param eip Number of the EIP
     * @returns The value requested or `undefined` if not found
     */
    // eslint-disable-next-line class-methods-use-this
    paramByEIP(topic, name, eip) {
        if (!(eip in EIPs)) {
            throw new Error(`${eip} not supported`);
        }
        // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
        const eipParams = EIPs[eip];
        if (!(topic in eipParams)) {
            throw new Error(`Topic ${topic} not defined`);
        }
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
        if (eipParams[topic][name] === undefined) {
            return undefined;
        }
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment
        const value = eipParams[topic][name].v;
        // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
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
        const hardfork = this.getHardforkByBlockNumber(blockNumber, td, timestamp);
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
        if (this.eips().includes(eip)) {
            return true;
        }
        for (const hfChanges of this.HARDFORK_CHANGES) {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
            const hf = hfChanges[1];
            // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-argument
            if (this.gteHardfork(hf.name) && 'eips' in hf) {
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                if (hf.eips.includes(eip)) {
                    return true;
                }
            }
        }
        return false;
    }
    /**
     * Checks if set or provided hardfork is active on block number
     * @param hardfork Hardfork name or null (for HF set)
     * @param blockNumber
     * @returns True if HF is active on block number
     */
    hardforkIsActiveOnBlock(
    // eslint-disable-next-line @typescript-eslint/ban-types
    _hardfork, _blockNumber) {
        const blockNumber = toType(_blockNumber, TypeOutput.BigInt);
        const hardfork = _hardfork !== null && _hardfork !== void 0 ? _hardfork : this._hardfork;
        const hfBlock = this.hardforkBlock(hardfork);
        if (typeof hfBlock === 'bigint' && hfBlock !== BigInt(0) && blockNumber >= hfBlock) {
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
        // eslint-disable-next-line no-null/no-null
        return this.hardforkIsActiveOnBlock(null, blockNumber);
    }
    /**
     * Sequence based check if given or set HF1 is greater than or equal HF2
     * @param hardfork1 Hardfork name or null (if set)
     * @param hardfork2 Hardfork name
     * @param opts Hardfork options
     * @returns True if HF1 gte HF2
     */
    hardforkGteHardfork(
    // eslint-disable-next-line @typescript-eslint/ban-types
    _hardfork1, hardfork2) {
        const hardfork1 = _hardfork1 !== null && _hardfork1 !== void 0 ? _hardfork1 : this._hardfork;
        const hardforks = this.hardforks();
        let posHf1 = -1;
        let posHf2 = -1;
        let index = 0;
        for (const hf of hardforks) {
            if (hf.name === hardfork1)
                posHf1 = index;
            if (hf.name === hardfork2)
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
        // eslint-disable-next-line no-null/no-null
        return this.hardforkGteHardfork(null, hardfork);
    }
    /**
     * Returns the hardfork change block for hardfork provided or set
     * @param hardfork Hardfork name, optional if HF set
     * @returns Block number or null if unscheduled
     */
    // eslint-disable-next-line @typescript-eslint/ban-types
    hardforkBlock(_hardfork) {
        var _a;
        const hardfork = _hardfork !== null && _hardfork !== void 0 ? _hardfork : this._hardfork;
        const block = (_a = this._getHardfork(hardfork)) === null || _a === void 0 ? void 0 : _a.block;
        // eslint-disable-next-line no-null/no-null
        if (block === undefined || block === null) {
            // eslint-disable-next-line no-null/no-null
            return null;
        }
        return BigInt(block);
    }
    // eslint-disable-next-line @typescript-eslint/ban-types
    hardforkTimestamp(_hardfork) {
        var _a;
        const hardfork = _hardfork !== null && _hardfork !== void 0 ? _hardfork : this._hardfork;
        const timestamp = (_a = this._getHardfork(hardfork)) === null || _a === void 0 ? void 0 : _a.timestamp;
        // eslint-disable-next-line no-null/no-null
        if (timestamp === undefined || timestamp === null) {
            // eslint-disable-next-line no-null/no-null
            return null;
        }
        return BigInt(timestamp);
    }
    /**
     * Returns the hardfork change block for eip
     * @param eip EIP number
     * @returns Block number or null if unscheduled
     */
    // eslint-disable-next-line @typescript-eslint/ban-types
    eipBlock(eip) {
        for (const hfChanges of this.HARDFORK_CHANGES) {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
            const hf = hfChanges[1];
            if ('eips' in hf) {
                // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions, @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call
                if (hf.eips.includes(eip)) {
                    return this.hardforkBlock(typeof hfChanges[0] === 'number' ? String(hfChanges[0]) : hfChanges[0]);
                }
            }
        }
        // eslint-disable-next-line no-null/no-null
        return null;
    }
    /**
     * Returns the hardfork change total difficulty (Merge HF) for hardfork provided or set
     * @param hardfork Hardfork name, optional if HF set
     * @returns Total difficulty or null if no set
     */
    // eslint-disable-next-line @typescript-eslint/ban-types
    hardforkTTD(_hardfork) {
        var _a;
        const hardfork = _hardfork !== null && _hardfork !== void 0 ? _hardfork : this._hardfork;
        const ttd = (_a = this._getHardfork(hardfork)) === null || _a === void 0 ? void 0 : _a.ttd;
        // eslint-disable-next-line no-null/no-null
        if (ttd === undefined || ttd === null) {
            // eslint-disable-next-line no-null/no-null
            return null;
        }
        return BigInt(ttd);
    }
    /**
     * True if block number provided is the hardfork (given or set) change block
     * @param blockNumber Number of the block to check
     * @param hardfork Hardfork name, optional if HF set
     * @returns True if blockNumber is HF block
     * @deprecated
     */
    isHardforkBlock(_blockNumber, _hardfork) {
        const blockNumber = toType(_blockNumber, TypeOutput.BigInt);
        const hardfork = _hardfork !== null && _hardfork !== void 0 ? _hardfork : this._hardfork;
        const block = this.hardforkBlock(hardfork);
        return typeof block === 'bigint' && block !== BigInt(0) ? block === blockNumber : false;
    }
    /**
     * Returns the change block for the next hardfork after the hardfork provided or set
     * @param hardfork Hardfork name, optional if HF set
     * @returns Block timestamp, number or null if not available
     */
    // eslint-disable-next-line @typescript-eslint/ban-types
    nextHardforkBlockOrTimestamp(_hardfork) {
        var _a, _b, _c;
        const hardfork = (_a = _hardfork) !== null && _a !== void 0 ? _a : this._hardfork;
        const hfs = this.hardforks();
        let hfIndex = hfs.findIndex(hf => hf.name === hardfork);
        // If the current hardfork is merge, go one behind as merge hf is not part of these
        // calcs even if the merge hf block is set
        if (hardfork === Hardfork.Merge) {
            hfIndex -= 1;
        }
        // Hardfork not found
        if (hfIndex < 0) {
            // eslint-disable-next-line no-null/no-null
            return null;
        }
        let currHfTimeOrBlock = (_b = hfs[hfIndex].timestamp) !== null && _b !== void 0 ? _b : hfs[hfIndex].block;
        currHfTimeOrBlock =
            // eslint-disable-next-line no-null/no-null
            currHfTimeOrBlock !== null && currHfTimeOrBlock !== undefined
                ? Number(currHfTimeOrBlock)
                : // eslint-disable-next-line no-null/no-null
                    null;
        const nextHf = hfs.slice(hfIndex + 1).find(hf => {
            var _a;
            let hfTimeOrBlock = (_a = hf.timestamp) !== null && _a !== void 0 ? _a : hf.block;
            hfTimeOrBlock =
                // eslint-disable-next-line no-null/no-null
                hfTimeOrBlock !== null && hfTimeOrBlock !== undefined
                    ? Number(hfTimeOrBlock)
                    : // eslint-disable-next-line no-null/no-null
                        null;
            return (hf.name !== Hardfork.Merge &&
                // eslint-disable-next-line no-null/no-null
                hfTimeOrBlock !== null &&
                hfTimeOrBlock !== undefined &&
                hfTimeOrBlock !== currHfTimeOrBlock);
        });
        // If no next hf found with valid block or timestamp return null
        if (nextHf === undefined) {
            // eslint-disable-next-line no-null/no-null
            return null;
        }
        const nextHfBlock = (_c = nextHf.timestamp) !== null && _c !== void 0 ? _c : nextHf.block;
        // eslint-disable-next-line no-null/no-null
        if (nextHfBlock === null || nextHfBlock === undefined) {
            // eslint-disable-next-line no-null/no-null
            return null;
        }
        return BigInt(nextHfBlock);
    }
    /**
     * Returns the change block for the next hardfork after the hardfork provided or set
     * @param hardfork Hardfork name, optional if HF set
     * @returns Block number or null if not available
     * @deprecated
     */
    // eslint-disable-next-line @typescript-eslint/ban-types
    nextHardforkBlock(_hardfork) {
        var _a;
        const hardfork = (_a = _hardfork) !== null && _a !== void 0 ? _a : this._hardfork;
        let hfBlock = this.hardforkBlock(hardfork);
        // If this is a merge hardfork with block not set, then we fallback to previous hardfork
        // to find the nextHardforkBlock
        // eslint-disable-next-line no-null/no-null
        if (hfBlock === null && hardfork === Hardfork.Merge) {
            const hfs = this.hardforks();
            // eslint-disable-next-line no-null/no-null
            const mergeIndex = hfs.findIndex(hf => hf.ttd !== null && hf.ttd !== undefined);
            if (mergeIndex < 0) {
                throw Error(`Merge hardfork should have been found`);
            }
            hfBlock = this.hardforkBlock(hfs[mergeIndex - 1].name);
        }
        // eslint-disable-next-line no-null/no-null
        if (hfBlock === null) {
            // eslint-disable-next-line no-null/no-null
            return null;
        }
        // Next fork block number or null if none available
        // Logic: if accumulator is still null and on the first occurrence of
        // a block greater than the current hfBlock set the accumulator,
        // pass on the accumulator as the final result from this time on
        // eslint-disable-next-line no-null/no-null, @typescript-eslint/ban-types
        const nextHfBlock = this.hardforks().reduce((acc, hf) => {
            // We need to ignore the merge block in our next hardfork calc
            const block = BigInt(
            // eslint-disable-next-line no-null/no-null
            hf.block === null || (hf.ttd !== undefined && hf.ttd !== null) ? 0 : hf.block);
            // TypeScript can't seem to follow that the hfBlock is not null at this point
            // eslint-disable-next-line no-null/no-null
            return block > hfBlock && acc === null ? block : acc;
            // eslint-disable-next-line no-null/no-null
        }, null);
        return nextHfBlock;
    }
    /**
     * True if block number provided is the hardfork change block following the hardfork given or set
     * @param blockNumber Number of the block to check
     * @param hardfork Hardfork name, optional if HF set
     * @returns True if blockNumber is HF block
     * @deprecated
     */
    isNextHardforkBlock(_blockNumber, _hardfork) {
        const blockNumber = toType(_blockNumber, TypeOutput.BigInt);
        const hardfork = _hardfork !== null && _hardfork !== void 0 ? _hardfork : this._hardfork;
        // eslint-disable-next-line deprecation/deprecation
        const nextHardforkBlock = this.nextHardforkBlock(hardfork);
        // eslint-disable-next-line no-null/no-null
        return nextHardforkBlock === null ? false : nextHardforkBlock === blockNumber;
    }
    /**
     * Internal helper function to calculate a fork hash
     * @param hardfork Hardfork name
     * @param genesisHash Genesis block hash of the chain
     * @returns Fork hash as hex string
     */
    _calcForkHash(hardfork, genesisHash) {
        let hfUint8Array = new Uint8Array();
        let prevBlockOrTime = 0;
        for (const hf of this.hardforks()) {
            const { block, timestamp, name } = hf;
            // Timestamp to be used for timestamp based hfs even if we may bundle
            // block number with them retrospectively
            let blockOrTime = timestamp !== null && timestamp !== void 0 ? timestamp : block;
            // eslint-disable-next-line no-null/no-null
            blockOrTime = blockOrTime !== null ? Number(blockOrTime) : null;
            // Skip for chainstart (0), not applied HFs (null) and
            // when already applied on same blockOrTime HFs
            // and on the merge since forkhash doesn't change on merge hf
            if (typeof blockOrTime === 'number' &&
                blockOrTime !== 0 &&
                blockOrTime !== prevBlockOrTime &&
                name !== Hardfork.Merge) {
                const hfBlockUint8Array = hexToBytes(blockOrTime.toString(16).padStart(16, '0'));
                hfUint8Array = uint8ArrayConcat(hfUint8Array, hfBlockUint8Array);
                prevBlockOrTime = blockOrTime;
            }
            if (hf.name === hardfork)
                break;
        }
        const inputUint8Array = uint8ArrayConcat(genesisHash, hfUint8Array);
        // CRC32 delivers result as signed (negative) 32-bit integer,
        // convert to hex string
        // eslint-disable-next-line no-bitwise
        const forkhash = bytesToHex(intToUint8Array(crc32Uint8Array(inputUint8Array) >>> 0));
        return forkhash;
    }
    /**
     * Returns an eth/64 compliant fork hash (EIP-2124)
     * @param hardfork Hardfork name, optional if HF set
     * @param genesisHash Genesis block hash of the chain, optional if already defined and not needed to be calculated
     */
    forkHash(_hardfork, genesisHash) {
        const hardfork = _hardfork !== null && _hardfork !== void 0 ? _hardfork : this._hardfork;
        const data = this._getHardfork(hardfork);
        if (
        // eslint-disable-next-line no-null/no-null
        data === null ||
            // eslint-disable-next-line no-null/no-null
            ((data === null || data === void 0 ? void 0 : data.block) === null && (data === null || data === void 0 ? void 0 : data.timestamp) === undefined && (data === null || data === void 0 ? void 0 : data.ttd) === undefined)) {
            const msg = 'No fork hash calculation possible for future hardfork';
            throw new Error(msg);
        }
        // eslint-disable-next-line no-null/no-null
        if ((data === null || data === void 0 ? void 0 : data.forkHash) !== null && (data === null || data === void 0 ? void 0 : data.forkHash) !== undefined) {
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
    // eslint-disable-next-line @typescript-eslint/ban-types
    hardforkForForkHash(forkHash) {
        const resArray = this.hardforks().filter((hf) => hf.forkHash === forkHash);
        // eslint-disable-next-line no-null/no-null
        return resArray.length >= 1 ? resArray[resArray.length - 1] : null;
    }
    /**
     * Sets any missing forkHashes on the passed-in {@link Common} instance
     * @param common The {@link Common} to set the forkHashes for
     * @param genesisHash The genesis block hash
     */
    setForkHashes(genesisHash) {
        var _a;
        for (const hf of this.hardforks()) {
            const blockOrTime = (_a = hf.timestamp) !== null && _a !== void 0 ? _a : hf.block;
            if (
            // eslint-disable-next-line no-null/no-null
            (hf.forkHash === null || hf.forkHash === undefined) &&
                // eslint-disable-next-line no-null/no-null
                ((blockOrTime !== null && blockOrTime !== undefined) ||
                    typeof hf.ttd !== 'undefined')) {
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
     * Returns the active EIPs
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
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment
                value = hfChanges[1].consensus.type;
            }
            if (hfChanges[0] === hardfork)
                break;
        }
        // eslint-disable-next-line @typescript-eslint/no-unsafe-return
        return value !== null && value !== void 0 ? value : this._chainParams.consensus.type;
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
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment
                value = hfChanges[1].consensus.algorithm;
            }
            if (hfChanges[0] === hardfork)
                break;
        }
        // eslint-disable-next-line @typescript-eslint/no-unsafe-return
        return value !== null && value !== void 0 ? value : this._chainParams.consensus.algorithm;
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
        var _a;
        const hardfork = this.hardfork();
        let value;
        for (const hfChanges of this.HARDFORK_CHANGES) {
            if ('consensus' in hfChanges[1]) {
                // The config parameter is named after the respective consensus algorithm
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment
                value = hfChanges[1].consensus[hfChanges[1].consensus.algorithm];
            }
            if (hfChanges[0] === hardfork)
                break;
        }
        // eslint-disable-next-line @typescript-eslint/no-unsafe-return
        return ((_a = value !== null && value !== void 0 ? value : this._chainParams.consensus[this.consensusAlgorithm()]) !== null && _a !== void 0 ? _a : {});
    }
    /**
     * Returns a deep copy of this {@link Common} instance.
     */
    copy() {
        // eslint-disable-next-line @typescript-eslint/no-unsafe-argument, @typescript-eslint/no-unsafe-assignment
        const copy = Object.assign(Object.create(Object.getPrototypeOf(this)), this);
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call
        copy.removeAllListeners();
        // eslint-disable-next-line @typescript-eslint/no-unsafe-return
        return copy;
    }
    static _getInitializedChains(customChains) {
        const names = {};
        for (const [name, id] of Object.entries(Chain)) {
            names[id] = name.toLowerCase();
        }
        const chains = { mainnet, goerli, sepolia };
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
//# sourceMappingURL=common.js.map