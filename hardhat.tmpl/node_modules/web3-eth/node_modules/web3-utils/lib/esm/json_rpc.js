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
import { isNullish } from 'web3-validator';
import { rpcErrorsMap } from 'web3-errors';
import { uuidV4 } from './uuid.js';
// check if code is a valid rpc server error code
export const isResponseRpcError = (rpcError) => {
    const errorCode = rpcError.error.code;
    return rpcErrorsMap.has(errorCode) || (errorCode >= -32099 && errorCode <= -32000);
};
export const isResponseWithResult = (response) => !Array.isArray(response) &&
    !!response &&
    response.jsonrpc === '2.0' &&
    // JSON RPC consider "null" as valid response
    'result' in response &&
    isNullish(response.error) &&
    (typeof response.id === 'number' || typeof response.id === 'string');
// To avoid circular package dependency, copied to code here. If you update this please update same function in `response_errors.ts`
export const isResponseWithError = (response) => !Array.isArray(response) &&
    response.jsonrpc === '2.0' &&
    !!response &&
    isNullish(response.result) &&
    // JSON RPC consider "null" as valid response
    'error' in response &&
    (typeof response.id === 'number' || typeof response.id === 'string');
export const isResponseWithNotification = (response) => !Array.isArray(response) &&
    !!response &&
    response.jsonrpc === '2.0' &&
    !isNullish(response.params) &&
    !isNullish(response.method);
export const isSubscriptionResult = (response) => !Array.isArray(response) &&
    !!response &&
    response.jsonrpc === '2.0' &&
    'id' in response &&
    // JSON RPC consider "null" as valid response
    'result' in response;
export const validateResponse = (response) => isResponseWithResult(response) || isResponseWithError(response);
export const isValidResponse = (response) => Array.isArray(response) ? response.every(validateResponse) : validateResponse(response);
export const isBatchResponse = (response) => Array.isArray(response) && response.length > 0 && isValidResponse(response);
// internal optional variable to increment and use for the jsonrpc `id`
let requestIdSeed;
/**
 * Optionally use to make the jsonrpc `id` start from a specific number.
 * Without calling this function, the `id` will be filled with a Uuid.
 * But after this being called with a number, the `id` will be a number starting from the provided `start` variable.
 * However, if `undefined` was passed to this function, the `id` will be a Uuid again.
 * @param start - a number to start incrementing from.
 * 	Or `undefined` to use a new Uuid (this is the default behavior)
 */
export const setRequestIdStart = (start) => {
    requestIdSeed = start;
};
export const toPayload = (request) => {
    var _a, _b, _c, _d;
    if (typeof requestIdSeed !== 'undefined') {
        requestIdSeed += 1;
    }
    return {
        jsonrpc: (_a = request.jsonrpc) !== null && _a !== void 0 ? _a : '2.0',
        id: (_c = (_b = request.id) !== null && _b !== void 0 ? _b : requestIdSeed) !== null && _c !== void 0 ? _c : uuidV4(),
        method: request.method,
        params: (_d = request.params) !== null && _d !== void 0 ? _d : undefined,
    };
};
export const toBatchPayload = (requests) => requests.map(request => toPayload(request));
export const isBatchRequest = (request) => Array.isArray(request) && request.length > 0;
//# sourceMappingURL=json_rpc.js.map