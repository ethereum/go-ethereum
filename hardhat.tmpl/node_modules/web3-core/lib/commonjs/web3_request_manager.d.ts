import { EthExecutionAPI, JsonRpcBatchRequest, JsonRpcBatchResponse, SupportedProviders, Web3APIMethod, Web3APIRequest, Web3APIReturnType, Web3APISpec, Web3BaseProviderConstructor } from 'web3-types';
import { Web3EventEmitter } from './web3_event_emitter.js';
import { RequestManagerMiddleware } from './types.js';
export declare enum Web3RequestManagerEvent {
    PROVIDER_CHANGED = "PROVIDER_CHANGED",
    BEFORE_PROVIDER_CHANGE = "BEFORE_PROVIDER_CHANGE"
}
export declare class Web3RequestManager<API extends Web3APISpec = EthExecutionAPI> extends Web3EventEmitter<{
    [key in Web3RequestManagerEvent]: SupportedProviders<API> | undefined;
}> {
    private _provider?;
    private readonly useRpcCallSpecification?;
    middleware?: RequestManagerMiddleware<API>;
    constructor(provider?: SupportedProviders<API> | string, useRpcCallSpecification?: boolean, requestManagerMiddleware?: RequestManagerMiddleware<API>);
    /**
     * Will return all available providers
     */
    static get providers(): {
        HttpProvider: Web3BaseProviderConstructor;
        WebsocketProvider: Web3BaseProviderConstructor;
    };
    /**
     * Will return the current provider.
     *
     * @returns Returns the current provider
     */
    get provider(): SupportedProviders<API> | undefined;
    /**
     * Will return all available providers
     */
    get providers(): {
        HttpProvider: Web3BaseProviderConstructor;
        WebsocketProvider: Web3BaseProviderConstructor;
    };
    /**
     * Use to set provider. Provider can be a provider instance or a string.
     *
     * @param provider - The provider to set
     */
    setProvider(provider?: SupportedProviders<API> | string): boolean;
    setMiddleware(requestManagerMiddleware: RequestManagerMiddleware<API>): void;
    /**
     *
     * Will execute a request
     *
     * @param request - {@link Web3APIRequest} The request to send
     *
     * @returns The response of the request {@link ResponseType}. If there is error
     * in the response, will throw an error
     */
    send<Method extends Web3APIMethod<API>, ResponseType = Web3APIReturnType<API, Method>>(request: Web3APIRequest<API, Method>): Promise<ResponseType>;
    /**
     * Same as send, but, will execute a batch of requests
     *
     * @param request {@link JsonRpcBatchRequest} The batch request to send
     */
    sendBatch(request: JsonRpcBatchRequest): Promise<JsonRpcBatchResponse<unknown>>;
    private _sendRequest;
    private _processJsonRpcResponse;
    private static _isReverted;
    private _buildResponse;
}
