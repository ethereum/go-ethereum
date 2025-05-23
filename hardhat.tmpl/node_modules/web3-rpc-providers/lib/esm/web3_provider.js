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
import HttpProvider from 'web3-providers-http';
import WebSocketProvider from 'web3-providers-ws';
import { Eip1193Provider } from 'web3-utils';
import { Transport } from './types.js';
import { ProviderConfigOptionsError } from './errors.js';
/*
This class can be used to create new providers only when there is custom logic required in each Request method like
checking specific HTTP status codes and performing any action, throwing new error types or setting additional HTTP headers in requests, or even modifying requests.

Another simpler approach can be a function simply returning URL strings instead of using the following class in case if
no additional logic implementation is required in the provider.
*/
export class Web3ExternalProvider extends Eip1193Provider {
    constructor(network, transport, token, host, providerConfigOptions) {
        super();
        if (providerConfigOptions !== undefined &&
            transport === Transport.HTTPS &&
            !('providerOptions' in providerConfigOptions)) {
            throw new ProviderConfigOptionsError('HTTP Provider');
        }
        else if (providerConfigOptions !== undefined &&
            transport === Transport.WebSocket &&
            !('socketOptions' in providerConfigOptions ||
                'reconnectOptions' in providerConfigOptions)) {
            throw new ProviderConfigOptionsError('Websocket Provider');
        }
        this.transport = transport;
        if (transport === Transport.HTTPS) {
            this.provider = new HttpProvider(this.getRPCURL(network, transport, token, host), providerConfigOptions);
        }
        else if (transport === Transport.WebSocket) {
            this.provider = new WebSocketProvider(this.getRPCURL(network, transport, token, host), providerConfigOptions === null || providerConfigOptions === void 0 ? void 0 : providerConfigOptions.socketOptions, providerConfigOptions === null || providerConfigOptions === void 0 ? void 0 : providerConfigOptions.reconnectOptions);
        }
    }
    request(payload, requestOptions) {
        return __awaiter(this, void 0, void 0, function* () {
            if (this.transport === Transport.HTTPS) {
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
//# sourceMappingURL=web3_provider.js.map