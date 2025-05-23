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
import { Web3Context } from 'web3-core';
import { DataFormat, DEFAULT_RETURN_FORMAT, Web3NetAPI } from 'web3-types';

import * as rpcMethodsWrappers from './rpc_method_wrappers.js';

/**
 * Net class allows you to interact with an Ethereum node’s network properties.
 * For using Net package, first install Web3 package using: `npm i web3` or `yarn add web3` based on your package manager, after that Net features can be used.
 * ```ts
 *
 * import { Web3 } from 'web3';
 * const web3 = new Web3('https://mainnet.infura.io/v3/<YOURPROJID>');
 *
 * console.log(await web3.eth.net.getId());
 *
 * ```
 * For using individual package install `web3-net` packages using: `npm i web3-net` or `yarn add web3-net`.
 *
 * ```ts
 * import {Net} from 'web3-net';
 *
 *  const net = new Net('https://mainnet.infura.io/v3/<YOURPROJID>');
 *  console.log(await net.getId());
 * ```
 */
export class Net extends Web3Context<Web3NetAPI> {
	/**
	 * Gets the current network ID
	 *
	 * @param returnFormat - Return format
	 * @returns A Promise of the network ID.
	 * @example
	 * ```ts
	 * const net = new Net(Net.givenProvider || 'ws://some.local-or-remote.node:8546');
	 * await net.getId();
	 * > 1
	 * ```
	 */
	public async getId<ReturnFormat extends DataFormat = typeof DEFAULT_RETURN_FORMAT>(
		returnFormat: ReturnFormat = this.defaultReturnFormat as ReturnFormat,
	) {
		return rpcMethodsWrappers.getId(this, returnFormat);
	}

	/**
	 * Get the number of peers connected to.
	 *
	 * @param returnFormat - Return format
	 * @returns A promise of the number of the peers connected to.
	 * @example
	 * ```ts
	 * const net = new Net(Net.givenProvider || 'ws://some.local-or-remote.node:8546');
	 * await net.getPeerCount();
	 * > 0
	 * ```
	 */
	public async getPeerCount<ReturnFormat extends DataFormat = typeof DEFAULT_RETURN_FORMAT>(
		returnFormat: ReturnFormat = this.defaultReturnFormat as ReturnFormat,
	) {
		return rpcMethodsWrappers.getPeerCount(this, returnFormat);
	}

	/**
	 * Check if the node is listening for peers
	 *
	 * @returns A promise of a boolean if the node is listening to peers
	 * @example
	 * ```ts
	 * const net = new Net(Net.givenProvider || 'ws://some.local-or-remote.node:8546');
	 * await net.isListening();
	 * > true
	 * ```
	 */
	public async isListening() {
		return rpcMethodsWrappers.isListening(this);
	}
}
