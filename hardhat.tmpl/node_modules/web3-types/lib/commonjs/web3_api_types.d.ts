import { JsonRpcId, JsonRpcIdentifier } from './json_rpc_types.js';
export interface ProviderMessage {
    readonly type: string;
    readonly data: unknown;
}
export interface EthSubscription extends ProviderMessage {
    readonly type: 'eth_subscription';
    readonly data: {
        readonly subscription: string;
        readonly result: unknown;
    };
}
export interface ProviderRpcError extends Error {
    code: number;
    data?: unknown;
}
export interface ProviderConnectInfo {
    readonly chainId: string;
}
export type Web3APISpec = Record<string, (...params: any) => any> | unknown;
export type Web3APIMethod<T extends Web3APISpec> = string & keyof Exclude<T, unknown>;
export type Web3APIParams<API extends Web3APISpec, Method extends Web3APIMethod<API>> = API extends Exclude<Web3APISpec, unknown> ? Parameters<API[Method]> : unknown;
export interface Web3APIRequest<API extends Web3APISpec, Method extends Web3APIMethod<API>> {
    method: Method | string;
    params?: Web3APIParams<API, Method> | readonly unknown[] | object;
}
export interface Web3APIPayload<API extends Web3APISpec, Method extends Web3APIMethod<API>> extends Web3APIRequest<API, Method> {
    readonly jsonrpc?: JsonRpcIdentifier;
    readonly id?: JsonRpcId;
    readonly requestOptions?: unknown;
}
export type Web3APIReturnType<API extends Web3APISpec, Method extends Web3APIMethod<API>> = API extends Record<string, (...params: any) => any> ? ReturnType<API[Method]> : any;
