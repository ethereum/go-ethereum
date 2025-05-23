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
exports.RequestAlreadySentError = exports.PendingRequestsOnReconnectingError = exports.MaxAttemptsReachedOnReconnectingError = exports.ConnectionCloseError = exports.ConnectionNotOpenError = exports.ConnectionTimeoutError = exports.InvalidConnectionError = exports.ConnectionError = void 0;
const error_codes_js_1 = require("../error_codes.js");
const web3_error_base_js_1 = require("../web3_error_base.js");
class ConnectionError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(message, event) {
        super(message);
        this.code = error_codes_js_1.ERR_CONN;
        if (event) {
            this.errorCode = event.code;
            this.errorReason = event.reason;
        }
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { errorCode: this.errorCode, errorReason: this.errorReason });
    }
}
exports.ConnectionError = ConnectionError;
class InvalidConnectionError extends ConnectionError {
    constructor(host, event) {
        super(`CONNECTION ERROR: Couldn't connect to node ${host}.`, event);
        this.host = host;
        this.code = error_codes_js_1.ERR_CONN_INVALID;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { host: this.host });
    }
}
exports.InvalidConnectionError = InvalidConnectionError;
class ConnectionTimeoutError extends ConnectionError {
    constructor(duration) {
        super(`CONNECTION TIMEOUT: timeout of ${duration}ms achieved`);
        this.duration = duration;
        this.code = error_codes_js_1.ERR_CONN_TIMEOUT;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { duration: this.duration });
    }
}
exports.ConnectionTimeoutError = ConnectionTimeoutError;
class ConnectionNotOpenError extends ConnectionError {
    constructor(event) {
        super('Connection not open', event);
        this.code = error_codes_js_1.ERR_CONN_NOT_OPEN;
    }
}
exports.ConnectionNotOpenError = ConnectionNotOpenError;
class ConnectionCloseError extends ConnectionError {
    constructor(event) {
        var _a, _b;
        super(`CONNECTION ERROR: The connection got closed with the close code ${(_a = event === null || event === void 0 ? void 0 : event.code) !== null && _a !== void 0 ? _a : ''} and the following reason string ${(_b = event === null || event === void 0 ? void 0 : event.reason) !== null && _b !== void 0 ? _b : ''}`, event);
        this.code = error_codes_js_1.ERR_CONN_CLOSE;
    }
}
exports.ConnectionCloseError = ConnectionCloseError;
class MaxAttemptsReachedOnReconnectingError extends ConnectionError {
    constructor(numberOfAttempts) {
        super(`Maximum number of reconnect attempts reached! (${numberOfAttempts})`);
        this.code = error_codes_js_1.ERR_CONN_MAX_ATTEMPTS;
    }
}
exports.MaxAttemptsReachedOnReconnectingError = MaxAttemptsReachedOnReconnectingError;
class PendingRequestsOnReconnectingError extends ConnectionError {
    constructor() {
        super('CONNECTION ERROR: Provider started to reconnect before the response got received!');
        this.code = error_codes_js_1.ERR_CONN_PENDING_REQUESTS;
    }
}
exports.PendingRequestsOnReconnectingError = PendingRequestsOnReconnectingError;
class RequestAlreadySentError extends ConnectionError {
    constructor(id) {
        super(`Request already sent with following id: ${id}`);
        this.code = error_codes_js_1.ERR_REQ_ALREADY_SENT;
    }
}
exports.RequestAlreadySentError = RequestAlreadySentError;
//# sourceMappingURL=connection_errors.js.map