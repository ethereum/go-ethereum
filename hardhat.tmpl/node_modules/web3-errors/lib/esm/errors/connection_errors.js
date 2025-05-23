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
import { ERR_CONN, ERR_CONN_INVALID, ERR_CONN_TIMEOUT, ERR_CONN_NOT_OPEN, ERR_CONN_CLOSE, ERR_CONN_MAX_ATTEMPTS, ERR_CONN_PENDING_REQUESTS, ERR_REQ_ALREADY_SENT, } from '../error_codes.js';
import { BaseWeb3Error } from '../web3_error_base.js';
export class ConnectionError extends BaseWeb3Error {
    constructor(message, event) {
        super(message);
        this.code = ERR_CONN;
        if (event) {
            this.errorCode = event.code;
            this.errorReason = event.reason;
        }
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { errorCode: this.errorCode, errorReason: this.errorReason });
    }
}
export class InvalidConnectionError extends ConnectionError {
    constructor(host, event) {
        super(`CONNECTION ERROR: Couldn't connect to node ${host}.`, event);
        this.host = host;
        this.code = ERR_CONN_INVALID;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { host: this.host });
    }
}
export class ConnectionTimeoutError extends ConnectionError {
    constructor(duration) {
        super(`CONNECTION TIMEOUT: timeout of ${duration}ms achieved`);
        this.duration = duration;
        this.code = ERR_CONN_TIMEOUT;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { duration: this.duration });
    }
}
export class ConnectionNotOpenError extends ConnectionError {
    constructor(event) {
        super('Connection not open', event);
        this.code = ERR_CONN_NOT_OPEN;
    }
}
export class ConnectionCloseError extends ConnectionError {
    constructor(event) {
        var _a, _b;
        super(`CONNECTION ERROR: The connection got closed with the close code ${(_a = event === null || event === void 0 ? void 0 : event.code) !== null && _a !== void 0 ? _a : ''} and the following reason string ${(_b = event === null || event === void 0 ? void 0 : event.reason) !== null && _b !== void 0 ? _b : ''}`, event);
        this.code = ERR_CONN_CLOSE;
    }
}
export class MaxAttemptsReachedOnReconnectingError extends ConnectionError {
    constructor(numberOfAttempts) {
        super(`Maximum number of reconnect attempts reached! (${numberOfAttempts})`);
        this.code = ERR_CONN_MAX_ATTEMPTS;
    }
}
export class PendingRequestsOnReconnectingError extends ConnectionError {
    constructor() {
        super('CONNECTION ERROR: Provider started to reconnect before the response got received!');
        this.code = ERR_CONN_PENDING_REQUESTS;
    }
}
export class RequestAlreadySentError extends ConnectionError {
    constructor(id) {
        super(`Request already sent with following id: ${id}`);
        this.code = ERR_REQ_ALREADY_SENT;
    }
}
//# sourceMappingURL=connection_errors.js.map