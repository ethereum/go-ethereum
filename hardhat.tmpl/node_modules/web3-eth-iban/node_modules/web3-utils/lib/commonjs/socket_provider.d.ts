import { ConnectionEvent, EthExecutionAPI, JsonRpcId, JsonRpcResponse, JsonRpcResponseWithResult, JsonRpcResult, ProviderConnectInfo, ProviderMessage, ProviderRpcError, SocketRequestItem, Web3APIMethod, Web3APIPayload, Web3APIReturnType, Web3APISpec, Web3Eip1193ProviderEventCallback, Web3ProviderEventCallback, Web3ProviderMessageEventCallback, Web3ProviderStatus } from 'web3-types';
import { Eip1193Provider } from './web3_eip1193_provider.js';
import { ChunkResponseParser } from './chunk_response_parser.js';
export type ReconnectOptions = {
    autoReconnect: boolean;
    delay: number;
    maxAttempts: number;
};
export declare abstract class SocketProvider<MessageEvent, CloseEvent, ErrorEvent, API extends Web3APISpec = EthExecutionAPI> extends Eip1193Provider<API> {
    protected isReconnecting: boolean;
    protected readonly _socketPath: string;
    protected readonly chunkResponseParser: ChunkResponseParser;
    protected readonly _pendingRequestsQueue: Map<JsonRpcId, SocketRequestItem<any, any, any>>;
    protected readonly _sentRequestsQueue: Map<JsonRpcId, SocketRequestItem<any, any, any>>;
    protected _reconnectAttempts: number;
    protected readonly _socketOptions?: unknown;
    protected readonly _reconnectOptions: ReconnectOptions;
    protected _socketConnection?: unknown;
    get SocketConnection(): unknown;
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
    constructor(socketPath: string, socketOptions?: unknown, reconnectOptions?: Partial<ReconnectOptions>);
    protected _init(): void;
    /**
     * Try to establish a connection to the socket
     */
    connect(): void;
    protected abstract _openSocketConnection(): void;
    protected abstract _addSocketListeners(): void;
    protected abstract _removeSocketListeners(): void;
    protected abstract _onCloseEvent(_event: unknown): void;
    protected abstract _sendToSocket(_payload: Web3APIPayload<API, any>): void;
    protected abstract _parseResponses(_event: MessageEvent): JsonRpcResponse[];
    protected abstract _closeSocketConnection(_code?: number, _data?: string): void;
    protected _validateProviderPath(path: string): boolean;
    /**
     *
     * @returns the pendingRequestQueue size
     */
    getPendingRequestQueueSize(): number;
    /**
     *
     * @returns the sendPendingRequests size
     */
    getSentRequestsQueueSize(): number;
    /**
     *
     * @returns `true` if the socket supports subscriptions
     */
    supportsSubscriptions(): boolean;
    /**
     * Registers a listener for the specified event type.
     * @param type - The event type to listen for
     * @param listener - The callback to be invoked when the event is emitted
     */
    on(type: 'disconnect', listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>): void;
    on(type: 'connect', listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>): void;
    on(type: 'chainChanged', listener: Web3Eip1193ProviderEventCallback<string>): void;
    on(type: 'accountsChanged', listener: Web3Eip1193ProviderEventCallback<string[]>): void;
    on<T = JsonRpcResult>(type: 'message', listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderMessageEventCallback<T>): void;
    on<T = JsonRpcResult>(type: string, listener: Web3Eip1193ProviderEventCallback<unknown> | Web3ProviderEventCallback<T>): void;
    /**
     * Registers a listener for the specified event type that will be invoked at most once.
     * @param type  - The event type to listen for
     * @param listener - The callback to be invoked when the event is emitted
     */
    once(type: 'disconnect', listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>): void;
    once(type: 'connect', listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>): void;
    once(type: 'chainChanged', listener: Web3Eip1193ProviderEventCallback<string>): void;
    once(type: 'accountsChanged', listener: Web3Eip1193ProviderEventCallback<string[]>): void;
    once<T = JsonRpcResult>(type: 'message', listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderMessageEventCallback<T>): void;
    once<T = JsonRpcResult>(type: string, listener: Web3Eip1193ProviderEventCallback<unknown> | Web3ProviderEventCallback<T>): void;
    /**
     *  Removes a listener for the specified event type.
     * @param type - The event type to remove the listener for
     * @param listener - The callback to be executed
     */
    removeListener(type: 'disconnect', listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>): void;
    removeListener(type: 'connect', listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>): void;
    removeListener(type: 'chainChanged', listener: Web3Eip1193ProviderEventCallback<string>): void;
    removeListener(type: 'accountsChanged', listener: Web3Eip1193ProviderEventCallback<string[]>): void;
    removeListener<T = JsonRpcResult>(type: 'message', listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderMessageEventCallback<T>): void;
    removeListener<T = JsonRpcResult>(type: string, listener: Web3Eip1193ProviderEventCallback<unknown> | Web3ProviderEventCallback<T>): void;
    protected _onDisconnect(code: number, data?: string): void;
    /**
     * Disconnects the socket
     * @param code - The code to be sent to the server
     * @param data - The data to be sent to the server
     */
    disconnect(code?: number, data?: string): void;
    /**
     * Safely disconnects the socket, async and waits for request size to be 0 before disconnecting
     * @param forceDisconnect - If true, will clear queue after 5 attempts of waiting for both pending and sent queue to be 0
     * @param ms - Determines the ms of setInterval
     * @param code - The code to be sent to the server
     * @param data - The data to be sent to the server
     */
    safeDisconnect(code?: number, data?: string, forceDisconnect?: boolean, ms?: number): Promise<void>;
    /**
     * Removes all listeners for the specified event type.
     * @param type - The event type to remove the listeners for
     */
    removeAllListeners(type: string): void;
    protected _onError(event: ErrorEvent): void;
    /**
     * Resets the socket, removing all listeners and pending requests
     */
    reset(): void;
    protected _reconnect(): void;
    /**
     *  Creates a request object to be sent to the server
     */
    request<Method extends Web3APIMethod<API>, ResultType = Web3APIReturnType<API, Method>>(request: Web3APIPayload<API, Method>): Promise<JsonRpcResponseWithResult<ResultType>>;
    protected _onConnect(): void;
    private _sendPendingRequests;
    protected _onMessage(event: MessageEvent): void;
    clearQueues(event?: ConnectionEvent): void;
    protected _clearQueues(event?: ConnectionEvent): void;
}
