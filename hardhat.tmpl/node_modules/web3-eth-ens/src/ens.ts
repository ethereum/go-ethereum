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

import { Web3Context, Web3ContextObject } from 'web3-core';
import {
	ENSNetworkNotSyncedError,
	ENSUnsupportedNetworkError,
	RevertInstructionError,
} from 'web3-errors';
import { isSyncing } from 'web3-eth';
import { Contract } from 'web3-eth-contract';
import { getId } from 'web3-net';
import {
	Address,
	EthExecutionAPI,
	FMT_NUMBER,
	PayableCallOptions,
	SupportedProviders,
	TransactionReceipt,
	Web3NetAPI,
} from 'web3-types';
import { isAddress } from 'web3-validator';
import { PublicResolverAbi } from './abi/ens/PublicResolver.js';
import { networkIds, registryAddresses } from './config.js';
import { Registry } from './registry.js';
import { Resolver } from './resolver.js';

/**
 * This class is designed to interact with the ENS system on the Ethereum blockchain.
 * For using ENS package, first install Web3 package using: `npm i web3` or `yarn add web3` based on your package manager, after that ENS features can be used as mentioned in following snippet.
 * ```ts
 *
 * import { Web3 } from 'web3';
 *
 * const web3 = new Web3('https://127.0.0.1:4545');
 *
 * console.log(await web3.eth.ens.getAddress('ethereum.eth'))
 * ```
 * For using individual package install `web3-eth-ens` packages using: `npm i web3-eth-ens` or `yarn add web3-eth-ens`. This is more efficient approach for building lightweight applications.
 *
 * ```ts
 *import { ENS } from 'web3-eth-ens';
 *
 * const ens = new ENS(undefined,'https://127.0.0.1:4545');
 *
 * console.log(await ens.getAddress('vitalik.eth'));
 * ```
 */
export class ENS extends Web3Context<EthExecutionAPI & Web3NetAPI> {
	/**
	 * The registryAddress property can be used to define a custom registry address when you are connected to an unknown chain. It defaults to the main registry address.
	 */
	public registryAddress: string;
	private readonly _registry: Registry;
	private readonly _resolver: Resolver;
	private _detectedAddress?: string;
	private _lastSyncCheck?: number;

	/**
	 * Use to create an instance of ENS
	 * @param registryAddr - (Optional) The address of the ENS registry (default: mainnet registry address)
	 * @param provider - (Optional) The provider to use for the ENS instance
	 * @example
	 * ```ts
	 * const ens = new ENS(
	 * 	"0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e",
	 * 	"http://localhost:8545"
	 * );
	 *
	 * console.log( ens.defaultChain);
	 * > mainnet
	 * ```
	 */
	public constructor(
		registryAddr?: string,
		provider?:
			| SupportedProviders<EthExecutionAPI & Web3NetAPI>
			| Web3ContextObject<EthExecutionAPI & Web3NetAPI>
			| string,
	) {
		super(provider ?? '');
		this.registryAddress = registryAddr ?? registryAddresses.main; // will default to main registry address
		this._registry = new Registry(this.getContextObject(), registryAddr);
		this._resolver = new Resolver(this._registry);
	}

	/**
	 * Returns the Resolver by the given address
	 * @param name - The name of the ENS domain
	 * @returns - An contract instance of the resolver
	 *
	 * @example
	 * ```ts
	 * const resolver = await ens.getResolver('resolver');
	 *
	 * console.log(resolver.options.address);
	 * > '0x1234567890123456789012345678901234567890'
	 * ```
	 */
	public async getResolver(name: string): Promise<Contract<typeof PublicResolverAbi>> {
		return this._registry.getResolver(name);
	}

	/**
	 * Returns true if the record exists
	 * @param name - The ENS name
	 * @returns - Returns `true` if node exists in this ENS registry. This will return `false` for records that are in the legacy ENS registry but have not yet been migrated to the new one.
	 * @example
	 * ```ts
	 * const exists = await web3.eth.ens.recordExists('ethereum.eth');
	 * ```
	 */
	public async recordExists(name: string): Promise<unknown> {
		return this._registry.recordExists(name);
	}

	/**
	 * Returns the caching TTL (time-to-live) of an ENS name.
	 * @param name - The ENS name
	 * @returns - Returns the caching TTL (time-to-live) of a name.
	 * @example
	 * ```ts
	 * const owner = await web3.eth.ens.getTTL('ethereum.eth');
	 * ```
	 */
	public async getTTL(name: string): Promise<unknown> {
		return this._registry.getTTL(name);
	}

	/**
	 * Returns the owner by the given name and current configured or detected Registry
	 * @param name - The ENS name
	 * @returns - Returns the address of the owner of the name.
	 * @example
	 * ```ts
	 * const owner = await web3.eth.ens.getOwner('ethereum.eth');
	 * ```
	 */
	public async getOwner(name: string): Promise<unknown> {
		return this._registry.getOwner(name);
	}

	/**
	 * Resolves an ENS name to an Ethereum address.
	 * @param ENSName - The ENS name to resolve
	 * @param coinType - (Optional) The coin type, defaults to 60 (ETH)
	 * @returns - The Ethereum address of the given name
	 * ```ts
	 * const address = await web3.eth.ens.getAddress('ethereum.eth');
	 * console.log(address);
	 * > '0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359'
	 * ```
	 */
	public async getAddress(ENSName: string, coinType = 60) {
		return this._resolver.getAddress(ENSName, coinType);
	}

	/**
	 * ERC-634 - Returns the text content stored in the resolver for the specified key.
	 * @param ENSName - The ENS name to resolve
	 * @param key - The key to resolve https://github.com/ethereum/ercs/blob/master/ERCS/erc-634.md#global-keys
	 * @returns - The value content stored in the resolver for the specified key
	 */
	public async getText(ENSNameOrAddr: string | Address, key: string): Promise<string> {
		if(isAddress(ENSNameOrAddr))
			return this._resolver.getText(await(this._resolver.getName(ENSNameOrAddr,false)), key);
		return this._resolver.getText(ENSNameOrAddr, key);
	}

	/**
	 * Resolves the name of an ENS node.
	 * @param ENSName - The node to resolve
	 * @returns - The name
	 */
	public async getName(ENSName: string, checkInterfaceSupport = true): Promise<string> {
		return this._resolver.getName(ENSName, checkInterfaceSupport);
	}

	/**
	 * Returns the X and Y coordinates of the curve point for the public key.
	 * @param ENSName - The ENS name
	 * @returns - The X and Y coordinates of the curve point for the public key
	 * @example
	 * ```ts
	 * const key = await web3.eth.ens.getPubkey('ethereum.eth');
	 * console.log(key);
	 * > {
	 * "0": "0x0000000000000000000000000000000000000000000000000000000000000000",
	 * "1": "0x0000000000000000000000000000000000000000000000000000000000000000",
	 * "x": "0x0000000000000000000000000000000000000000000000000000000000000000",
	 * "y": "0x0000000000000000000000000000000000000000000000000000000000000000"
	 * }
	 * ```
	 */
	public async getPubkey(ENSName: string) {
		return this._resolver.getPubkey(ENSName);
	}

	/**
	 * Returns the content hash object associated with an ENS node.
	 * @param ENSName - The ENS name
	 * @returns - The content hash object associated with an ENS node
	 * @example
	 * ```ts
	 * const hash = await web3.eth.ens.getContenthash('ethereum.eth');
	 * console.log(hash);
	 * > 'QmaEBknbGT4bTQiQoe2VNgBJbRfygQGktnaW5TbuKixjYL'
	 * ```
	 */
	public async getContenthash(ENSName: string) {
		return this._resolver.getContenthash(ENSName);
	}

	/**
	 * Checks if the current used network is synced and looks for ENS support there.
	 * Throws an error if not.
	 * @returns - The address of the ENS registry if the network has been detected successfully
	 * @example
	 * ```ts
	 * console.log(await web3.eth.ens.checkNetwork());
	 * > '0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e'
	 * ```
	 */
	public async checkNetwork() {
		const now = Date.now() / 1000;
		if (!this._lastSyncCheck || now - this._lastSyncCheck > 3600) {
			const syncInfo = await isSyncing(this);

			if (!(typeof syncInfo === 'boolean' && !syncInfo)) {
				throw new ENSNetworkNotSyncedError();
			}

			this._lastSyncCheck = now;
		}

		if (this._detectedAddress) {
			return this._detectedAddress;
		}
		const networkType = await getId(this, {
			...this.defaultReturnFormat,
			number: FMT_NUMBER.HEX,
		}); // get the network from provider
		const addr = registryAddresses[networkIds[networkType]];

		if (typeof addr === 'undefined') {
			throw new ENSUnsupportedNetworkError(networkType);
		}

		this._detectedAddress = addr;
		return this._detectedAddress;
	}

	/**
	 * Returns true if the related Resolver does support the given signature or interfaceId.
	 * @param ENSName - The ENS name
	 * @param interfaceId - The signature of the function or the interfaceId as described in the ENS documentation
	 * @returns - `true` if the related Resolver does support the given signature or interfaceId.
	 * @example
	 * ```ts
	 * const supports = await web3.eth.ens.supportsInterface('ethereum.eth', 'addr(bytes32');
	 * console.log(supports);
	 * > true
	 * ```
	 */
	public async supportsInterface(ENSName: string, interfaceId: string) {
		return this._resolver.supportsInterface(ENSName, interfaceId);
	}

	/**
	 * @returns - Returns all events that can be emitted by the ENS registry.
	 */
	public get events() {
		return this._registry.events;
	}

	/**
	 * Sets the address of an ENS name in his resolver.
	 * @param name - The ENS name
	 * @param address - The address to set
	 * @param txConfig - (Optional) The transaction config
	 * @returns - The transaction receipt
	 * ```ts
	 * const receipt = await ens.setAddress('web3js.eth','0xe2597eb05cf9a87eb1309e86750c903ec38e527e');
	 *```
	 */
	public async setAddress(
		name: string,
		address: Address,
		txConfig: PayableCallOptions,
	): Promise<TransactionReceipt | RevertInstructionError> {
		return this._resolver.setAddress(name, address, txConfig);
	}
}
