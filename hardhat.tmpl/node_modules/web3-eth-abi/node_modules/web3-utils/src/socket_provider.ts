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
	ConnectionEvent,
	Eip1193EventName,
	EthExecutionAPI,
	JsonRpcBatchRequest,
	JsonRpcBatchResponse,
	JsonRpcId,
	JsonRpcNotification,
	JsonRpcRequest,
	JsonRpcResponse,
	JsonRpcResponseWithResult,
	JsonRpcResult,
	ProviderConnectInfo,
	ProviderMessage,
	ProviderRpcError,
	SocketRequestItem,
	Web3APIMethod,
	Web3APIPayload,
	Web3APIReturnType,
	Web3APISpec,
	Web3Eip1193ProviderEventCallback,
	Web3ProviderEventCallback,
	Web3ProviderMessageEventCallback,
	Web3ProviderStatus,
} from 'web3-types';
import {
	ConnectionError,
	ConnectionNotOpenError,
	InvalidClientError,
	MaxAttemptsReachedOnReconnectingError,
	PendingRequestsOnReconnectingError,
	RequestAlreadySentError,
	Web3WSProviderError,
} from 'web3-errors';
import { Eip1193Provider } from './web3_eip1193_provider.js';
import { ChunkResponseParser } from './chunk_response_parser.js';
import { isNullish } from './validation.js';
import { Web3DeferredPromise } from './web3_deferred_promise.js';
import * as jsonRpc from './json_rpc.js';

export type ReconnectOptions = {
	autoReconnect: boolean;
	delay: number;
	maxAttempts: number;
};

const DEFAULT_RECONNECTION_OPTIONS = {
	autoReconnect: true,
	delay: 5000,
	maxAttempts: 5,
};

const NORMAL_CLOSE_CODE = 1000; // https://developer.mozilla.org/en-US/docs/Web/API/WebSocket/close

export abstract class SocketProvider<
	MessageEvent,
	CloseEvent,
	ErrorEvent,
	API extends Web3APISpec = EthExecutionAPI,
> extends Eip1193Provider<API> {
	protected isReconnecting: boolean;
	protected readonly _socketPath: string;
	protected readonly chunkResponseParser: ChunkResponseParser;
	/* eslint-disable @typescript-eslint/no-explicit-any */
	protected readonly _pendingRequestsQueue: Map<JsonRpcId, SocketRequestItem<any, any, any>>;
	/* eslint-disable @typescript-eslint/no-explicit-any */
	protected readonly _sentRequestsQueue: Map<JsonRpcId, SocketRequestItem<any, any, any>>;
	protected _reconnectAttempts!: number;
	protected readonly _socketOptions?: unknown;
	protected readonly _reconnectOptions: ReconnectOptions;
	protected _socketConnection?: unknown;
	public get SocketConnection() {
		return this._socketConnection;
	}
	protected _connectionStatus: Web3ProviderStatus;
	protected readonly _onMessageHandler: (event: MessageEvent) => void;
	protected readonly _onOpenHandler: () => void;
	protected readonly _onCloseHandler: (event: CloseEvent) => void;
	protected readonly _onErrorHandler: (event: ErrorEvent) => void;

	/**
	 * This is an abstract class for implementing a socket provider (e.g. WebSocket, IPC). It extends the EIP-1193 provider {@link EIP1193Provider}.
	 * @param socketPath - The path to the socket (e.g. /ipc/path or ws://localhost:8546)
	 * @param socketOptions - The options for the socket connection. Its type is supposed to be specified in the inherited classes.
	 * @param reconnectOptions - The options for the socket reconnection {@link ReconnectOptions}
	 */
	public constructor(
		socketPath: string,
		socketOptions?: unknown,
		reconnectOptions?: Partial<ReconnectOptions>,
	) {
		super();
		this._connectionStatus = 'connecting';

		// Message handlers. Due to bounding of `this` and removing the listeners we have to keep it's reference.
		this._onMessageHandler = this._onMessage.bind(this);
		this._onOpenHandler = this._onConnect.bind(this);
		this._onCloseHandler = this._onCloseEvent.bind(this);
		this._onErrorHandler = this._onError.bind(this);

		if (!this._validateProviderPath(socketPath)) throw new InvalidClientError(socketPath);

		this._socketPath = socketPath;
		this._socketOptions = socketOptions;
		this._reconnectOptions = {
			...DEFAULT_RECONNECTION_OPTIONS,
			...(reconnectOptions ?? {}),
		};

		this._pendingRequestsQueue = new Map<JsonRpcId, SocketRequestItem<any, any, any>>();
		this._sentRequestsQueue = new Map<JsonRpcId, SocketRequestItem<any, any, any>>();

		this._init();
		this.connect();
		this.chunkResponseParser = new ChunkResponseParser(
			this._eventEmitter,
			this._reconnectOptions.autoReconnect,
		);
		this.chunkResponseParser.onError(() => {
			this._clearQueues();
		});
		this.isReconnecting = false;
	}

	protected _init() {
		this._reconnectAttempts = 0;
	}

	/**
	 * Try to establish a connection to the socket
	 */
	public connect(): void {
		try {
			this._openSocketConnection();
			this._connectionStatus = 'connecting';
			this._addSocketListeners();
		} catch (e) {
			if (!this.isReconnecting) {
				this._connectionStatus = 'disconnected';
				if (e && (e as Error).message) {
					throw new ConnectionError(
						`Error while connecting to ${this._socketPath}. Reason: ${
							(e as Error).message
						}`,
					);
				} else {
					throw new InvalidClientError(this._socketPath);
				}
			} else {
				setImmediate(() => {
					this._reconnect();
				});
			}
		}
	}

	protected abstract _openSocketConnection(): void;
	protected abstract _addSocketListeners(): void;

	protected abstract _removeSocketListeners(): void;

	protected abstract _onCloseEvent(_event: unknown): void;

	protected abstract _sendToSocket(_payload: Web3APIPayload<API, any>): void;

	protected abstract _parseResponses(_event: MessageEvent): JsonRpcResponse[];

	protected abstract _closeSocketConnection(_code?: number, _data?: string): void;

	// eslint-disable-next-line class-methods-use-this
	protected _validateProviderPath(path: string): boolean {
		return !!path;
	}

	/**
	 *
	 * @returns the pendingRequestQueue size
	 */
	// eslint-disable-next-line class-methods-use-this
	public getPendingRequestQueueSize() {
		return this._pendingRequestsQueue.size;
	}

	/**
	 *
	 * @returns the sendPendingRequests size
	 */
	// eslint-disable-next-line class-methods-use-this
	public getSentRequestsQueueSize() {
		return this._sentRequestsQueue.size;
	}

	/**
	 *
	 * @returns `true` if the socket supports subscriptions
	 */
	// eslint-disable-next-line class-methods-use-this
	public supportsSubscriptions(): boolean {
		return true;
	}

	/**
	 * Registers a listener for the specified event type.
	 * @param type - The event type to listen for
	 * @param listener - The callback to be invoked when the event is emitted
	 */
	public on(
		type: 'disconnect',
		listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>,
	): void;
	public on(
		type: 'connect',
		listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>,
	): void;
	public on(type: 'chainChanged', listener: Web3Eip1193ProviderEventCallback<string>): void;
	public on(type: 'accountsChanged', listener: Web3Eip1193ProviderEventCallback<string[]>): void;
	public on<T = JsonRpcResult>(
		type: 'message',
		listener:
			| Web3Eip1193ProviderEventCallback<ProviderMessage>
			| Web3ProviderMessageEventCallback<T>,
	): void;
	public on<T = JsonRpcResult>(
		type: string,
		listener: Web3Eip1193ProviderEventCallback<unknown> | Web3ProviderEventCallback<T>,
	): void;
	public on<T = JsonRpcResult, P = unknown>(
		type: string | Eip1193EventName,
		listener:
			| Web3Eip1193ProviderEventCallback<P>
			| Web3ProviderMessageEventCallback<T>
			| Web3ProviderEventCallback<T>,
	): void {
		this._eventEmitter.on(type, listener);
	}

	/**
	 * Registers a listener for the specified event type that will be invoked at most once.
	 * @param type  - The event type to listen for
	 * @param listener - The callback to be invoked when the event is emitted
	 */
	public once(
		type: 'disconnect',
		listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>,
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
	public once<T = JsonRpcResult>(
		type: 'message',
		listener:
			| Web3Eip1193ProviderEventCallback<ProviderMessage>
			| Web3ProviderMessageEventCallback<T>,
	): void;
	public once<T = JsonRpcResult>(
		type: string,
		listener: Web3Eip1193ProviderEventCallback<unknown> | Web3ProviderEventCallback<T>,
	): void;
	public once<T = JsonRpcResult, P = unknown>(
		type: string | Eip1193EventName,
		listener:
			| Web3Eip1193ProviderEventCallback<P>
			| Web3ProviderMessageEventCallback<T>
			| Web3ProviderEventCallback<T>,
	): void {
		this._eventEmitter.once(type, listener);
	}

	/**
	 *  Removes a listener for the specified event type.
	 * @param type - The event type to remove the listener for
	 * @param listener - The callback to be executed
	 */
	public removeListener(
		type: 'disconnect',
		listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>,
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
	public removeListener<T = JsonRpcResult>(
		type: 'message',
		listener:
			| Web3Eip1193ProviderEventCallback<ProviderMessage>
			| Web3ProviderMessageEventCallback<T>,
	): void;
	public removeListener<T = JsonRpcResult>(
		type: string,
		listener: Web3Eip1193ProviderEventCallback<unknown> | Web3ProviderEventCallback<T>,
	): void;
	public removeListener<T = JsonRpcResult, P = unknown>(
		type: string | Eip1193EventName,
		listener:
			| Web3Eip1193ProviderEventCallback<P>
			| Web3ProviderMessageEventCallback<T>
			| Web3ProviderEventCallback<T>,
	): void {
		this._eventEmitter.removeListener(type, listener);
	}

	protected _onDisconnect(code: number, data?: string) {
		this._connectionStatus = 'disconnected';
		super._onDisconnect(code, data);
	}

	/**
	 * Disconnects the socket
	 * @param code - The code to be sent to the server
	 * @param data - The data to be sent to the server
	 */
	public disconnect(code?: number, data?: string): void {
		const disconnectCode = code ?? NORMAL_CLOSE_CODE;
		this._removeSocketListeners();
		if (this.getStatus() !== 'disconnected') {
			this._closeSocketConnection(disconnectCode, data);
		}
		this._onDisconnect(disconnectCode, data);
	}

	/**
	 * Safely disconnects the socket, async and waits for request size to be 0 before disconnecting
	 * @param forceDisconnect - If true, will clear queue after 5 attempts of waiting for both pending and sent queue to be 0
	 * @param ms - Determines the ms of setInterval
	 * @param code - The code to be sent to the server
	 * @param data - The data to be sent to the server
	 */
	public async safeDisconnect(code?: number, data?: string, forceDisconnect = false, ms = 1000) {
		let retryAttempt = 0;
		const checkQueue = async () =>
			new Promise(resolve => {
				const interval = setInterval(() => {
					if (forceDisconnect && retryAttempt >= 5) {
						this.clearQueues();
					}
					if (
						this.getPendingRequestQueueSize() === 0 &&
						this.getSentRequestsQueueSize() === 0
					) {
						clearInterval(interval);
						resolve(true);
					}
					retryAttempt += 1;
				}, ms);
			});

		await checkQueue();
		this.disconnect(code, data);
	}

	/**
	 * Removes all listeners for the specified event type.
	 * @param type - The event type to remove the listeners for
	 */
	public removeAllListeners(type: string): void {
		this._eventEmitter.removeAllListeners(type);
	}

	protected _onError(event: ErrorEvent): void {
		// do not emit error while trying to reconnect
		if (this.isReconnecting) {
			this._reconnect();
		} else {
			this._eventEmitter.emit('error', event);
		}
	}

	/**
	 * Resets the socket, removing all listeners and pending requests
	 */
	public reset(): void {
		this._sentRequestsQueue.clear();
		this._pendingRequestsQueue.clear();

		this._init();
		this._removeSocketListeners();
		this._addSocketListeners();
	}

	protected _reconnect(): void {
		if (this.isReconnecting) {
			return;
		}
		this.isReconnecting = true;

		if (this._sentRequestsQueue.size > 0) {
			this._sentRequestsQueue.forEach(
				(request: SocketRequestItem<any, any, any>, key: JsonRpcId) => {
					request.deferredPromise.reject(new PendingRequestsOnReconnectingError());
					this._sentRequestsQueue.delete(key);
				},
			);
		}

		if (this._reconnectAttempts < this._reconnectOptions.maxAttempts) {
			this._reconnectAttempts += 1;
			setTimeout(() => {
				this._removeSocketListeners();
				this.connect(); // this can error out
				this.isReconnecting = false;
			}, this._reconnectOptions.delay);
		} else {
			this.isReconnecting = false;
			this._clearQueues();
			this._removeSocketListeners();
			this._eventEmitter.emit(
				'error',
				new MaxAttemptsReachedOnReconnectingError(this._reconnectOptions.maxAttempts),
			);
		}
	}

	/**
	 *  Creates a request object to be sent to the server
	 */
	public async request<
		Method extends Web3APIMethod<API>,
		ResultType = Web3APIReturnType<API, Method>,
	>(request: Web3APIPayload<API, Method>): Promise<JsonRpcResponseWithResult<ResultType>> {
		if (isNullish(this._socketConnection)) {
			throw new Error('Connection is undefined');
		}
		// if socket disconnected - open connection
		if (this.getStatus() === 'disconnected') {
			this.connect();
		}

		const requestId = jsonRpc.isBatchRequest(request)
			? (request as unknown as JsonRpcBatchRequest)[0].id
			: (request as unknown as JsonRpcRequest).id;

		if (!requestId) {
			throw new Web3WSProviderError('Request Id not defined');
		}

		if (this._sentRequestsQueue.has(requestId)) {
			throw new RequestAlreadySentError(requestId);
		}
		const deferredPromise = new Web3DeferredPromise<JsonRpcResponseWithResult<ResultType>>();
		deferredPromise.catch(error => {
			this._eventEmitter.emit('error', error);
		});
		const reqItem: SocketRequestItem<API, Method, JsonRpcResponseWithResult<ResultType>> = {
			payload: request,
			deferredPromise,
		};

		if (this.getStatus() === 'connecting') {
			this._pendingRequestsQueue.set(requestId, reqItem);

			return reqItem.deferredPromise;
		}

		this._sentRequestsQueue.set(requestId, reqItem);

		try {
			this._sendToSocket(reqItem.payload);
		} catch (error) {
			this._sentRequestsQueue.delete(requestId);

			this._eventEmitter.emit('error', error);
		}

		return deferredPromise;
	}

	protected _onConnect() {
		this._connectionStatus = 'connected';
		this._reconnectAttempts = 0;
		super._onConnect();
		this._sendPendingRequests();
	}

	private _sendPendingRequests() {
		for (const [id, value] of this._pendingRequestsQueue.entries()) {
			try {
				this._sendToSocket(value.payload as Web3APIPayload<API, any>);
				this._pendingRequestsQueue.delete(id);
				this._sentRequestsQueue.set(id, value);
			} catch (error) {
				// catches if sendTosocket fails
				this._pendingRequestsQueue.delete(id);
				this._eventEmitter.emit('error', error);
			}
		}
	}

	protected _onMessage(event: MessageEvent): void {
		const responses = this._parseResponses(event);
		if (isNullish(responses) || responses.length === 0) {
			return;
		}

		for (const response of responses) {
			if (
				jsonRpc.isResponseWithNotification(response as JsonRpcNotification) &&
				(response as JsonRpcNotification).method.endsWith('_subscription')
			) {
				this._eventEmitter.emit('message', response);
				return;
			}

			const requestId = jsonRpc.isBatchResponse(response)
				? (response as unknown as JsonRpcBatchResponse)[0].id
				: (response as unknown as JsonRpcResponseWithResult).id;

			const requestItem = this._sentRequestsQueue.get(requestId);

			if (!requestItem) {
				return;
			}

			if (
				jsonRpc.isBatchResponse(response) ||
				jsonRpc.isResponseWithResult(response) ||
				jsonRpc.isResponseWithError(response)
			) {
				this._eventEmitter.emit('message', response);
				requestItem.deferredPromise.resolve(response);
			}

			this._sentRequestsQueue.delete(requestId);
		}
	}

	public clearQueues(event?: ConnectionEvent) {
		this._clearQueues(event);
	}

	protected _clearQueues(event?: ConnectionEvent) {
		if (this._pendingRequestsQueue.size > 0) {
			this._pendingRequestsQueue.forEach(
				(request: SocketRequestItem<any, any, any>, key: JsonRpcId) => {
					request.deferredPromise.reject(new ConnectionNotOpenError(event));
					this._pendingRequestsQueue.delete(key);
				},
			);
		}

		if (this._sentRequestsQueue.size > 0) {
			this._sentRequestsQueue.forEach(
				(request: SocketRequestItem<any, any, any>, key: JsonRpcId) => {
					request.deferredPromise.reject(new ConnectionNotOpenError(event));
					this._sentRequestsQueue.delete(key);
				},
			);
		}

		this._removeSocketListeners();
	}
}
