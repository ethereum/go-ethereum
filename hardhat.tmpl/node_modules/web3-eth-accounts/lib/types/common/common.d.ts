import { EventEmitter } from 'web3-utils';
import type { Numbers } from 'web3-types';
import type { ConsensusAlgorithm, ConsensusType } from './enums.js';
import { Chain, CustomChain, Hardfork } from './enums.js';
import type { BootstrapNodeConfig, CasperConfig, ChainConfig, ChainsConfig, CliqueConfig, CommonOpts, CustomCommonOpts, EthashConfig, GenesisBlockConfig, GethConfigOpts, HardforkConfig } from './types.js';
/**
 * Common class to access chain and hardfork parameters and to provide
 * a unified and shared view on the network and hardfork state.
 *
 * Use the {@link Common.custom} static constructor for creating simple
 * custom chain {@link Common} objects (more complete custom chain setups
 * can be created via the main constructor and the {@link CommonOpts.customChains} parameter).
 */
export declare class Common extends EventEmitter {
    readonly DEFAULT_HARDFORK: string | Hardfork;
    private _chainParams;
    private _hardfork;
    private _eips;
    private readonly _customChains;
    private readonly HARDFORK_CHANGES;
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
    static custom(chainParamsOrName: Partial<ChainConfig> | CustomChain, opts?: CustomCommonOpts): Common;
    /**
     * Static method to load and set common from a geth genesis json
     * @param genesisJson json of geth configuration
     * @param { chain, eips, genesisHash, hardfork, mergeForkIdPostMerge } to further configure the common instance
     * @returns Common
     */
    static fromGethGenesis(genesisJson: any, { chain, eips, genesisHash, hardfork, mergeForkIdPostMerge }: GethConfigOpts): Common;
    /**
     * Static method to determine if a {@link chainId} is supported as a standard chain
     * @param chainId bigint id (`1`) of a standard chain
     * @returns boolean
     */
    static isSupportedChainId(chainId: bigint): boolean;
    private static _getChainParams;
    constructor(opts: CommonOpts);
    /**
     * Sets the chain
     * @param chain String ('mainnet') or Number (1) chain representation.
     *              Or, a Dictionary of chain parameters for a private network.
     * @returns The dictionary with parameters set as chain
     */
    setChain(chain: string | number | Chain | bigint | object): ChainConfig;
    /**
     * Sets the hardfork to get params for
     * @param hardfork String identifier (e.g. 'byzantium') or {@link Hardfork} enum
     */
    setHardfork(hardfork: string | Hardfork): void;
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
    getHardforkByBlockNumber(_blockNumber: Numbers, _td?: Numbers, _timestamp?: Numbers): string;
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
    setHardforkByBlockNumber(blockNumber: Numbers, td?: Numbers, timestamp?: Numbers): string;
    /**
     * Internal helper function, returns the params for the given hardfork for the chain set
     * @param hardfork Hardfork name
     * @returns Dictionary with hardfork params or null if hardfork not on chain
     */
    _getHardfork(hardfork: string | Hardfork): HardforkConfig | null;
    /**
     * Sets the active EIPs
     * @param eips
     */
    setEIPs(eips?: number[]): void;
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
    param(topic: string, name: string): bigint;
    /**
     * Returns the parameter corresponding to a hardfork
     * @param topic Parameter topic ('gasConfig', 'gasPrices', 'vm', 'pow')
     * @param name Parameter name (e.g. 'minGasLimit' for 'gasConfig' topic)
     * @param hardfork Hardfork name
     * @returns The value requested or `BigInt(0)` if not found
     */
    paramByHardfork(topic: string, name: string, hardfork: string | Hardfork): bigint;
    /**
     * Returns a parameter corresponding to an EIP
     * @param topic Parameter topic ('gasConfig', 'gasPrices', 'vm', 'pow')
     * @param name Parameter name (e.g. 'minGasLimit' for 'gasConfig' topic)
     * @param eip Number of the EIP
     * @returns The value requested or `undefined` if not found
     */
    paramByEIP(topic: string, name: string, eip: number): bigint | undefined;
    /**
     * Returns a parameter for the hardfork active on block number or
     * optional provided total difficulty (Merge HF)
     * @param topic Parameter topic
     * @param name Parameter name
     * @param blockNumber Block number
     * @param td Total difficulty
     *    * @returns The value requested or `BigInt(0)` if not found
     */
    paramByBlock(topic: string, name: string, blockNumber: Numbers, td?: Numbers, timestamp?: Numbers): bigint;
    /**
     * Checks if an EIP is activated by either being included in the EIPs
     * manually passed in with the {@link CommonOpts.eips} or in a
     * hardfork currently being active
     *
     * Note: this method only works for EIPs being supported
     * by the {@link CommonOpts.eips} constructor option
     * @param eip
     */
    isActivatedEIP(eip: number): boolean;
    /**
     * Checks if set or provided hardfork is active on block number
     * @param hardfork Hardfork name or null (for HF set)
     * @param blockNumber
     * @returns True if HF is active on block number
     */
    hardforkIsActiveOnBlock(_hardfork: string | Hardfork | null, _blockNumber: Numbers): boolean;
    /**
     * Alias to hardforkIsActiveOnBlock when hardfork is set
     * @param blockNumber
     * @returns True if HF is active on block number
     */
    activeOnBlock(blockNumber: Numbers): boolean;
    /**
     * Sequence based check if given or set HF1 is greater than or equal HF2
     * @param hardfork1 Hardfork name or null (if set)
     * @param hardfork2 Hardfork name
     * @param opts Hardfork options
     * @returns True if HF1 gte HF2
     */
    hardforkGteHardfork(_hardfork1: string | Hardfork | null, hardfork2: string | Hardfork): boolean;
    /**
     * Alias to hardforkGteHardfork when hardfork is set
     * @param hardfork Hardfork name
     * @returns True if hardfork set is greater than hardfork provided
     */
    gteHardfork(hardfork: string | Hardfork): boolean;
    /**
     * Returns the hardfork change block for hardfork provided or set
     * @param hardfork Hardfork name, optional if HF set
     * @returns Block number or null if unscheduled
     */
    hardforkBlock(_hardfork?: string | Hardfork): bigint | null;
    hardforkTimestamp(_hardfork?: string | Hardfork): bigint | null;
    /**
     * Returns the hardfork change block for eip
     * @param eip EIP number
     * @returns Block number or null if unscheduled
     */
    eipBlock(eip: number): bigint | null;
    /**
     * Returns the hardfork change total difficulty (Merge HF) for hardfork provided or set
     * @param hardfork Hardfork name, optional if HF set
     * @returns Total difficulty or null if no set
     */
    hardforkTTD(_hardfork?: string | Hardfork): bigint | null;
    /**
     * True if block number provided is the hardfork (given or set) change block
     * @param blockNumber Number of the block to check
     * @param hardfork Hardfork name, optional if HF set
     * @returns True if blockNumber is HF block
     * @deprecated
     */
    isHardforkBlock(_blockNumber: Numbers, _hardfork?: string | Hardfork): boolean;
    /**
     * Returns the change block for the next hardfork after the hardfork provided or set
     * @param hardfork Hardfork name, optional if HF set
     * @returns Block timestamp, number or null if not available
     */
    nextHardforkBlockOrTimestamp(_hardfork?: string | Hardfork): bigint | null;
    /**
     * Returns the change block for the next hardfork after the hardfork provided or set
     * @param hardfork Hardfork name, optional if HF set
     * @returns Block number or null if not available
     * @deprecated
     */
    nextHardforkBlock(_hardfork?: string | Hardfork): bigint | null;
    /**
     * True if block number provided is the hardfork change block following the hardfork given or set
     * @param blockNumber Number of the block to check
     * @param hardfork Hardfork name, optional if HF set
     * @returns True if blockNumber is HF block
     * @deprecated
     */
    isNextHardforkBlock(_blockNumber: Numbers, _hardfork?: string | Hardfork): boolean;
    /**
     * Internal helper function to calculate a fork hash
     * @param hardfork Hardfork name
     * @param genesisHash Genesis block hash of the chain
     * @returns Fork hash as hex string
     */
    _calcForkHash(hardfork: string | Hardfork, genesisHash: Uint8Array): string;
    /**
     * Returns an eth/64 compliant fork hash (EIP-2124)
     * @param hardfork Hardfork name, optional if HF set
     * @param genesisHash Genesis block hash of the chain, optional if already defined and not needed to be calculated
     */
    forkHash(_hardfork?: string | Hardfork, genesisHash?: Uint8Array): string;
    /**
     *
     * @param forkHash Fork hash as a hex string
     * @returns Array with hardfork data (name, block, forkHash)
     */
    hardforkForForkHash(forkHash: string): HardforkConfig | null;
    /**
     * Sets any missing forkHashes on the passed-in {@link Common} instance
     * @param common The {@link Common} to set the forkHashes for
     * @param genesisHash The genesis block hash
     */
    setForkHashes(genesisHash: Uint8Array): void;
    /**
     * Returns the Genesis parameters of the current chain
     * @returns Genesis dictionary
     */
    genesis(): GenesisBlockConfig;
    /**
     * Returns the hardforks for current chain
     * @returns {Array} Array with arrays of hardforks
     */
    hardforks(): HardforkConfig[];
    /**
     * Returns bootstrap nodes for the current chain
     * @returns {Dictionary} Dict with bootstrap nodes
     */
    bootstrapNodes(): BootstrapNodeConfig[] | undefined;
    /**
     * Returns DNS networks for the current chain
     * @returns {String[]} Array of DNS ENR urls
     */
    dnsNetworks(): string[];
    /**
     * Returns the hardfork set
     * @returns Hardfork name
     */
    hardfork(): string | Hardfork;
    /**
     * Returns the Id of current chain
     * @returns chain Id
     */
    chainId(): bigint;
    /**
     * Returns the name of current chain
     * @returns chain name (lower case)
     */
    chainName(): string;
    /**
     * Returns the Id of current network
     * @returns network Id
     */
    networkId(): bigint;
    /**
     * Returns the active EIPs
     * @returns List of EIPs
     */
    eips(): number[];
    /**
     * Returns the consensus type of the network
     * Possible values: "pow"|"poa"|"pos"
     *
     * Note: This value can update along a Hardfork.
     */
    consensusType(): string | ConsensusType;
    /**
     * Returns the concrete consensus implementation
     * algorithm or protocol for the network
     * e.g. "ethash" for "pow" consensus type,
     * "clique" for "poa" consensus type or
     * "casper" for "pos" consensus type.
     *
     * Note: This value can update along a Hardfork.
     */
    consensusAlgorithm(): string | ConsensusAlgorithm;
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
    consensusConfig(): {
        [key: string]: CliqueConfig | EthashConfig | CasperConfig;
    };
    /**
     * Returns a deep copy of this {@link Common} instance.
     */
    copy(): Common;
    static _getInitializedChains(customChains?: ChainConfig[]): ChainsConfig;
}
//# sourceMappingURL=common.d.ts.map