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

import { JsonRpcBatchResponse, JsonRpcOptionalRequest, JsonRpcRequest } from 'web3-types';
import { jsonRpc, Web3DeferredPromise } from 'web3-utils';
import { OperationAbortError, OperationTimeoutError, ResponseError } from 'web3-errors';
import { Web3RequestManager } from './web3_request_manager.js';

export const DEFAULT_BATCH_REQUEST_TIMEOUT = 1000;

export class Web3BatchRequest {
	private readonly _requestManager: Web3RequestManager;
	private readonly _requests: Map<
		number,
		{ payload: JsonRpcRequest; promise: Web3DeferredPromise<unknown> }
	>;

	public constructor(requestManager: Web3RequestManager) {
		this._requestManager = requestManager;
		this._requests = new Map();
	}

	public get requests() {
		return [...this._requests.values()].map(r => r.payload);
	}

	public add<ResponseType = unknown>(request: JsonRpcOptionalRequest<unknown>) {
		const payload = jsonRpc.toPayload(request) as JsonRpcRequest;
		const promise = new Web3DeferredPromise<ResponseType>();

		this._requests.set(payload.id as number, { payload, promise });

		return promise;
	}

	// eslint-disable-next-line class-methods-use-this
	public async execute(options?: {
		timeout?: number;
	}): Promise<JsonRpcBatchResponse<unknown, unknown>> {
		if (this.requests.length === 0) {
			return Promise.resolve([]);
		}

		const request = new Web3DeferredPromise<JsonRpcBatchResponse<unknown, unknown>>({
			timeout: options?.timeout ?? DEFAULT_BATCH_REQUEST_TIMEOUT,
			eagerStart: true,
			timeoutMessage: 'Batch request timeout',
		});

		this._processBatchRequest(request).catch(err => request.reject(err));

		request.catch((err: Error) => {
			if (err instanceof OperationTimeoutError) {
				this._abortAllRequests('Batch request timeout');
			}

			request.reject(err);
		});

		return request;
	}

	private async _processBatchRequest(
		promise: Web3DeferredPromise<JsonRpcBatchResponse<unknown, unknown>>,
	) {
		const response = await this._requestManager.sendBatch(
			[...this._requests.values()].map(r => r.payload),
		);

		if (response.length !== this._requests.size) {
			this._abortAllRequests('Invalid batch response');

			throw new ResponseError(
				response,
				`Batch request size mismatch the results size. Requests: ${this._requests.size}, Responses: ${response.length}`,
			);
		}

		const requestIds = this.requests
			.map(r => r.id)
			.map(Number)
			.sort((a, b) => a - b);

		const responseIds = response
			.map(r => r.id)
			.map(Number)
			.sort((a, b) => a - b);

		if (JSON.stringify(requestIds) !== JSON.stringify(responseIds)) {
			this._abortAllRequests('Invalid batch response');

			throw new ResponseError(
				response,
				`Batch request mismatch the results. Requests: [${requestIds.join()}], Responses: [${responseIds.join()}]`,
			);
		}

		for (const res of response) {
			if (jsonRpc.isResponseWithResult(res)) {
				this._requests.get(res.id as number)?.promise.resolve(res.result);
			} else if (jsonRpc.isResponseWithError(res)) {
				this._requests.get(res.id as number)?.promise.reject(res.error);
			}
		}

		promise.resolve(response);
	}

	private _abortAllRequests(msg: string) {
		for (const { promise } of this._requests.values()) {
			promise.reject(new OperationAbortError(msg));
		}
	}
}
