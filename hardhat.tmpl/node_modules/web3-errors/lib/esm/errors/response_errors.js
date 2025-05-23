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
import { BaseWeb3Error, MultipleErrors } from '../web3_error_base.js';
import { ERR_INVALID_RESPONSE, ERR_RESPONSE } from '../error_codes.js';
// To avoid circular package dependency, copied to code here. If you update this please update same function in `json_rpc.ts`
const isResponseWithError = (response) => !Array.isArray(response) &&
    response.jsonrpc === '2.0' &&
    !!response &&
    // eslint-disable-next-line no-null/no-null
    (response.result === undefined || response.result === null) &&
    // JSON RPC consider "null" as valid response
    'error' in response &&
    (typeof response.id === 'number' || typeof response.id === 'string');
const buildErrorMessage = (response) => isResponseWithError(response) ? response.error.message : '';
export class ResponseError extends BaseWeb3Error {
    constructor(response, message, request, statusCode) {
        var _a;
        super(message !== null && message !== void 0 ? message : `Returned error: ${Array.isArray(response)
            ? response.map(r => buildErrorMessage(r)).join(',')
            : buildErrorMessage(response)}`);
        this.code = ERR_RESPONSE;
        if (!message) {
            this.data = Array.isArray(response)
                ? response.map(r => { var _a; return (_a = r.error) === null || _a === void 0 ? void 0 : _a.data; })
                : (_a = response === null || response === void 0 ? void 0 : response.error) === null || _a === void 0 ? void 0 : _a.data;
        }
        this.statusCode = statusCode;
        this.request = request;
        let errorOrErrors;
        if (`error` in response) {
            errorOrErrors = response.error;
        }
        else if (response instanceof Array) {
            errorOrErrors = response.filter(r => r.error).map(r => r.error);
        }
        if (Array.isArray(errorOrErrors) && errorOrErrors.length > 0) {
            this.cause = new MultipleErrors(errorOrErrors);
        }
        else {
            this.cause = errorOrErrors;
        }
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { data: this.data, request: this.request, statusCode: this.statusCode });
    }
}
export class InvalidResponseError extends ResponseError {
    constructor(result, request) {
        super(result, undefined, request);
        this.code = ERR_INVALID_RESPONSE;
        let errorOrErrors;
        if (`error` in result) {
            errorOrErrors = result.error;
        }
        else if (result instanceof Array) {
            errorOrErrors = result.map(r => r.error);
        }
        if (Array.isArray(errorOrErrors)) {
            this.cause = new MultipleErrors(errorOrErrors);
        }
        else {
            this.cause = errorOrErrors;
        }
    }
}
//# sourceMappingURL=response_errors.js.map