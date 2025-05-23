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
/* eslint-disable max-classes-per-file */
import { ERR_PROVIDER, ERR_INVALID_PROVIDER, ERR_INVALID_CLIENT, ERR_SUBSCRIPTION, ERR_WS_PROVIDER, } from '../error_codes.js';
import { BaseWeb3Error } from '../web3_error_base.js';
export class ProviderError extends BaseWeb3Error {
    constructor() {
        super(...arguments);
        this.code = ERR_PROVIDER;
    }
}
export class InvalidProviderError extends BaseWeb3Error {
    constructor(clientUrl) {
        super(`Provider with url "${clientUrl}" is not set or invalid`);
        this.clientUrl = clientUrl;
        this.code = ERR_INVALID_PROVIDER;
    }
}
export class InvalidClientError extends BaseWeb3Error {
    constructor(clientUrl) {
        super(`Client URL "${clientUrl}" is invalid.`);
        this.code = ERR_INVALID_CLIENT;
    }
}
export class SubscriptionError extends BaseWeb3Error {
    constructor() {
        super(...arguments);
        this.code = ERR_SUBSCRIPTION;
    }
}
export class Web3WSProviderError extends BaseWeb3Error {
    constructor() {
        super(...arguments);
        this.code = ERR_WS_PROVIDER; // this had duplicate code with generic provider
    }
}
//# sourceMappingURL=provider_errors.js.map