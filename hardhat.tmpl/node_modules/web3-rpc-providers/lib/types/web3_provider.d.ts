import { HttpProviderOptions } from 'web3-providers-http';
import { EthExecutionAPI, JsonRpcResult, ProviderConnectInfo, ProviderMessage, ProviderRpcError, Web3APIMethod, Web3APIPayload, Web3APIReturnType, Web3APISpec, Web3BaseProvider, Web3Eip1193ProviderEventCallback, Web3ProviderEventCallback, Web3ProviderMessageEventCallback, Web3ProviderStatus, JsonRpcResponseWithResult } from 'web3-types';
import { Eip1193Provider } from 'web3-utils';
import { Transport, Network, SocketOptions } from './types.js';
export declare abstract class Web3ExternalProvider<API extends Web3APISpec = EthExecutionAPI> extends Eip1193Provider {
    provider: Web3BaseProvider;
    readonly transport: Transport;
    abstract getRPCURL(network: Network, transport: Transport, token: string, host: string): string;
    constructor(network: Network, transport: Transport, token: string, host: string, providerConfigOptions?: HttpProviderOptions | SocketOptions);
    request<Method extends Web3APIMethod<API>, ResultType = Web3APIReturnType<API, Method>>(payload: Web3APIPayload<EthExecutionAPI, Method>, requestOptions?: RequestInit): Promise<JsonRpcResponseWithResult<ResultType>>;
    getStatus(): Web3ProviderStatus;
    supportsSubscriptions(): boolean;
    once(type: 'disconnect', listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>): void;
    once<T = JsonRpcResult>(type: string, listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderEventCallback<T>): void;
    once(type: 'connect', listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>): void;
    once(type: 'chainChanged', listener: Web3Eip1193ProviderEventCallback<string>): void;
    once(type: 'accountsChanged', listener: Web3Eip1193ProviderEventCallback<string[]>): void;
    removeAllListeners?(_type: string): void;
    connect(): void;
    disconnect(_code?: number | undefined, _data?: string | undefined): void;
    reset(): void;
    on(type: 'disconnect', listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>): void;
    on<T = JsonRpcResult>(type: string, listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderMessageEventCallback<T>): void;
    on<T = JsonRpcResult>(type: string, listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderMessageEventCallback<T>): void;
    on(type: 'connect', listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>): void;
    on(type: 'chainChanged', listener: Web3Eip1193ProviderEventCallback<string>): void;
    on(type: 'accountsChanged', listener: Web3Eip1193ProviderEventCallback<string[]>): void;
    removeListener(type: 'disconnect', listener: Web3Eip1193ProviderEventCallback<ProviderRpcError>): void;
    removeListener<T = JsonRpcResult>(type: string, listener: Web3Eip1193ProviderEventCallback<ProviderMessage> | Web3ProviderEventCallback<T>): void;
    removeListener(type: 'connect', listener: Web3Eip1193ProviderEventCallback<ProviderConnectInfo>): void;
    removeListener(type: 'chainChanged', listener: Web3Eip1193ProviderEventCallback<string>): void;
    removeListener(type: 'accountsChanged', listener: Web3Eip1193ProviderEventCallback<string[]>): void;
}
//# sourceMappingURL=web3_provider.d.ts.map