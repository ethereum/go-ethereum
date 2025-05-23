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

import HttpProvider, { HttpProviderOptions } from 'web3-providers-http';
import WebSocketProvider from 'web3-providers-ws';
import {
	EthExecutionAPI,
	JsonRpcResult,
	ProviderConnectInfo,
	ProviderMessage,
	ProviderRpcError,
	Web3APIMethod,
	Web3APIPayload,
	Web3APIReturnType,
	Web3APISpec,
	Web3BaseProvider,
	Web3Eip1193ProviderEventCallback,
	Web3ProviderEventCallback,
	Web3ProviderMessageEventCallback,
	Web3ProviderStatus,
	JsonRpcResponseWithResult,
} from 'web3-types';
import { Eip1193Provider } from 'web3-utils';
import { Transport, Network, SocketOptions } from './types.js';
import { ProviderConfigOptionsError } from './errors.js';

/* 
This class can be used to create new providers only when there is custom logic required in each Request method like
checking specific HTTP status codes and performing any action, throwing new error types or setting additional HTTP headers in requests, or even modifying requests.

Another simpler approach can be a function simply returning URL strings instead of using the following class in case if
no additional logic implementation is required in the provider.
*/

export abstract class Web3ExternalProvider<
	API extends Web3APISpec = EthExecutionAPI,
> extends Eip1193Provider {
	public provider!: Web3BaseProvider;
	public readonly transport: Transport;

	public abstract getRPCURL(
		network: Network,
		transport: Transport,
		token: string,
		host: string,
	): string;

	public constructor(
		network: Network,
		transport: Transport,
		token: string,
		host: string,
		providerConfigOptions?: HttpProviderOptions | SocketOptions,
	) {
		super();

		if (
			providerConfigOptions !== undefined &&
			transport === Transport.HTTPS &&
			!('providerOptions' in providerConfigOptions)
		) {
			throw new ProviderConfigOptionsError('HTTP Provider');
		} else if (
			providerConfigOptions !== undefined &&
			transport === Transport.WebSocket &&
			!(
				'socketOptions' in providerConfigOptions ||
				'reconnectOptions' in providerConfigOptions
			)
		) {
			throw new ProviderConfigOptionsError('Websocket Provider');
		}

		this.transport = transport;
		if (transport === Transport.HTTPS) {
			this.provider = new HttpProvider(
				this.getRPCURL(network, transport, token, host),
				providerConfigOptions as HttpProviderOptions,
			);
		} else if (transport === Transport.WebSocket) {
			this.provider = new WebSocketProvider(
				this.getRPCURL(network, transport, token, host),
				(providerConfigOptions as SocketOptions)?.socketOptions,
				(providerConfigOptions as SocketOptions)?.reconnectOptions,
			);
		}
	}

	public async request<
		Method extends Web3APIMethod<API>,
		ResultType = Web3APIReturnType<API, Method>,
	>(
		payload: Web3APIPayload<EthExecutionAPI, Method>,
		requestOptions?: RequestInit,
	): Promise<JsonRpcResponseWithResult<ResultType>> {
		if (this.transport === Transport.HTTPS) {
			return (await (this.provider as HttpProvider).request(
				payload,
				requestOptions,
			)) as unknown as JsonRpcResponseWithResult<ResultType>;
		}

		return (this.provider as WebSocketProvider).request(payload);
	}

	public getStatus(): Web3ProviderStatus {
		return this.provider.getStatus();
	}
	public supportsSubscriptions(): boolean {
		return this.provider.supportsSubscriptions();
	}
	public once(
		type: 'disconnect',
		listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>,
	): void;
	public once<T = JsonRpcResult>(
		type: string,
		listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderEventCallback<T>,
	): void;
	public once(
		type: 'connect',
		listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>,
	): void;
	public once(type: 'chainChanged', listener: Web3Eip1193ProviderEventCallback<string>): void;
	public once(
		type: 'accountsChanged',
		listener: Web3Eip1193ProviderEventCallback<string[]>,
	): void;
	public once(_type: string, _listener: unknown): void {
		if (this.provider?.once) {
			// eslint-disable-next-line @typescript-eslint/no-unsafe-argument
			this.provider.once(
				_type,
				_listener as
					| Web3Eip1193ProviderEventCallback<ProviderMessage>
					| Web3ProviderEventCallback,
			);
		}
	}
	public removeAllListeners?(_type: string): void {
		if (this.provider?.removeAllListeners) this.provider.removeAllListeners(_type);
	}
	public connect(): void {
		if (this.provider?.connect) this.provider.connect();
	}
	public disconnect(_code?: number | undefined, _data?: string | undefined): void {
		if (this.provider?.disconnect) this.provider.disconnect(_code, _data);
	}
	public reset(): void {
		if (this.provider?.reset) this.provider.reset();
	}

	public on(
		type: 'disconnect',
		listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>,
	): void;
	public on<T = JsonRpcResult>(
		type: string,
		listener:
			| Web3Eip1193ProviderEventCallback<ProviderMessage>
			| Web3ProviderMessageEventCallback<T>,
	): void;
	public on<T = JsonRpcResult>(
		type: string,
		listener:
			| Web3Eip1193ProviderEventCallback<ProviderMessage>
			| Web3ProviderMessageEventCallback<T>,
	): void;
	public on(
		type: 'connect',
		listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>,
	): void;
	public on(type: 'chainChanged', listener: Web3Eip1193ProviderEventCallback<string>): void;
	public on(type: 'accountsChanged', listener: Web3Eip1193ProviderEventCallback<string[]>): void;
	public on(_type: unknown, _listener: unknown): void {
		if (this.provider)
			// eslint-disable-next-line @typescript-eslint/no-unsafe-argument
			this.provider.on(
				_type as string,
				_listener as
					| Web3Eip1193ProviderEventCallback<ProviderMessage>
					| Web3ProviderMessageEventCallback,
			);
	}
	public removeListener(
		type: 'disconnect',
		listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>,
	): void;
	public removeListener<T = JsonRpcResult>(
		type: string,
		listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderEventCallback<T>,
	): void;
	public removeListener(
		type: 'connect',
		listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>,
	): void;
	public removeListener(
		type: 'chainChanged',
		listener: Web3Eip1193ProviderEventCallback<string>,
	): void;
	public removeListener(
		type: 'accountsChanged',
		listener: Web3Eip1193ProviderEventCallback<string[]>,
	): void;
	public removeListener(_type: unknown, _listener: unknown): void {
		if (this.provider)
			this.provider.removeListener(
				_type as string,
				_listener as
					| Web3Eip1193ProviderEventCallback<ProviderMessage>
					| Web3ProviderEventCallback,
			);
	}
}
