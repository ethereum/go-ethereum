/// <reference types="node" />
import { EventEmitter } from 'events';
import { Chain, CustomChain, Hardfork } from './enums.js';
import { hardforks as HARDFORK_SPECS } from './hardforks.js';
import type { ConsensusAlgorithm, ConsensusType } from './enums.js';
import type { BootstrapNodeConfig, CasperConfig, ChainConfig, ChainsConfig, CliqueConfig, CommonOpts, CustomCommonOpts, CustomCrypto, EIPConfig, EIPOrHFConfig, EthashConfig, GenesisBlockConfig, GethConfigOpts, HardforkByOpts, HardforkConfig, HardforkTransitionConfig } from './types.js';
import type { BigIntLike, PrefixedHexString } from '@nomicfoundation/ethereumjs-util';
declare type HardforkSpecKeys = string;
declare type HardforkSpecValues = typeof HARDFORK_SPECS[HardforkSpecKeys];
declare type ParamsCacheConfig = Omit<EIPOrHFConfig, 'comment' | 'url' | 'status'>;
/**
 * Common class to access chain and hardfork parameters and to provide
 * a unified and shared view on the network and hardfork state.
 *
 * Use the {@link Common.custom} static constructor for creating simple
 * custom chain {@link Common} objects (more complete custom chain setups
 * can be created via the main constructor and the {@link CommonOpts.customChains} parameter).
 */
export declare class Common {
    readonly DEFAULT_HARDFORK: string | Hardfork;
    protected _chainParams: ChainConfig;
    protected _hardfork: string | Hardfork;
    protected _eips: number[];
    protected _customChains: ChainConfig[];
    readonly customCrypto: CustomCrypto;
    protected _paramsCache: ParamsCacheConfig;
    protected _activatedEIPsCache: number[];
    protected HARDFORK_CHANGES: [HardforkSpecKeys, HardforkSpecValues][];
    events: EventEmitter;
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
    protected static _getChainParams(chain: string | number | Chain | bigint, customChains?: ChainConfig[]): ChainConfig;
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
    getHardforkBy(opts: HardforkByOpts): string;
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
    setHardforkBy(opts: HardforkByOpts): string;
    /**
     * Internal helper function, returns the params for the given hardfork for the chain set
     * @param hardfork Hardfork name
     * @returns Dictionary with hardfork params or null if hardfork not on chain
     */
    protected _getHardfork(hardfork: string | Hardfork): HardforkTransitionConfig | null;
    /**
     * Sets the active EIPs
     * @param eips
     */
    setEIPs(eips?: number[]): void;
    /**
     * Internal helper for _buildParamsCache()
     */
    protected _mergeWithParamsCache(params: HardforkConfig | EIPConfig): void;
    /**
     * Build up a cache for all parameter values for the current HF and all activated EIPs
     */
    protected _buildParamsCache(): void;
    protected _buildActivatedEIPsCache(): void;
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
    paramByBlock(topic: string, name: string, blockNumber: BigIntLike, td?: BigIntLike, timestamp?: BigIntLike): bigint;
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
    hardforkIsActiveOnBlock(hardfork: string | Hardfork | null, blockNumber: BigIntLike): boolean;
    /**
     * Alias to hardforkIsActiveOnBlock when hardfork is set
     * @param blockNumber
     * @returns True if HF is active on block number
     */
    activeOnBlock(blockNumber: BigIntLike): boolean;
    /**
     * Sequence based check if given or set HF1 is greater than or equal HF2
     * @param hardfork1 Hardfork name or null (if set)
     * @param hardfork2 Hardfork name
     * @param opts Hardfork options
     * @returns True if HF1 gte HF2
     */
    hardforkGteHardfork(hardfork1: string | Hardfork | null, hardfork2: string | Hardfork): boolean;
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
    hardforkBlock(hardfork?: string | Hardfork): bigint | null;
    hardforkTimestamp(hardfork?: string | Hardfork): bigint | null;
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
    hardforkTTD(hardfork?: string | Hardfork): bigint | null;
    /**
     * Returns the change block for the next hardfork after the hardfork provided or set
     * @param hardfork Hardfork name, optional if HF set
     * @returns Block timestamp, number or null if not available
     */
    nextHardforkBlockOrTimestamp(hardfork?: string | Hardfork): bigint | null;
    /**
     * Internal helper function to calculate a fork hash
     * @param hardfork Hardfork name
     * @param genesisHash Genesis block hash of the chain
     * @returns Fork hash as hex string
     */
    protected _calcForkHash(hardfork: string | Hardfork, genesisHash: Uint8Array): PrefixedHexString;
    /**
     * Returns an eth/64 compliant fork hash (EIP-2124)
     * @param hardfork Hardfork name, optional if HF set
     * @param genesisHash Genesis block hash of the chain, optional if already defined and not needed to be calculated
     */
    forkHash(hardfork?: string | Hardfork, genesisHash?: Uint8Array): PrefixedHexString;
    /**
     *
     * @param forkHash Fork hash as a hex string
     * @returns Array with hardfork data (name, block, forkHash)
     */
    hardforkForForkHash(forkHash: string): HardforkTransitionConfig | null;
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
    hardforks(): HardforkTransitionConfig[];
    /**
     * Returns bootstrap nodes for the current chain
     * @returns {Dictionary} Dict with bootstrap nodes
     */
    bootstrapNodes(): BootstrapNodeConfig[];
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
     * Returns the additionally activated EIPs
     * (by using the `eips` constructor option)
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
    static getInitializedChains(customChains?: ChainConfig[]): ChainsConfig;
}
export {};
//# sourceMappingURL=common.d.ts.map