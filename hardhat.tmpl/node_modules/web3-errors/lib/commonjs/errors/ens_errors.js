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
exports.ENSNetworkNotSyncedError = exports.ENSUnsupportedNetworkError = exports.ENSCheckInterfaceSupportError = void 0;
/* eslint-disable max-classes-per-file */
const error_codes_js_1 = require("../error_codes.js");
const web3_error_base_js_1 = require("../web3_error_base.js");
class ENSCheckInterfaceSupportError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(errorDetails) {
        super(`ENS resolver check interface support error. "${errorDetails}"`);
        this.code = error_codes_js_1.ERR_ENS_CHECK_INTERFACE_SUPPORT;
    }
}
exports.ENSCheckInterfaceSupportError = ENSCheckInterfaceSupportError;
class ENSUnsupportedNetworkError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(networkType) {
        super(`ENS is not supported on network ${networkType}`);
        this.code = error_codes_js_1.ERR_ENS_UNSUPPORTED_NETWORK;
    }
}
exports.ENSUnsupportedNetworkError = ENSUnsupportedNetworkError;
class ENSNetworkNotSyncedError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(`Network not synced`);
        this.code = error_codes_js_1.ERR_ENS_NETWORK_NOT_SYNCED;
    }
}
exports.ENSNetworkNotSyncedError = ENSNetworkNotSyncedError;
//# sourceMappingURL=ens_errors.js.map