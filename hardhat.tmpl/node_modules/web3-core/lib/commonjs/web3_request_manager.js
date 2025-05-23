"use strict";
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
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Web3RequestManager = exports.Web3RequestManagerEvent = void 0;
const web3_errors_1 = require("web3-errors");
const web3_providers_http_1 = __importDefault(require("web3-providers-http"));
const web3_providers_ws_1 = __importDefault(require("web3-providers-ws"));
const web3_utils_1 = require("web3-utils");
const utils_js_1 = require("./utils.js");
const web3_event_emitter_js_1 = require("./web3_event_emitter.js");
var Web3RequestManagerEvent;
(function (Web3RequestManagerEvent) {
    Web3RequestManagerEvent["PROVIDER_CHANGED"] = "PROVIDER_CHANGED";
    Web3RequestManagerEvent["BEFORE_PROVIDER_CHANGE"] = "BEFORE_PROVIDER_CHANGE";
})(Web3RequestManagerEvent || (exports.Web3RequestManagerEvent = Web3RequestManagerEvent = {}));
const availableProviders = {
    HttpProvider: web3_providers_http_1.default,
    WebsocketProvider: web3_providers_ws_1.default,
};
class Web3RequestManager extends web3_event_emitter_js_1.Web3EventEmitter {
    constructor(provider, useRpcCallSpecification, requestManagerMiddleware) {
        super();
        if (!(0, web3_utils_1.isNullish)(provider)) {
            this.setProvider(provider);
        }
        this.useRpcCallSpecification = useRpcCallSpecification;
        if (!(0, web3_utils_1.isNullish)(requestManagerMiddleware))
            this.middleware = requestManagerMiddleware;
    }
    /**
     * Will return all available providers
     */
    static get providers() {
        return availableProviders;
    }
    /**
     * Will return the current provider.
     *
     * @returns Returns the current provider
     */
    get provider() {
        return this._provider;
    }
    /**
     * Will return all available providers
     */
    // eslint-disable-next-line class-methods-use-this
    get providers() {
        return availableProviders;
    }
    /**
     * Use to set provider. Provider can be a provider instance or a string.
     *
     * @param provider - The provider to set
     */
    setProvider(provider) {
        let newProvider;
        // autodetect provider
        if (provider && typeof provider === 'string' && this.providers) {
            // HTTP
            if (/^http(s)?:\/\//i.test(provider)) {
                newProvider = new this.providers.HttpProvider(provider);
                // WS
            }
            else if (/^ws(s)?:\/\//i.test(provider)) {
                newProvider = new this.providers.WebsocketProvider(provider);
            }
            else {
                throw new web3_errors_1.ProviderError(`Can't autodetect provider for "${provider}"`);
            }
        }
        else if ((0, web3_utils_1.isNullish)(provider)) {
            // In case want to unset the provider
            newProvider = undefined;
        }
        else {
            newProvider = provider;
        }
        this.emit(Web3RequestManagerEvent.BEFORE_PROVIDER_CHANGE, this._provider);
        this._provider = newProvider;
        this.emit(Web3RequestManagerEvent.PROVIDER_CHANGED, this._provider);
        return true;
    }
    setMiddleware(requestManagerMiddleware) {
        this.middleware = requestManagerMiddleware;
    }
    /**
     *
     * Will execute a request
     *
     * @param request - {@link Web3APIRequest} The request to send
     *
     * @returns The response of the request {@link ResponseType}. If there is error
     * in the response, will throw an error
     */
    send(request) {
        return __awaiter(this, void 0, void 0, function* () {
            const requestObj = Object.assign({}, request);
            let response = yield this._sendRequest(requestObj);
            if (!(0, web3_utils_1.isNullish)(this.middleware))
                response = yield this.middleware.processResponse(response);
            if (web3_utils_1.jsonRpc.isResponseWithResult(response)) {
                return response.result;
            }
            throw new web3_errors_1.ResponseError(response);
        });
    }
    /**
     * Same as send, but, will execute a batch of requests
     *
     * @param request {@link JsonRpcBatchRequest} The batch request to send
     */
    sendBatch(request) {
        return __awaiter(this, void 0, void 0, function* () {
            const response = yield this._sendRequest(request);
            return response;
        });
    }
    _sendRequest(request) {
        return __awaiter(this, void 0, void 0, function* () {
            const { provider } = this;
            if ((0, web3_utils_1.isNullish)(provider)) {
                throw new web3_errors_1.ProviderError('Provider not available. Use `.setProvider` or `.provider=` to initialize the provider.');
            }
            let payload = (web3_utils_1.jsonRpc.isBatchRequest(request)
                ? web3_utils_1.jsonRpc.toBatchPayload(request)
                : web3_utils_1.jsonRpc.toPayload(request));
            if (!(0, web3_utils_1.isNullish)(this.middleware)) {
                payload = yield this.middleware.processRequest(payload);
            }
            if ((0, utils_js_1.isWeb3Provider)(provider)) {
                let response;
                try {
                    response = yield provider.request(payload);
                }
                catch (error) {
                    // Check if the provider throw an error instead of reject with error
                    response = error;
                }
                return this._processJsonRpcResponse(payload, response, { legacy: false, error: false });
            }
            if ((0, utils_js_1.isEIP1193Provider)(provider)) {
                return provider
                    .request(payload)
                    .then(res => this._processJsonRpcResponse(payload, res, {
                    legacy: true,
                    error: false,
                }))
                    .catch(error => this._processJsonRpcResponse(payload, error, { legacy: true, error: true }));
            }
            // TODO: This could be deprecated and removed.
            if ((0, utils_js_1.isLegacyRequestProvider)(provider)) {
                return new Promise((resolve, reject) => {
                    const rejectWithError = (err) => {
                        reject(this._processJsonRpcResponse(payload, err, {
                            legacy: true,
                            error: true,
                        }));
                    };
                    const resolveWithResponse = (response) => resolve(this._processJsonRpcResponse(payload, response, {
                        legacy: true,
                        error: false,
                    }));
                    const result = provider.request(payload, 
                    // a callback that is expected to be called after getting the response:
                    (err, response) => {
                        if (err) {
                            return rejectWithError(err);
                        }
                        return resolveWithResponse(response);
                    });
                    // Some providers, that follow a previous drafted version of EIP1193, has a `request` function
                    //	that is not defined as `async`, but it returns a promise.
                    // Such providers would not be picked with if(isEIP1193Provider(provider)) above
                    //	because the `request` function was not defined with `async` and so the function definition is not `AsyncFunction`.
                    // Like this provider: https://github.dev/NomicFoundation/hardhat/blob/62bea2600785595ba36f2105564076cf5cdf0fd8/packages/hardhat-core/src/internal/core/providers/backwards-compatibility.ts#L19
                    // So check if the returned result is a Promise, and resolve with it accordingly.
                    // Note: in this case we expect the callback provided above to never be called.
                    if ((0, web3_utils_1.isPromise)(result)) {
                        const responsePromise = result;
                        responsePromise.then(resolveWithResponse).catch(error => {
                            try {
                                // Attempt to process the error response
                                const processedError = this._processJsonRpcResponse(payload, error, { legacy: true, error: true });
                                reject(processedError);
                            }
                            catch (processingError) {
                                // Catch any errors that occur during the error processing
                                reject(processingError);
                            }
                        });
                    }
                });
            }
            // TODO: This could be deprecated and removed.
            if ((0, utils_js_1.isLegacySendProvider)(provider)) {
                return new Promise((resolve, reject) => {
                    provider.send(payload, (err, response) => {
                        if (err) {
                            return reject(this._processJsonRpcResponse(payload, err, {
                                legacy: true,
                                error: true,
                            }));
                        }
                        if ((0, web3_utils_1.isNullish)(response)) {
                            throw new web3_errors_1.ResponseError({}, 'Got a "nullish" response from provider.');
                        }
                        return resolve(this._processJsonRpcResponse(payload, response, {
                            legacy: true,
                            error: false,
                        }));
                    });
                });
            }
            // TODO: This could be deprecated and removed.
            if ((0, utils_js_1.isLegacySendAsyncProvider)(provider)) {
                return provider
                    .sendAsync(payload)
                    .then(response => this._processJsonRpcResponse(payload, response, { legacy: true, error: false }))
                    .catch(error => this._processJsonRpcResponse(payload, error, {
                    legacy: true,
                    error: true,
                }));
            }
            throw new web3_errors_1.ProviderError('Provider does not have a request or send method to use.');
        });
    }
    // eslint-disable-next-line class-methods-use-this
    _processJsonRpcResponse(payload, response, { legacy, error }) {
        if ((0, web3_utils_1.isNullish)(response)) {
            return this._buildResponse(payload, 
            // Some providers uses "null" as valid empty response
            // eslint-disable-next-line no-null/no-null
            null, error);
        }
        // This is the majority of the cases so check these first
        // A valid JSON-RPC response with error object
        if (web3_utils_1.jsonRpc.isResponseWithError(response)) {
            // check if its an rpc error
            if (this.useRpcCallSpecification &&
                (0, web3_utils_1.isResponseRpcError)(response)) {
                const rpcErrorResponse = response;
                // check if rpc error flag is on and response error code match an EIP-1474 or a standard rpc error code
                if (web3_errors_1.rpcErrorsMap.get(rpcErrorResponse.error.code)) {
                    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
                    const Err = web3_errors_1.rpcErrorsMap.get(rpcErrorResponse.error.code).error;
                    throw new Err(rpcErrorResponse);
                }
                else {
                    throw new web3_errors_1.RpcError(rpcErrorResponse);
                }
            }
            else if (!Web3RequestManager._isReverted(response)) {
                throw new web3_errors_1.InvalidResponseError(response, payload);
            }
        }
        // This is the majority of the cases so check these first
        // A valid JSON-RPC response with result object
        if (web3_utils_1.jsonRpc.isResponseWithResult(response)) {
            return response;
        }
        if (response instanceof Error) {
            Web3RequestManager._isReverted(response);
            throw response;
        }
        if (!legacy && web3_utils_1.jsonRpc.isBatchRequest(payload) && web3_utils_1.jsonRpc.isBatchResponse(response)) {
            return response;
        }
        if (legacy && !error && web3_utils_1.jsonRpc.isBatchRequest(payload)) {
            return response;
        }
        if (legacy && error && web3_utils_1.jsonRpc.isBatchRequest(payload)) {
            // In case of error batch response we don't want to throw Invalid response
            throw response;
        }
        if (legacy &&
            !web3_utils_1.jsonRpc.isResponseWithError(response) &&
            !web3_utils_1.jsonRpc.isResponseWithResult(response)) {
            return this._buildResponse(payload, response, error);
        }
        if (web3_utils_1.jsonRpc.isBatchRequest(payload) && !Array.isArray(response)) {
            throw new web3_errors_1.ResponseError(response, 'Got normal response for a batch request.');
        }
        if (!web3_utils_1.jsonRpc.isBatchRequest(payload) && Array.isArray(response)) {
            throw new web3_errors_1.ResponseError(response, 'Got batch response for a normal request.');
        }
        throw new web3_errors_1.ResponseError(response, 'Invalid response');
    }
    static _isReverted(response) {
        let error;
        if (web3_utils_1.jsonRpc.isResponseWithError(response)) {
            error = response.error;
        }
        else if (response instanceof Error) {
            error = response;
        }
        // This message means that there was an error while executing the code of the smart contract
        // However, more processing will happen at a higher level to decode the error data,
        //	according to the Error ABI, if it was available as of EIP-838.
        if (error === null || error === void 0 ? void 0 : error.message.includes('revert'))
            throw new web3_errors_1.ContractExecutionError(error);
        return false;
    }
    // Need to use same types as _processJsonRpcResponse so have to declare as instance method
    // eslint-disable-next-line class-methods-use-this
    _buildResponse(payload, response, error) {
        const res = {
            jsonrpc: '2.0',
            // eslint-disable-next-line no-nested-ternary
            id: web3_utils_1.jsonRpc.isBatchRequest(payload)
                ? payload[0].id
                : 'id' in payload
                    ? payload.id
                    : // Have to use the null here explicitly
                        // eslint-disable-next-line no-null/no-null
                        null,
        };
        if (error) {
            return Object.assign(Object.assign({}, res), { error: response });
        }
        return Object.assign(Object.assign({}, res), { result: response });
    }
}
exports.Web3RequestManager = Web3RequestManager;
//# sourceMappingURL=web3_request_manager.js.map