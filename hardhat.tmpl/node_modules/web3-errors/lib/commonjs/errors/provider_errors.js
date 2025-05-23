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
exports.Web3WSProviderError = exports.SubscriptionError = exports.InvalidClientError = exports.InvalidProviderError = exports.ProviderError = void 0;
/* eslint-disable max-classes-per-file */
const error_codes_js_1 = require("../error_codes.js");
const web3_error_base_js_1 = require("../web3_error_base.js");
class ProviderError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(...arguments);
        this.code = error_codes_js_1.ERR_PROVIDER;
    }
}
exports.ProviderError = ProviderError;
class InvalidProviderError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(clientUrl) {
        super(`Provider with url "${clientUrl}" is not set or invalid`);
        this.clientUrl = clientUrl;
        this.code = error_codes_js_1.ERR_INVALID_PROVIDER;
    }
}
exports.InvalidProviderError = InvalidProviderError;
class InvalidClientError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(clientUrl) {
        super(`Client URL "${clientUrl}" is invalid.`);
        this.code = error_codes_js_1.ERR_INVALID_CLIENT;
    }
}
exports.InvalidClientError = InvalidClientError;
class SubscriptionError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(...arguments);
        this.code = error_codes_js_1.ERR_SUBSCRIPTION;
    }
}
exports.SubscriptionError = SubscriptionError;
class Web3WSProviderError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(...arguments);
        this.code = error_codes_js_1.ERR_WS_PROVIDER; // this had duplicate code with generic provider
    }
}
exports.Web3WSProviderError = Web3WSProviderError;
//# sourceMappingURL=provider_errors.js.map