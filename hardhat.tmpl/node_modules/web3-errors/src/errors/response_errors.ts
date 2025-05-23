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

// eslint-disable-next-line max-classes-per-file
import {
	JsonRpcError,
	JsonRpcPayload,
	JsonRpcResponse,
	JsonRpcResponseWithError,
} from 'web3-types';
import { BaseWeb3Error, MultipleErrors } from '../web3_error_base.js';
import { ERR_INVALID_RESPONSE, ERR_RESPONSE } from '../error_codes.js';

// To avoid circular package dependency, copied to code here. If you update this please update same function in `json_rpc.ts`
const isResponseWithError = <Error = unknown, Result = unknown>(
	response: JsonRpcResponse<Result, Error>,
): response is JsonRpcResponseWithError<Error> =>
	!Array.isArray(response) &&
	response.jsonrpc === '2.0' &&
	!!response &&
	// eslint-disable-next-line no-null/no-null
	(response.result === undefined || response.result === null) &&
	// JSON RPC consider "null" as valid response
	'error' in response &&
	(typeof response.id === 'number' || typeof response.id === 'string');

const buildErrorMessage = (response: JsonRpcResponse<unknown, unknown>): string =>
	isResponseWithError(response) ? response.error.message : '';

export class ResponseError<ErrorType = unknown, RequestType = unknown> extends BaseWeb3Error {
	public code = ERR_RESPONSE;
	public data?: ErrorType | ErrorType[];
	public request?: JsonRpcPayload<RequestType>;
	public statusCode?: number;

	public constructor(
		response: JsonRpcResponse<unknown, ErrorType>,
		message?: string,
		request?: JsonRpcPayload<RequestType>,
		statusCode?: number,
	) {
		super(
			message ??
				`Returned error: ${
					Array.isArray(response)
						? response.map(r => buildErrorMessage(r)).join(',')
						: buildErrorMessage(response)
				}`,
		);

		if (!message) {
			this.data = Array.isArray(response)
				? response.map(r => r.error?.data as ErrorType)
				: response?.error?.data;
		}

		this.statusCode = statusCode;
		this.request = request;
		let errorOrErrors: JsonRpcError | JsonRpcError[] | undefined;
		if (`error` in response) {
			errorOrErrors = response.error as JsonRpcError;
		} else if (response instanceof Array) {
			errorOrErrors = response.filter(r => r.error).map(r => r.error) as JsonRpcError[];
		}

		if (Array.isArray(errorOrErrors) && errorOrErrors.length > 0) {
			this.cause = new MultipleErrors(errorOrErrors as unknown as Error[]);
		} else {
			this.cause = errorOrErrors as Error | undefined;
		}
	}

	public toJSON() {
		return {
			...super.toJSON(),
			data: this.data,
			request: this.request,
			statusCode: this.statusCode,
		};
	}
}

export class InvalidResponseError<ErrorType = unknown, RequestType = unknown> extends ResponseError<
	ErrorType,
	RequestType
> {
	public constructor(
		result: JsonRpcResponse<unknown, ErrorType>,
		request?: JsonRpcPayload<RequestType>,
	) {
		super(result, undefined, request);
		this.code = ERR_INVALID_RESPONSE;
		let errorOrErrors: JsonRpcError | JsonRpcError[] | undefined;
		if (`error` in result) {
			errorOrErrors = result.error as JsonRpcError;
		} else if (result instanceof Array) {
			errorOrErrors = result.map(r => r.error) as JsonRpcError[];
		}
		if (Array.isArray(errorOrErrors)) {
			this.cause = new MultipleErrors(errorOrErrors as unknown as Error[]);
		} else {
			this.cause = errorOrErrors as Error | undefined;
		}
	}
}
