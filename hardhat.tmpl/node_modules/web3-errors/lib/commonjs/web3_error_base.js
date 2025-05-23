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
exports.InvalidValueError = exports.MultipleErrors = exports.BaseWeb3Error = void 0;
const error_codes_js_1 = require("./error_codes.js");
/**
 * Base class for Web3 errors.
 */
class BaseWeb3Error extends Error {
    /**
     * @deprecated Use the `cause` property instead.
     */
    get innerError() {
        // eslint-disable-next-line no-use-before-define
        if (this.cause instanceof MultipleErrors) {
            return this.cause.errors;
        }
        return this.cause;
    }
    /**
     * @deprecated Use the `cause` property instead.
     */
    set innerError(cause) {
        if (Array.isArray(cause)) {
            // eslint-disable-next-line no-use-before-define
            this.cause = new MultipleErrors(cause);
        }
        else {
            this.cause = cause;
        }
    }
    constructor(msg, cause) {
        super(msg);
        if (Array.isArray(cause)) {
            // eslint-disable-next-line no-use-before-define
            this.cause = new MultipleErrors(cause);
        }
        else {
            this.cause = cause;
        }
        this.name = this.constructor.name;
        if (typeof Error.captureStackTrace === 'function') {
            Error.captureStackTrace(new.target.constructor);
        }
        else {
            this.stack = new Error().stack;
        }
    }
    static convertToString(value, unquotValue = false) {
        // Using "null" value intentionally for validation
        // eslint-disable-next-line no-null/no-null
        if (value === null || value === undefined)
            return 'undefined';
        const result = JSON.stringify(value, (_, v) => (typeof v === 'bigint' ? v.toString() : v));
        return unquotValue && ['bigint', 'string'].includes(typeof value)
            ? result.replace(/['\\"]+/g, '')
            : result;
    }
    toJSON() {
        return {
            name: this.name,
            code: this.code,
            message: this.message,
            cause: this.cause,
            // deprecated
            innerError: this.cause,
        };
    }
}
exports.BaseWeb3Error = BaseWeb3Error;
class MultipleErrors extends BaseWeb3Error {
    constructor(errors) {
        super(`Multiple errors occurred: [${errors.map(e => e.message).join('], [')}]`);
        this.code = error_codes_js_1.ERR_MULTIPLE_ERRORS;
        this.errors = errors;
    }
}
exports.MultipleErrors = MultipleErrors;
class InvalidValueError extends BaseWeb3Error {
    constructor(value, msg) {
        super(`Invalid value given "${BaseWeb3Error.convertToString(value, true)}". Error: ${msg}.`);
        this.name = this.constructor.name;
    }
}
exports.InvalidValueError = InvalidValueError;
//# sourceMappingURL=web3_error_base.js.map