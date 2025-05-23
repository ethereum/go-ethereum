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
	EthExecutionAPI,
	HexString,
	ProviderConnectInfo,
	Web3APIMethod,
	Web3APIPayload,
	Web3APISpec,
	Web3BaseProvider,
} from 'web3-types';
import { EventEmitter } from 'eventemitter3';
import { EIP1193ProviderRpcError } from 'web3-errors';
import { toPayload } from './json_rpc.js';

/**
 * This is an abstract class, which extends {@link Web3BaseProvider} class. This class is used to implement a provider that adheres to the EIP-1193 standard for Ethereum providers.
 */
export abstract class Eip1193Provider<
	API extends Web3APISpec = EthExecutionAPI,
> extends Web3BaseProvider<API> {
	protected readonly _eventEmitter: EventEmitter = new EventEmitter();
	private _chainId: HexString = '';
	private _accounts: HexString[] = [];

	private async _getChainId(): Promise<HexString> {
		const data = await (this as Web3BaseProvider<API>).request<
			Web3APIMethod<API>,
			ResponseType
		>(
			toPayload({
				method: 'eth_chainId',
				params: [],
			}) as Web3APIPayload<API, Web3APIMethod<API>>,
		);
		return data?.result ?? '';
	}

	private async _getAccounts(): Promise<HexString[]> {
		const data = await (this as Web3BaseProvider<API>).request<Web3APIMethod<API>, HexString[]>(
			toPayload({
				method: 'eth_accounts',
				params: [],
			}) as Web3APIPayload<API, Web3APIMethod<API>>,
		);
		return data?.result ?? [];
	}

	protected _onConnect() {
		Promise.all([
			this._getChainId()
				.then(chainId => {
					if (chainId !== this._chainId) {
						this._chainId = chainId;
						this._eventEmitter.emit('chainChanged', this._chainId);
					}
				})
				.catch(err => {
					// todo: add error handler
					console.error(err);
				}),

			this._getAccounts()
				.then(accounts => {
					if (
						!(
							this._accounts.length === accounts.length &&
							accounts.every(v => accounts.includes(v))
						)
					) {
						this._accounts = accounts;
						this._onAccountsChanged();
					}
				})
				.catch(err => {
					// todo: add error handler
					// eslint-disable-next-line no-console
					console.error(err);
				}),
		])
			.then(() =>
				this._eventEmitter.emit('connect', {
					chainId: this._chainId,
				} as ProviderConnectInfo),
			)
			.catch(err => {
				// todo: add error handler
				// eslint-disable-next-line no-console
				console.error(err);
			});
	}

	// todo this must be ProvideRpcError with a message too
	protected _onDisconnect(code: number, data?: unknown) {
		this._eventEmitter.emit('disconnect', new EIP1193ProviderRpcError(code, data));
	}

	private _onAccountsChanged() {
		// get chainId and safe to local
		this._eventEmitter.emit('accountsChanged', this._accounts);
	}
}
