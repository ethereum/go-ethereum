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

import { Socket, SocketConstructorOpts } from 'net';
import { ConnectionNotOpenError, InvalidClientError } from 'web3-errors';
import { ReconnectOptions, SocketProvider, toUtf8 } from 'web3-utils';
import {
	EthExecutionAPI,
	Web3APIMethod,
	Web3APIPayload,
	Web3APISpec,
	Web3ProviderStatus,
} from 'web3-types';
import { existsSync } from 'fs';

/**
 * The IPC Provider could be used in node.js dapps when running a local node. And it provide the most secure connection.
 *
 * @example
 * ```ts
 * const provider = new IpcProvider(
 * 		`path.ipc`,
 * 		{
 * 			writable: false,
 * 		},
 * 		{
 * 			delay: 500,
 * 			autoReconnect: true,
 * 			maxAttempts: 10,
 * 		},
 * 	);
 * ```
 *
 * The second and the third parameters are both optional. And you can for example, the second parameter could be an empty object or undefined.
 *  * @example
 * ```ts
 * const provider = new IpcProvider(
 * 		`path.ipc`,
 * 		{},
 * 		{
 * 			delay: 500,
 * 			autoReconnect: true,
 * 			maxAttempts: 10,
 * 		},
 * 	);
 * ```
 */
export default class IpcProvider<API extends Web3APISpec = EthExecutionAPI> extends SocketProvider<
	Uint8Array | string,
	CloseEvent,
	Error,
	API
> {
	protected readonly _socketOptions?: SocketConstructorOpts;

	protected _socketConnection?: Socket;

	/**
	 * This is a class used for IPC connections. It extends the abstract class SocketProvider {@link SocketProvider} that extends the EIP-1193 provider {@link EIP1193Provider}.
	 * @param socketPath - The path to the IPC socket.
	 * @param socketOptions - The options for the IPC socket connection.
	 * @param reconnectOptions - The options for the socket reconnection {@link ReconnectOptions}
	 */
	// this constructor is to specify the type for `socketOptions` for a better intellisense.
	// eslint-disable-next-line no-useless-constructor
	public constructor(
		socketPath: string,
		socketOptions?: SocketConstructorOpts,
		reconnectOptions?: Partial<ReconnectOptions>,
	) {
		super(socketPath, socketOptions, reconnectOptions);
	}

	public getStatus(): Web3ProviderStatus {
		if (this._socketConnection?.connecting) {
			return 'connecting';
		}
		return this._connectionStatus;
	}

	protected _openSocketConnection() {
		if (!existsSync(this._socketPath)) {
			throw new InvalidClientError(this._socketPath);
		}
		if (!this._socketConnection || this.getStatus() === 'disconnected') {
			this._socketConnection = new Socket(this._socketOptions);
		}

		this._socketConnection.connect({ path: this._socketPath });
	}

	protected _closeSocketConnection(code: number, data?: string) {
		this._socketConnection?.end(() => {
			this._onDisconnect(code, data);
		});
	}

	protected _sendToSocket<Method extends Web3APIMethod<API>>(
		payload: Web3APIPayload<API, Method>,
	): void {
		if (this.getStatus() === 'disconnected') {
			throw new ConnectionNotOpenError();
		}
		this._socketConnection?.write(JSON.stringify(payload));
	}

	protected _parseResponses(e: Uint8Array | string) {
		return this.chunkResponseParser.parseResponse(typeof e === 'string' ? e : toUtf8(e));
	}

	protected _addSocketListeners(): void {
		this._socketConnection?.on('data', this._onMessageHandler);
		this._socketConnection?.on('connect', this._onOpenHandler);
		this._socketConnection?.on('close', this._onClose.bind(this));
		this._socketConnection?.on('end', this._onCloseHandler);
		this._socketConnection?.on('error', this._onErrorHandler);
	}

	protected _removeSocketListeners(): void {
		this._socketConnection?.removeAllListeners('connect');
		this._socketConnection?.removeAllListeners('end');
		this._socketConnection?.removeAllListeners('close');
		this._socketConnection?.removeAllListeners('data');
		// note: we intentionally keep the error event listener to be able to emit it in case an error happens when closing the connection
	}

	protected _onCloseEvent(event: CloseEvent): void {
		if (!event && this._reconnectOptions.autoReconnect) {
			this._connectionStatus = 'disconnected';
			this._reconnect();
			return;
		}

		this._clearQueues(event);
		this._removeSocketListeners();
		this._onDisconnect(event?.code, event?.reason);
		// disconnect was successful and can safely remove error listener
		this._socketConnection?.removeAllListeners('error');
	}

	protected _onClose(event: CloseEvent): void {
		this._clearQueues(event);
		this._removeSocketListeners();
	}
}

export { IpcProvider };
