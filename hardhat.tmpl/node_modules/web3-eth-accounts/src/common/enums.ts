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
export enum Chain {
	Mainnet = 1,
	Goerli = 5,
	Sepolia = 11155111,
}

export enum Hardfork {
	Chainstart = 'chainstart',
	Homestead = 'homestead',
	Dao = 'dao',
	TangerineWhistle = 'tangerineWhistle',
	SpuriousDragon = 'spuriousDragon',
	Byzantium = 'byzantium',
	Constantinople = 'constantinople',
	Petersburg = 'petersburg',
	Istanbul = 'istanbul',
	MuirGlacier = 'muirGlacier',
	Berlin = 'berlin',
	London = 'london',
	ArrowGlacier = 'arrowGlacier',
	GrayGlacier = 'grayGlacier',
	MergeForkIdTransition = 'mergeForkIdTransition',
	Merge = 'merge',
	Shanghai = 'shanghai',
	ShardingForkDev = 'shardingFork',
}

export enum ConsensusType {
	ProofOfStake = 'pos',
	ProofOfWork = 'pow',
	ProofOfAuthority = 'poa',
}

export enum ConsensusAlgorithm {
	Ethash = 'ethash',
	Clique = 'clique',
	Casper = 'casper',
}

export enum CustomChain {
	/**
	 * Polygon (Matic) Mainnet
	 *
	 * - [Documentation](https://docs.matic.network/docs/develop/network-details/network)
	 */
	PolygonMainnet = 'polygon-mainnet',

	/**
	 * Polygon (Matic) Mumbai Testnet
	 *
	 * - [Documentation](https://docs.matic.network/docs/develop/network-details/network)
	 */
	PolygonMumbai = 'polygon-mumbai',

	/**
	 * Arbitrum Rinkeby Testnet
	 *
	 * - [Documentation](https://developer.offchainlabs.com/docs/public_testnet)
	 */
	ArbitrumRinkebyTestnet = 'arbitrum-rinkeby-testnet',

	/**
	 * Arbitrum One - mainnet for Arbitrum roll-up
	 *
	 * - [Documentation](https://developer.offchainlabs.com/public-chains)
	 */
	ArbitrumOne = 'arbitrum-one',

	/**
	 * xDai EVM sidechain with a native stable token
	 *
	 * - [Documentation](https://www.xdaichain.com/)
	 */
	xDaiChain = 'x-dai-chain',

	/**
	 * Optimistic Kovan - testnet for Optimism roll-up
	 *
	 * - [Documentation](https://community.optimism.io/docs/developers/tutorials.html)
	 */
	OptimisticKovan = 'optimistic-kovan',

	/**
	 * Optimistic Ethereum - mainnet for Optimism roll-up
	 *
	 * - [Documentation](https://community.optimism.io/docs/developers/tutorials.html)
	 */
	OptimisticEthereum = 'optimistic-ethereum',
}
