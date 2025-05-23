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
import { jsonRpc, Web3DeferredPromise } from 'web3-utils';
import { OperationAbortError, OperationTimeoutError, ResponseError } from 'web3-errors';
export const DEFAULT_BATCH_REQUEST_TIMEOUT = 1000;
export class Web3BatchRequest {
    constructor(requestManager) {
        this._requestManager = requestManager;
        this._requests = new Map();
    }
    get requests() {
        return [...this._requests.values()].map(r => r.payload);
    }
    add(request) {
        const payload = jsonRpc.toPayload(request);
        const promise = new Web3DeferredPromise();
        this._requests.set(payload.id, { payload, promise });
        return promise;
    }
    // eslint-disable-next-line class-methods-use-this
    execute(options) {
        return __awaiter(this, void 0, void 0, function* () {
            var _a;
            if (this.requests.length === 0) {
                return Promise.resolve([]);
            }
            const request = new Web3DeferredPromise({
                timeout: (_a = options === null || options === void 0 ? void 0 : options.timeout) !== null && _a !== void 0 ? _a : DEFAULT_BATCH_REQUEST_TIMEOUT,
                eagerStart: true,
                timeoutMessage: 'Batch request timeout',
            });
            this._processBatchRequest(request).catch(err => request.reject(err));
            request.catch((err) => {
                if (err instanceof OperationTimeoutError) {
                    this._abortAllRequests('Batch request timeout');
                }
                request.reject(err);
            });
            return request;
        });
    }
    _processBatchRequest(promise) {
        return __awaiter(this, void 0, void 0, function* () {
            var _a, _b;
            const response = yield this._requestManager.sendBatch([...this._requests.values()].map(r => r.payload));
            if (response.length !== this._requests.size) {
                this._abortAllRequests('Invalid batch response');
                throw new ResponseError(response, `Batch request size mismatch the results size. Requests: ${this._requests.size}, Responses: ${response.length}`);
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
                throw new ResponseError(response, `Batch request mismatch the results. Requests: [${requestIds.join()}], Responses: [${responseIds.join()}]`);
            }
            for (const res of response) {
                if (jsonRpc.isResponseWithResult(res)) {
                    (_a = this._requests.get(res.id)) === null || _a === void 0 ? void 0 : _a.promise.resolve(res.result);
                }
                else if (jsonRpc.isResponseWithError(res)) {
                    (_b = this._requests.get(res.id)) === null || _b === void 0 ? void 0 : _b.promise.reject(res.error);
                }
            }
            promise.resolve(response);
        });
    }
    _abortAllRequests(msg) {
        for (const { promise } of this._requests.values()) {
            promise.reject(new OperationAbortError(msg));
        }
    }
}
//# sourceMappingURL=web3_batch_request.js.map