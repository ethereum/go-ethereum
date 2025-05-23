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
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Web3ExternalProvider = void 0;
const web3_providers_http_1 = __importDefault(require("web3-providers-http"));
const web3_providers_ws_1 = __importDefault(require("web3-providers-ws"));
const web3_utils_1 = require("web3-utils");
const types_js_1 = require("./types.js");
const errors_js_1 = require("./errors.js");
/*
This class can be used to create new providers only when there is custom logic required in each Request method like
checking specific HTTP status codes and performing any action, throwing new error types or setting additional HTTP headers in requests, or even modifying requests.

Another simpler approach can be a function simply returning URL strings instead of using the following class in case if
no additional logic implementation is required in the provider.
*/
class Web3ExternalProvider extends web3_utils_1.Eip1193Provider {
    constructor(network, transport, token, host, providerConfigOptions) {
        super();
        if (providerConfigOptions !== undefined &&
            transport === types_js_1.Transport.HTTPS &&
            !('providerOptions' in providerConfigOptions)) {
            throw new errors_js_1.ProviderConfigOptionsError('HTTP Provider');
        }
        else if (providerConfigOptions !== undefined &&
            transport === types_js_1.Transport.WebSocket &&
            !('socketOptions' in providerConfigOptions ||
                'reconnectOptions' in providerConfigOptions)) {
            throw new errors_js_1.ProviderConfigOptionsError('Websocket Provider');
        }
        this.transport = transport;
        if (transport === types_js_1.Transport.HTTPS) {
            this.provider = new web3_providers_http_1.default(this.getRPCURL(network, transport, token, host), providerConfigOptions);
        }
        else if (transport === types_js_1.Transport.WebSocket) {
            this.provider = new web3_providers_ws_1.default(this.getRPCURL(network, transport, token, host), providerConfigOptions === null || providerConfigOptions === void 0 ? void 0 : providerConfigOptions.socketOptions, providerConfigOptions === null || providerConfigOptions === void 0 ? void 0 : providerConfigOptions.reconnectOptions);
        }
    }
    request(payload, requestOptions) {
        return __awaiter(this, void 0, void 0, function* () {
            if (this.transport === types_js_1.Transport.HTTPS) {
                return (yield this.provider.request(payload, requestOptions));
            }
            return this.provider.request(payload);
        });
    }
    getStatus() {
        return this.provider.getStatus();
    }
    supportsSubscriptions() {
        return this.provider.supportsSubscriptions();
    }
    once(_type, _listener) {
        var _a;
        if ((_a = this.provider) === null || _a === void 0 ? void 0 : _a.once) {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
            this.provider.once(_type, _listener);
        }
    }
    removeAllListeners(_type) {
        var _a;
        if ((_a = this.provider) === null || _a === void 0 ? void 0 : _a.removeAllListeners)
            this.provider.removeAllListeners(_type);
    }
    connect() {
        var _a;
        if ((_a = this.provider) === null || _a === void 0 ? void 0 : _a.connect)
            this.provider.connect();
    }
    disconnect(_code, _data) {
        var _a;
        if ((_a = this.provider) === null || _a === void 0 ? void 0 : _a.disconnect)
            this.provider.disconnect(_code, _data);
    }
    reset() {
        var _a;
        if ((_a = this.provider) === null || _a === void 0 ? void 0 : _a.reset)
            this.provider.reset();
    }
    on(_type, _listener) {
        if (this.provider)
            // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
            this.provider.on(_type, _listener);
    }
    removeListener(_type, _listener) {
        if (this.provider)
            this.provider.removeListener(_type, _listener);
    }
}
exports.Web3ExternalProvider = Web3ExternalProvider;
//# sourceMappingURL=web3_provider.js.map