/// <reference types="node" />
import { Socket, SocketConstructorOpts } from 'net';
import { ReconnectOptions, SocketProvider } from 'web3-utils';
import { EthExecutionAPI, Web3APIMethod, Web3APIPayload, Web3APISpec, Web3ProviderStatus } from 'web3-types';
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
export default class IpcProvider<API extends Web3APISpec = EthExecutionAPI> extends SocketProvider<Uint8Array | string, CloseEvent, Error, API> {
    protected readonly _socketOptions?: SocketConstructorOpts;
    protected _socketConnection?: Socket;
    /**
     * This is a class used for IPC connections. It extends the abstract class SocketProvider {@link SocketProvider} that extends the EIP-1193 provider {@link EIP1193Provider}.
     * @param socketPath - The path to the IPC socket.
     * @param socketOptions - The options for the IPC socket connection.
     * @param reconnectOptions - The options for the socket reconnection {@link ReconnectOptions}
     */
    constructor(socketPath: string, socketOptions?: SocketConstructorOpts, reconnectOptions?: Partial<ReconnectOptions>);
    getStatus(): Web3ProviderStatus;
    protected _openSocketConnection(): void;
    protected _closeSocketConnection(code: number, data?: string): void;
    protected _sendToSocket<Method extends Web3APIMethod<API>>(payload: Web3APIPayload<API, Method>): void;
    protected _parseResponses(e: Uint8Array | string): import("web3-types").JsonRpcResponse<import("web3-types").JsonRpcResult, import("web3-types").JsonRpcResult>[];
    protected _addSocketListeners(): void;
    protected _removeSocketListeners(): void;
    protected _onCloseEvent(event: CloseEvent): void;
    protected _onClose(event: CloseEvent): void;
}
export { IpcProvider };
