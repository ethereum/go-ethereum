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
Object.defineProperty(exports, "__esModule", { value: true });
exports.isBatchRequest = exports.toBatchPayload = exports.toPayload = exports.setRequestIdStart = exports.isBatchResponse = exports.isValidResponse = exports.validateResponse = exports.isSubscriptionResult = exports.isResponseWithNotification = exports.isResponseWithError = exports.isResponseWithResult = exports.isResponseRpcError = void 0;
const web3_validator_1 = require("web3-validator");
const web3_errors_1 = require("web3-errors");
const uuid_js_1 = require("./uuid.js");
// check if code is a valid rpc server error code
const isResponseRpcError = (rpcError) => {
    const errorCode = rpcError.error.code;
    return web3_errors_1.rpcErrorsMap.has(errorCode) || (errorCode >= -32099 && errorCode <= -32000);
};
exports.isResponseRpcError = isResponseRpcError;
const isResponseWithResult = (response) => !Array.isArray(response) &&
    !!response &&
    response.jsonrpc === '2.0' &&
    // JSON RPC consider "null" as valid response
    'result' in response &&
    (0, web3_validator_1.isNullish)(response.error) &&
    (typeof response.id === 'number' || typeof response.id === 'string');
exports.isResponseWithResult = isResponseWithResult;
// To avoid circular package dependency, copied to code here. If you update this please update same function in `response_errors.ts`
const isResponseWithError = (response) => !Array.isArray(response) &&
    response.jsonrpc === '2.0' &&
    !!response &&
    (0, web3_validator_1.isNullish)(response.result) &&
    // JSON RPC consider "null" as valid response
    'error' in response &&
    (typeof response.id === 'number' || typeof response.id === 'string');
exports.isResponseWithError = isResponseWithError;
const isResponseWithNotification = (response) => !Array.isArray(response) &&
    !!response &&
    response.jsonrpc === '2.0' &&
    !(0, web3_validator_1.isNullish)(response.params) &&
    !(0, web3_validator_1.isNullish)(response.method);
exports.isResponseWithNotification = isResponseWithNotification;
const isSubscriptionResult = (response) => !Array.isArray(response) &&
    !!response &&
    response.jsonrpc === '2.0' &&
    'id' in response &&
    // JSON RPC consider "null" as valid response
    'result' in response;
exports.isSubscriptionResult = isSubscriptionResult;
const validateResponse = (response) => (0, exports.isResponseWithResult)(response) || (0, exports.isResponseWithError)(response);
exports.validateResponse = validateResponse;
const isValidResponse = (response) => Array.isArray(response) ? response.every(exports.validateResponse) : (0, exports.validateResponse)(response);
exports.isValidResponse = isValidResponse;
const isBatchResponse = (response) => Array.isArray(response) && response.length > 0 && (0, exports.isValidResponse)(response);
exports.isBatchResponse = isBatchResponse;
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
const setRequestIdStart = (start) => {
    requestIdSeed = start;
};
exports.setRequestIdStart = setRequestIdStart;
const toPayload = (request) => {
    var _a, _b, _c, _d;
    if (typeof requestIdSeed !== 'undefined') {
        requestIdSeed += 1;
    }
    return {
        jsonrpc: (_a = request.jsonrpc) !== null && _a !== void 0 ? _a : '2.0',
        id: (_c = (_b = request.id) !== null && _b !== void 0 ? _b : requestIdSeed) !== null && _c !== void 0 ? _c : (0, uuid_js_1.uuidV4)(),
        method: request.method,
        params: (_d = request.params) !== null && _d !== void 0 ? _d : undefined,
    };
};
exports.toPayload = toPayload;
const toBatchPayload = (requests) => requests.map(request => (0, exports.toPayload)(request));
exports.toBatchPayload = toBatchPayload;
const isBatchRequest = (request) => Array.isArray(request) && request.length > 0;
exports.isBatchRequest = isBatchRequest;
//# sourceMappingURL=json_rpc.js.map