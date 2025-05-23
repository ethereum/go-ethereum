import { EthExecutionAPI, JsonRpcResponseWithResult, Web3APIMethod, Web3APIPayload, Web3APIReturnType, Web3APISpec, Web3BaseProvider, Web3ProviderStatus } from 'web3-types';
import { HttpProviderOptions } from './types.js';
export { HttpProviderOptions } from './types.js';
export default class HttpProvider<API extends Web3APISpec = EthExecutionAPI> extends Web3BaseProvider<API> {
    private readonly clientUrl;
    private readonly httpProviderOptions;
    constructor(clientUrl: string, httpProviderOptions?: HttpProviderOptions);
    private static validateClientUrl;
    getStatus(): Web3ProviderStatus;
    supportsSubscriptions(): boolean;
    request<Method extends Web3APIMethod<API>, ResultType = Web3APIReturnType<API, Method>>(payload: Web3APIPayload<API, Method>, requestOptions?: RequestInit): Promise<JsonRpcResponseWithResult<ResultType>>;
    on(): void;
    removeListener(): void;
    once(): void;
    removeAllListeners(): void;
    connect(): void;
    disconnect(): void;
    reset(): void;
    reconnect(): void;
}
export { HttpProvider };
