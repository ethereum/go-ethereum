import { EthExecutionAPI, JsonRpcResponseWithResult, Web3APIMethod, Web3APIPayload, Web3APIReturnType, Web3APISpec } from 'web3-types';
import { HttpProviderOptions } from 'web3-providers-http';
import { Transport, Network, SocketOptions } from './types.js';
import { Web3ExternalProvider } from './web3_provider.js';
export declare class QuickNodeProvider<API extends Web3APISpec = EthExecutionAPI> extends Web3ExternalProvider {
    constructor(network?: Network, transport?: Transport, token?: string, host?: string, providerConfigOptions?: HttpProviderOptions | SocketOptions);
    request<Method extends Web3APIMethod<API>, ResultType = Web3APIReturnType<API, Method>>(payload: Web3APIPayload<EthExecutionAPI, Method>, requestOptions?: RequestInit): Promise<JsonRpcResponseWithResult<ResultType>>;
    getRPCURL(network: Network, transport: Transport, _token: string, _host: string): string;
}
//# sourceMappingURL=web3_provider_quicknode.d.ts.map