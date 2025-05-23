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
exports.HttpProvider = void 0;
const cross_fetch_1 = __importDefault(require("cross-fetch"));
const web3_types_1 = require("web3-types");
const web3_errors_1 = require("web3-errors");
class HttpProvider extends web3_types_1.Web3BaseProvider {
    constructor(clientUrl, httpProviderOptions) {
        super();
        if (!HttpProvider.validateClientUrl(clientUrl))
            throw new web3_errors_1.InvalidClientError(clientUrl);
        this.clientUrl = clientUrl;
        this.httpProviderOptions = httpProviderOptions;
    }
    static validateClientUrl(clientUrl) {
        return typeof clientUrl === 'string' ? /^http(s)?:\/\//i.test(clientUrl) : false;
    }
    /* eslint-disable class-methods-use-this */
    getStatus() {
        throw new web3_errors_1.MethodNotImplementedError();
    }
    /* eslint-disable class-methods-use-this */
    supportsSubscriptions() {
        return false;
    }
    request(payload, requestOptions) {
        var _a;
        return __awaiter(this, void 0, void 0, function* () {
            const providerOptionsCombined = Object.assign(Object.assign({}, (_a = this.httpProviderOptions) === null || _a === void 0 ? void 0 : _a.providerOptions), requestOptions);
            const response = yield (0, cross_fetch_1.default)(this.clientUrl, Object.assign(Object.assign({}, providerOptionsCombined), { method: 'POST', headers: Object.assign(Object.assign({}, providerOptionsCombined.headers), { 'Content-Type': 'application/json' }), body: JSON.stringify(payload) }));
            if (!response.ok) {
                // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
                throw new web3_errors_1.ResponseError(yield response.json(), undefined, undefined, response.status);
            }
            ;
            return (yield response.json());
        });
    }
    /* eslint-disable class-methods-use-this */
    on() {
        throw new web3_errors_1.MethodNotImplementedError();
    }
    /* eslint-disable class-methods-use-this */
    removeListener() {
        throw new web3_errors_1.MethodNotImplementedError();
    }
    /* eslint-disable class-methods-use-this */
    once() {
        throw new web3_errors_1.MethodNotImplementedError();
    }
    /* eslint-disable class-methods-use-this */
    removeAllListeners() {
        throw new web3_errors_1.MethodNotImplementedError();
    }
    /* eslint-disable class-methods-use-this */
    connect() {
        throw new web3_errors_1.MethodNotImplementedError();
    }
    /* eslint-disable class-methods-use-this */
    disconnect() {
        throw new web3_errors_1.MethodNotImplementedError();
    }
    /* eslint-disable class-methods-use-this */
    reset() {
        throw new web3_errors_1.MethodNotImplementedError();
    }
    /* eslint-disable class-methods-use-this */
    reconnect() {
        throw new web3_errors_1.MethodNotImplementedError();
    }
}
exports.default = HttpProvider;
exports.HttpProvider = HttpProvider;
//# sourceMappingURL=index.js.map