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
import type { Chain, ConsensusAlgorithm, ConsensusType, Hardfork } from './enums.js';

export interface ChainName {
	[chainId: string]: string;
}

export type CliqueConfig = {
	period: number;
	epoch: number;
};

export type EthashConfig = Record<string, unknown>;

export type CasperConfig = Record<string, unknown>;

export interface GenesisBlockConfig {
	timestamp?: string;
	gasLimit: number;
	difficulty: number;
	nonce: string;
	extraData: string;
	baseFeePerGas?: string;
}

export interface HardforkConfig {
	name: Hardfork | string;
	// eslint-disable-next-line @typescript-eslint/ban-types
	block: number | null; // null is used for hardforks that should not be applied -- since `undefined` isn't a valid value in JSON
	ttd?: bigint | string;
	timestamp?: number | string;
	// eslint-disable-next-line @typescript-eslint/ban-types
	forkHash?: string | null;
}

export interface BootstrapNodeConfig {
	ip: string;
	port: number | string;
	network?: string;
	chainId?: number;
	id: string;
	location: string;
	comment: string;
}

export interface ChainConfig {
	name: string;
	chainId: number | bigint;
	networkId: number | bigint;
	defaultHardfork?: string;
	comment?: string;
	url?: string;
	genesis: GenesisBlockConfig;
	hardforks: HardforkConfig[];
	bootstrapNodes?: BootstrapNodeConfig[];
	dnsNetworks?: string[];
	consensus: {
		type: ConsensusType | string;
		algorithm: ConsensusAlgorithm | string;
		clique?: CliqueConfig;
		ethash?: EthashConfig;
		casper?: CasperConfig;
	};
}
export interface ChainsConfig {
	[key: string]: ChainConfig | ChainName;
}

interface BaseOpts {
	/**
	 * String identifier ('byzantium') for hardfork or {@link Hardfork} enum.
	 *
	 * Default: Hardfork.London
	 */
	hardfork?: string | Hardfork;
	/**
	 * Selected EIPs which can be activated, please use an array for instantiation
	 * (e.g. `eips: [ 2537, ]`)
	 *
	 * Currently supported:
	 *
	 * - [EIP-2537](https://eips.ethereum.org/EIPS/eip-2537) - BLS12-381 precompiles
	 */
	eips?: number[];
}

/**
 * Options for instantiating a {@link Common} instance.
 */
export interface CommonOpts extends BaseOpts {
	/**
	 * Chain name ('mainnet'), id (1), or {@link Chain} enum,
	 * either from a chain directly supported or a custom chain
	 * passed in via {@link CommonOpts.customChains}.
	 */
	chain: string | number | Chain | bigint | object;
	/**
	 * Initialize (in addition to the supported chains) with the selected
	 * custom chains. Custom genesis state should be passed to the Blockchain class if used.
	 *
	 * Usage (directly with the respective chain initialization via the {@link CommonOpts.chain} option):
	 *
	 * ```javascript
	 * import myCustomChain1 from '[PATH_TO_MY_CHAINS]/myCustomChain1.json'
	 * const common = new Common({ chain: 'myCustomChain1', customChains: [ myCustomChain1 ]})
	 * ```
	 */
	customChains?: ChainConfig[];
}

/**
 * Options to be used with the {@link Common.custom} static constructor.
 */
export interface CustomCommonOpts extends BaseOpts {
	/**
	 * The name (`mainnet`), id (`1`), or {@link Chain} enum of
	 * a standard chain used to base the custom chain params on.
	 */
	baseChain?: string | number | Chain | bigint;
}

export interface GethConfigOpts extends BaseOpts {
	chain?: string;
	genesisHash?: Uint8Array;
	mergeForkIdPostMerge?: boolean;
}

/*
 * A type that represents a `0x`-prefixed hex string.
 */
export type PrefixedHexString = string;

/*
 * A type that represents an input that can be converted to a Buffer.
 */
export type Uint8ArrayLike = Uint8Array | number[] | number | bigint | PrefixedHexString;

/*
 * A type that represents an input that can be converted to a BigInt.
 */
export type BigIntLike = bigint | PrefixedHexString | number | Uint8Array;

/*
 * A type that represents an object that has a `toArray()` method.
 */
export interface TransformableToArray {
	toArray(): Uint8Array;
}

export type NestedUint8Array = Array<Uint8Array | NestedUint8Array>;
/**
 * Type output options
 */
export enum TypeOutput {
	Number,
	BigInt,
	Uint8Array,
	PrefixedHexString,
}

export type TypeOutputReturnType = {
	[TypeOutput.Number]: number;
	[TypeOutput.BigInt]: bigint;
	[TypeOutput.Uint8Array]: Uint8Array;
	[TypeOutput.PrefixedHexString]: PrefixedHexString;
};
export type ToBytesInputTypes =
	| PrefixedHexString
	| number
	| bigint
	| Uint8Array
	| number[]
	| TransformableToArray
	// eslint-disable-next-line @typescript-eslint/ban-types
	| null
	| undefined;
