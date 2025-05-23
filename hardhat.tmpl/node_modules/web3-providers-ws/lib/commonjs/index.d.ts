/// <reference types="ws" />
/// <reference types="node" />
import { ClientRequestArgs } from 'http';
import WebSocket, { ClientOptions, CloseEvent } from 'isomorphic-ws';
import { EthExecutionAPI, Web3APIMethod, Web3APIPayload, Web3APISpec, Web3ProviderStatus } from 'web3-types';
import { ReconnectOptions, SocketProvider } from 'web3-utils';
export { ClientRequestArgs } from 'http';
export { ClientOptions } from 'isomorphic-ws';
/**
 * Use WebSocketProvider to connect to a Node using a WebSocket connection, i.e. over the `ws` or `wss` protocol.
 *
 * @example
 * ```ts
 * const provider = new WebSocketProvider(
 * 		`ws://localhost:8545`,
 * 		{
 * 			headers: {
 * 				// to provide the API key if the Node requires the key to be inside the `headers` for example:
 * 				'x-api-key': '<Api key>',
 * 			},
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
 * const provider = new WebSocketProvider(
 * 		`ws://localhost:8545`,
 * 		{},
 * 		{
 * 			delay: 500,
 * 			autoReconnect: true,
 * 			maxAttempts: 10,
 * 		},
 * 	);
 * ```
 */
export default class WebSocketProvider<API extends Web3APISpec = EthExecutionAPI> extends SocketProvider<WebSocket.MessageEvent, WebSocket.CloseEvent, WebSocket.ErrorEvent, API> {
    protected readonly _socketOptions?: ClientOptions | ClientRequestArgs;
    protected _socketConnection?: WebSocket;
    protected _validateProviderPath(providerUrl: string): boolean;
    /**
     * This is a class used for Web Socket connections. It extends the abstract class SocketProvider {@link SocketProvider} that extends the EIP-1193 provider {@link EIP1193Provider}.
     * @param socketPath - The path to the Web Socket.
     * @param socketOptions - The options for the Web Socket client.
     * @param reconnectOptions - The options for the socket reconnection {@link ReconnectOptions}
     */
    constructor(socketPath: string, socketOptions?: ClientOptions | ClientRequestArgs, reconnectOptions?: Partial<ReconnectOptions>);
    getStatus(): Web3ProviderStatus;
    protected _openSocketConnection(): void;
    protected _closeSocketConnection(code?: number, data?: string): void;
    protected _sendToSocket<Method extends Web3APIMethod<API>>(payload: Web3APIPayload<API, Method>): void;
    protected _parseResponses(event: WebSocket.MessageEvent): import("web3-types").JsonRpcResponse<import("web3-types").JsonRpcResult, import("web3-types").JsonRpcResult>[];
    protected _addSocketListeners(): void;
    protected _removeSocketListeners(): void;
    protected _onCloseEvent(event: CloseEvent): void;
}
export { WebSocketProvider };
