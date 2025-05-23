import { Socket } from 'net';
import { Web3Error } from './error_types.js';
import { EthExecutionAPI } from './apis/eth_execution_api.js';
import { JsonRpcNotification, JsonRpcPayload, JsonRpcResponse, JsonRpcResponseWithError, JsonRpcResponseWithResult, JsonRpcResult, JsonRpcSubscriptionResult } from './json_rpc_types';
import { Web3APISpec, Web3APIMethod, Web3APIReturnType, Web3APIPayload, ProviderConnectInfo, ProviderRpcError, ProviderMessage } from './web3_api_types';
import { Web3EthExecutionAPI } from './apis/web3_eth_execution_api';
import { Web3DeferredPromiseInterface } from './web3_deferred_promise_type';
declare const symbol: unique symbol;
export interface SocketRequestItem<API extends Web3APISpec, Method extends Web3APIMethod<API>, ResponseType> {
    payload: Web3APIPayload<API, Method>;
    deferredPromise: Web3DeferredPromiseInterface<ResponseType>;
}
export type Web3ProviderStatus = 'connecting' | 'connected' | 'disconnected';
export type Web3ProviderEventCallback<T = JsonRpcResult> = (error: Error | ProviderRpcError | undefined, result?: JsonRpcSubscriptionResult | JsonRpcNotification<T>) => void;
export type Web3ProviderMessageEventCallback<T = JsonRpcResult> = (result?: JsonRpcSubscriptionResult | JsonRpcNotification<T>) => void;
export type Web3Eip1193ProviderEventCallback<T> = (data: T) => void;
export type Web3ProviderRequestCallback<ResultType = unknown> = (err?: Error | Web3Error | null | JsonRpcResponseWithError<Error>, response?: JsonRpcResponseWithResult<ResultType>) => void;
export interface LegacySendProvider {
    send<R = JsonRpcResult, P = unknown>(payload: JsonRpcPayload<P>, callback: (err: Error | null, response?: JsonRpcResponse<R>) => void): void;
}
export interface LegacySendAsyncProvider {
    sendAsync<R = JsonRpcResult, P = unknown>(payload: JsonRpcPayload<P>): Promise<JsonRpcResponse<R>>;
}
export interface LegacyRequestProvider {
    request<R = JsonRpcResult, P = unknown>(payload: JsonRpcPayload<P>, callback: (err: Error | null, response: JsonRpcResponse<R>) => void): void;
}
export interface SimpleProvider<API extends Web3APISpec> {
    request<Method extends Web3APIMethod<API>, ResponseType = Web3APIReturnType<API, Method>>(args: Web3APIPayload<API, Method>): Promise<JsonRpcResponseWithResult<ResponseType> | unknown>;
}
export interface ProviderInfo {
    chainId: string;
}
export type ProviderChainId = string;
export type ProviderAccounts = string[];
export type Eip1193EventName = 'connect' | 'disconnect' | 'message' | 'chainChanged' | 'accountsChanged';
export interface EIP1193Provider<API extends Web3APISpec> extends SimpleProvider<API> {
    on(event: 'connect', listener: (info: ProviderInfo) => void): void;
    on(event: 'disconnect', listener: (error: ProviderRpcError) => void): void;
    on(event: 'message', listener: (message: ProviderMessage) => void): void;
    on(event: 'chainChanged', listener: (chainId: ProviderChainId) => void): void;
    on(event: 'accountsChanged', listener: (accounts: ProviderAccounts) => void): void;
    removeListener(event: 'connect', listener: (info: ProviderInfo) => void): void;
    removeListener(event: 'disconnect', listener: (error: ProviderRpcError) => void): void;
    removeListener(event: 'message', listener: (message: ProviderMessage) => void): void;
    removeListener(event: 'chainChanged', listener: (chainId: ProviderChainId) => void): void;
    removeListener(event: 'accountsChanged', listener: (accounts: ProviderAccounts) => void): void;
}
export interface MetaMaskProvider<API extends Web3APISpec> extends SimpleProvider<API> {
    on(event: 'connect', listener: (info: ProviderInfo) => void): void;
    on(event: 'disconnect', listener: (error: ProviderRpcError) => void): void;
    on(event: 'message', listener: (message: ProviderMessage) => void): void;
    on(event: 'chainChanged', listener: (chainId: ProviderChainId) => void): void;
    on(event: 'accountsChanged', listener: (accounts: ProviderAccounts) => void): void;
    removeListener(event: 'connect', listener: (info: ProviderInfo) => void): void;
    removeListener(event: 'disconnect', listener: (error: ProviderRpcError) => void): void;
    removeListener(event: 'message', listener: (message: ProviderMessage) => void): void;
    removeListener(event: 'chainChanged', listener: (chainId: ProviderChainId) => void): void;
    removeListener(event: 'accountsChanged', listener: (accounts: ProviderAccounts) => void): void;
    isMetaMask: boolean;
}
export type Eip1193Compatible<API extends Web3APISpec = EthExecutionAPI> = Omit<Omit<Web3BaseProvider, 'request'>, 'asEIP1193Provider'> & {
    request<Method extends Web3APIMethod<API>, ResultType = Web3APIReturnType<API, Method> | unknown>(request: Web3APIPayload<API, Method>): Promise<ResultType>;
};
export declare abstract class Web3BaseProvider<API extends Web3APISpec = EthExecutionAPI> implements LegacySendProvider, LegacySendAsyncProvider, EIP1193Provider<API> {
    static isWeb3Provider(provider: unknown): boolean;
    get [symbol](): boolean;
    abstract getStatus(): Web3ProviderStatus;
    abstract supportsSubscriptions(): boolean;
    /**
     * @deprecated Please use `.request` instead.
     * @param payload - Request Payload
     * @param callback - Callback
     */
    send<ResultType = JsonRpcResult, P = unknown>(payload: JsonRpcPayload<P>, callback: (err: Error | null, response?: JsonRpcResponse<ResultType>) => void): void;
    /**
     * @deprecated Please use `.request` instead.
     * @param payload - Request Payload
     */
    sendAsync<R = JsonRpcResult, P = unknown>(payload: JsonRpcPayload<P>): Promise<JsonRpcResponse<R>>;
    /**
     * Modify the return type of the request method to be fully compatible with EIP-1193
     *
     * [deprecated] In the future major releases (\>= v5) all providers are supposed to be fully compatible with EIP-1193.
     * So this method will not be needed and would not be available in the future.
     *
     * @returns A new instance of the provider with the request method fully compatible with EIP-1193
     *
     * @example
     * ```ts
     * const provider = new Web3HttpProvider('http://localhost:8545');
     * const fullyCompatibleProvider = provider.asEIP1193Provider();
     * const result = await fullyCompatibleProvider.request({ method: 'eth_getBalance' });
     * console.log(result); // '0x0234c8a3397aab58' or something like that
     * ```
     */
    asEIP1193Provider(): Eip1193Compatible<API>;
    abstract request<Method extends Web3APIMethod<API>, ResultType = Web3APIReturnType<API, Method> | unknown>(args: Web3APIPayload<API, Method>): Promise<JsonRpcResponseWithResult<ResultType>>;
    abstract on(type: 'disconnect', listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>): void;
    abstract on<T = JsonRpcResult>(type: 'message' | string, listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderMessageEventCallback<T>): void;
    abstract on<T = JsonRpcResult>(type: 'data' | string, listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderMessageEventCallback<T>): void;
    abstract on(type: 'connect', listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>): void;
    abstract on(type: 'chainChanged', listener: Web3Eip1193ProviderEventCallback<string>): void;
    abstract on(type: 'accountsChanged', listener: Web3Eip1193ProviderEventCallback<string[]>): void;
    abstract removeListener(type: 'disconnect', listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>): void;
    abstract removeListener<T = JsonRpcResult>(type: 'message' | string, listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderEventCallback<T>): void;
    abstract removeListener(type: 'connect', listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>): void;
    abstract removeListener(type: 'chainChanged', listener: Web3Eip1193ProviderEventCallback<string>): void;
    abstract removeListener(type: 'accountsChanged', listener: Web3Eip1193ProviderEventCallback<string[]>): void;
    abstract once(type: 'disconnect', listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>): void;
    abstract once<T = JsonRpcResult>(type: 'message' | string, listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderEventCallback<T>): void;
    abstract once(type: 'connect', listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>): void;
    abstract once(type: 'chainChanged', listener: Web3Eip1193ProviderEventCallback<string>): void;
    abstract once(type: 'accountsChanged', listener: Web3Eip1193ProviderEventCallback<string[]>): void;
    abstract removeAllListeners?(type: string): void;
    abstract connect(): void;
    abstract disconnect(code?: number, data?: string): void;
    abstract reset(): void;
}
export type SupportedProviders<API extends Web3APISpec = Web3EthExecutionAPI> = EIP1193Provider<API> | Web3BaseProvider<API> | LegacyRequestProvider | LegacySendProvider | LegacySendAsyncProvider | SimpleProvider<API> | MetaMaskProvider<API>;
export type Web3BaseProviderConstructor = new <API extends Web3APISpec>(url: string, net?: Socket) => Web3BaseProvider<API>;
export {};
//# sourceMappingURL=web3_base_provider.d.ts.map