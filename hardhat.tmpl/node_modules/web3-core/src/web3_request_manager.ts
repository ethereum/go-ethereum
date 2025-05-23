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

import {
	ContractExecutionError,
	InvalidResponseError,
	ProviderError,
	ResponseError,
	rpcErrorsMap,
	RpcError,
} from 'web3-errors';
import HttpProvider from 'web3-providers-http';
import WSProvider from 'web3-providers-ws';
import {
	EthExecutionAPI,
	JsonRpcBatchRequest,
	JsonRpcBatchResponse,
	JsonRpcPayload,
	JsonRpcResponse,
	JsonRpcError,
	JsonRpcResponseWithResult,
	JsonRpcResponseWithError,
	SupportedProviders,
	Web3APIMethod,
	Web3APIPayload,
	Web3APIRequest,
	Web3APIReturnType,
	Web3APISpec,
	Web3BaseProvider,
	Web3BaseProviderConstructor,
} from 'web3-types';
import { isNullish, isPromise, jsonRpc, isResponseRpcError } from 'web3-utils';
import {
	isEIP1193Provider,
	isLegacyRequestProvider,
	isLegacySendAsyncProvider,
	isLegacySendProvider,
	isWeb3Provider,
} from './utils.js';
import { Web3EventEmitter } from './web3_event_emitter.js';
import { RequestManagerMiddleware } from './types.js';

export enum Web3RequestManagerEvent {
	PROVIDER_CHANGED = 'PROVIDER_CHANGED',
	BEFORE_PROVIDER_CHANGE = 'BEFORE_PROVIDER_CHANGE',
}

const availableProviders: {
	HttpProvider: Web3BaseProviderConstructor;
	WebsocketProvider: Web3BaseProviderConstructor;
} = {
	HttpProvider: HttpProvider as Web3BaseProviderConstructor,
	WebsocketProvider: WSProvider as Web3BaseProviderConstructor,
};

export class Web3RequestManager<
	API extends Web3APISpec = EthExecutionAPI,
> extends Web3EventEmitter<{
	[key in Web3RequestManagerEvent]: SupportedProviders<API> | undefined;
}> {
	private _provider?: SupportedProviders<API>;
	private readonly useRpcCallSpecification?: boolean;
	public middleware?: RequestManagerMiddleware<API>;

	public constructor(
		provider?: SupportedProviders<API> | string,
		useRpcCallSpecification?: boolean,
		requestManagerMiddleware?: RequestManagerMiddleware<API>,
	) {
		super();

		if (!isNullish(provider)) {
			this.setProvider(provider);
		}
		this.useRpcCallSpecification = useRpcCallSpecification;

		if (!isNullish(requestManagerMiddleware)) this.middleware = requestManagerMiddleware;
	}

	/**
	 * Will return all available providers
	 */
	public static get providers() {
		return availableProviders;
	}

	/**
	 * Will return the current provider.
	 *
	 * @returns Returns the current provider
	 */
	public get provider() {
		return this._provider;
	}

	/**
	 * Will return all available providers
	 */
	// eslint-disable-next-line class-methods-use-this
	public get providers() {
		return availableProviders;
	}

	/**
	 * Use to set provider. Provider can be a provider instance or a string.
	 *
	 * @param provider - The provider to set
	 */
	public setProvider(provider?: SupportedProviders<API> | string): boolean {
		let newProvider: SupportedProviders<API> | undefined;

		// autodetect provider
		if (provider && typeof provider === 'string' && this.providers) {
			// HTTP
			if (/^http(s)?:\/\//i.test(provider)) {
				newProvider = new this.providers.HttpProvider<API>(provider);

				// WS
			} else if (/^ws(s)?:\/\//i.test(provider)) {
				newProvider = new this.providers.WebsocketProvider<API>(provider);
			} else {
				throw new ProviderError(`Can't autodetect provider for "${provider}"`);
			}
		} else if (isNullish(provider)) {
			// In case want to unset the provider
			newProvider = undefined;
		} else {
			newProvider = provider as SupportedProviders<API>;
		}

		this.emit(Web3RequestManagerEvent.BEFORE_PROVIDER_CHANGE, this._provider);
		this._provider = newProvider;
		this.emit(Web3RequestManagerEvent.PROVIDER_CHANGED, this._provider);
		return true;
	}

	public setMiddleware(requestManagerMiddleware: RequestManagerMiddleware<API>) {
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
	public async send<
		Method extends Web3APIMethod<API>,
		ResponseType = Web3APIReturnType<API, Method>,
	>(request: Web3APIRequest<API, Method>): Promise<ResponseType> {
		const requestObj = { ...request };

		let response = await this._sendRequest<Method, ResponseType>(requestObj);

		if (!isNullish(this.middleware)) response = await this.middleware.processResponse(response);

		if (jsonRpc.isResponseWithResult(response)) {
			return response.result;
		}

		throw new ResponseError(response);
	}

	/**
	 * Same as send, but, will execute a batch of requests
	 *
	 * @param request {@link JsonRpcBatchRequest} The batch request to send
	 */
	public async sendBatch(request: JsonRpcBatchRequest): Promise<JsonRpcBatchResponse<unknown>> {
		const response = await this._sendRequest<never, never>(request);

		return response as JsonRpcBatchResponse<unknown>;
	}

	private async _sendRequest<
		Method extends Web3APIMethod<API>,
		ResponseType = Web3APIReturnType<API, Method>,
	>(
		request: Web3APIRequest<API, Method> | JsonRpcBatchRequest,
	): Promise<JsonRpcResponse<ResponseType>> {
		const { provider } = this;

		if (isNullish(provider)) {
			throw new ProviderError(
				'Provider not available. Use `.setProvider` or `.provider=` to initialize the provider.',
			);
		}

		let payload = (
			jsonRpc.isBatchRequest(request)
				? jsonRpc.toBatchPayload(request)
				: jsonRpc.toPayload(request)
		) as JsonRpcPayload;

		if (!isNullish(this.middleware)) {
			payload = await this.middleware.processRequest(payload);
		}
		if (isWeb3Provider(provider)) {
			let response;

			try {
				response = await provider.request<Method, ResponseType>(
					payload as Web3APIPayload<API, Method>,
				);
			} catch (error) {
				// Check if the provider throw an error instead of reject with error
				response = error as JsonRpcResponse<ResponseType>;
			}
			return this._processJsonRpcResponse(payload, response, { legacy: false, error: false });
		}

		if (isEIP1193Provider(provider)) {
			return (provider as Web3BaseProvider<API>)
				.request<Method, ResponseType>(payload as Web3APIPayload<API, Method>)
				.then(
					res =>
						this._processJsonRpcResponse(payload, res, {
							legacy: true,
							error: false,
						}) as JsonRpcResponseWithResult<ResponseType>,
				)
				.catch(error =>
					this._processJsonRpcResponse(
						payload,
						error as JsonRpcResponse<ResponseType, unknown>,
						{ legacy: true, error: true },
					),
				);
		}

		// TODO: This could be deprecated and removed.
		if (isLegacyRequestProvider(provider)) {
			return new Promise<JsonRpcResponse<ResponseType>>((resolve, reject) => {
				const rejectWithError = (err: unknown) => {
					reject(
						this._processJsonRpcResponse(
							payload,
							err as JsonRpcResponse<ResponseType>,
							{
								legacy: true,
								error: true,
							},
						),
					);
				};

				const resolveWithResponse = (response: JsonRpcResponse<ResponseType>) =>
					resolve(
						this._processJsonRpcResponse(payload, response, {
							legacy: true,
							error: false,
						}),
					);
				const result = provider.request<ResponseType>(
					payload,
					// a callback that is expected to be called after getting the response:
					(err, response) => {
						if (err) {
							return rejectWithError(err);
						}

						return resolveWithResponse(response);
					},
				);
				// Some providers, that follow a previous drafted version of EIP1193, has a `request` function
				//	that is not defined as `async`, but it returns a promise.
				// Such providers would not be picked with if(isEIP1193Provider(provider)) above
				//	because the `request` function was not defined with `async` and so the function definition is not `AsyncFunction`.
				// Like this provider: https://github.dev/NomicFoundation/hardhat/blob/62bea2600785595ba36f2105564076cf5cdf0fd8/packages/hardhat-core/src/internal/core/providers/backwards-compatibility.ts#L19
				// So check if the returned result is a Promise, and resolve with it accordingly.
				// Note: in this case we expect the callback provided above to never be called.
				if (isPromise(result)) {
					const responsePromise = result as unknown as Promise<
						JsonRpcResponse<ResponseType>
					>;
					responsePromise.then(resolveWithResponse).catch(error => {
						try {
							// Attempt to process the error response
							const processedError = this._processJsonRpcResponse(
								payload,
								error as JsonRpcResponse<ResponseType, unknown>,
								{ legacy: true, error: true },
							);
							reject(processedError);
						} catch (processingError) {
							// Catch any errors that occur during the error processing
							reject(processingError);
						}
					});
				}
			});
		}

		// TODO: This could be deprecated and removed.
		if (isLegacySendProvider(provider)) {
			return new Promise<JsonRpcResponse<ResponseType>>((resolve, reject): void => {
				provider.send<ResponseType>(payload, (err, response) => {
					if (err) {
						return reject(
							this._processJsonRpcResponse(
								payload,
								err as unknown as JsonRpcResponse<ResponseType>,
								{
									legacy: true,
									error: true,
								},
							),
						);
					}

					if (isNullish(response)) {
						throw new ResponseError(
							{} as never,
							'Got a "nullish" response from provider.',
						);
					}

					return resolve(
						this._processJsonRpcResponse(payload, response, {
							legacy: true,
							error: false,
						}),
					);
				});
			});
		}

		// TODO: This could be deprecated and removed.
		if (isLegacySendAsyncProvider(provider)) {
			return provider
				.sendAsync<ResponseType>(payload)
				.then(response =>
					this._processJsonRpcResponse(payload, response, { legacy: true, error: false }),
				)
				.catch(error =>
					this._processJsonRpcResponse(payload, error as JsonRpcResponse<ResponseType>, {
						legacy: true,
						error: true,
					}),
				);
		}

		throw new ProviderError('Provider does not have a request or send method to use.');
	}

	// eslint-disable-next-line class-methods-use-this
	private _processJsonRpcResponse<ResultType, ErrorType, RequestType>(
		payload: JsonRpcPayload<RequestType>,
		response: JsonRpcResponse<ResultType, ErrorType>,
		{ legacy, error }: { legacy: boolean; error: boolean },
	): JsonRpcResponse<ResultType> | never {
		if (isNullish(response)) {
			return this._buildResponse(
				payload,
				// Some providers uses "null" as valid empty response
				// eslint-disable-next-line no-null/no-null
				null as unknown as JsonRpcResponse<ResultType, ErrorType>,
				error,
			);
		}

		// This is the majority of the cases so check these first
		// A valid JSON-RPC response with error object
		if (jsonRpc.isResponseWithError<ErrorType>(response)) {
			// check if its an rpc error
			if (
				this.useRpcCallSpecification &&
				isResponseRpcError(response as JsonRpcResponseWithError)
			) {
				const rpcErrorResponse = response as JsonRpcResponseWithError;
				// check if rpc error flag is on and response error code match an EIP-1474 or a standard rpc error code
				if (rpcErrorsMap.get(rpcErrorResponse.error.code)) {
					// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
					const Err = rpcErrorsMap.get(rpcErrorResponse.error.code)!.error;
					throw new Err(rpcErrorResponse);
				} else {
					throw new RpcError(rpcErrorResponse);
				}
			} else if (!Web3RequestManager._isReverted(response)) {
				throw new InvalidResponseError<ErrorType, RequestType>(response, payload);
			}
		}

		// This is the majority of the cases so check these first
		// A valid JSON-RPC response with result object
		if (jsonRpc.isResponseWithResult<ResultType>(response)) {
			return response;
		}

		if ((response as unknown) instanceof Error) {
			Web3RequestManager._isReverted(response);
			throw response;
		}

		if (!legacy && jsonRpc.isBatchRequest(payload) && jsonRpc.isBatchResponse(response)) {
			return response as JsonRpcBatchResponse<ResultType>;
		}

		if (legacy && !error && jsonRpc.isBatchRequest(payload)) {
			return response as JsonRpcBatchResponse<ResultType>;
		}

		if (legacy && error && jsonRpc.isBatchRequest(payload)) {
			// In case of error batch response we don't want to throw Invalid response
			throw response;
		}

		if (
			legacy &&
			!jsonRpc.isResponseWithError(response) &&
			!jsonRpc.isResponseWithResult(response)
		) {
			return this._buildResponse(payload, response, error);
		}
		if (jsonRpc.isBatchRequest(payload) && !Array.isArray(response)) {
			throw new ResponseError(response, 'Got normal response for a batch request.');
		}

		if (!jsonRpc.isBatchRequest(payload) && Array.isArray(response)) {
			throw new ResponseError(response, 'Got batch response for a normal request.');
		}

		throw new ResponseError(response, 'Invalid response');
	}

	private static _isReverted<ResultType, ErrorType>(
		response: JsonRpcResponse<ResultType, ErrorType>,
	): boolean {
		let error: JsonRpcError | undefined;

		if (jsonRpc.isResponseWithError<ErrorType>(response)) {
			error = (response as JsonRpcResponseWithError).error;
		} else if ((response as unknown) instanceof Error) {
			error = response as unknown as JsonRpcError;
		}

		// This message means that there was an error while executing the code of the smart contract
		// However, more processing will happen at a higher level to decode the error data,
		//	according to the Error ABI, if it was available as of EIP-838.
		if (error?.message.includes('revert')) throw new ContractExecutionError(error);

		return false;
	}
	// Need to use same types as _processJsonRpcResponse so have to declare as instance method
	// eslint-disable-next-line class-methods-use-this
	private _buildResponse<ResultType, ErrorType, RequestType>(
		payload: JsonRpcPayload<RequestType>,
		response: JsonRpcResponse<ResultType, ErrorType>,
		error: boolean,
	): JsonRpcResponse<ResultType> {
		const res = {
			jsonrpc: '2.0',
			// eslint-disable-next-line no-nested-ternary
			id: jsonRpc.isBatchRequest(payload)
				? payload[0].id
				: 'id' in payload
				? payload.id
				: // Have to use the null here explicitly
				  // eslint-disable-next-line no-null/no-null
				  null,
		};

		if (error) {
			return {
				...res,
				error: response as unknown,
			} as JsonRpcResponse<ResultType>;
		}

		return {
			...res,
			result: response as unknown,
		} as JsonRpcResponse<ResultType>;
	}
}
