"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Eip1193Provider = void 0;
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
const web3_types_1 = require("web3-types");
const eventemitter3_1 = require("eventemitter3");
const web3_errors_1 = require("web3-errors");
const json_rpc_js_1 = require("./json_rpc.js");
/**
 * This is an abstract class, which extends {@link Web3BaseProvider} class. This class is used to implement a provider that adheres to the EIP-1193 standard for Ethereum providers.
 */
class Eip1193Provider extends web3_types_1.Web3BaseProvider {
    constructor() {
        super(...arguments);
        this._eventEmitter = new eventemitter3_1.EventEmitter();
        this._chainId = '';
        this._accounts = [];
    }
    _getChainId() {
        return __awaiter(this, void 0, void 0, function* () {
            var _a;
            const data = yield this.request((0, json_rpc_js_1.toPayload)({
                method: 'eth_chainId',
                params: [],
            }));
            return (_a = data === null || data === void 0 ? void 0 : data.result) !== null && _a !== void 0 ? _a : '';
        });
    }
    _getAccounts() {
        return __awaiter(this, void 0, void 0, function* () {
            var _a;
            const data = yield this.request((0, json_rpc_js_1.toPayload)({
                method: 'eth_accounts',
                params: [],
            }));
            return (_a = data === null || data === void 0 ? void 0 : data.result) !== null && _a !== void 0 ? _a : [];
        });
    }
    _onConnect() {
        Promise.all([
            this._getChainId()
                .then(chainId => {
                if (chainId !== this._chainId) {
                    this._chainId = chainId;
                    this._eventEmitter.emit('chainChanged', this._chainId);
                }
            })
                .catch(err => {
                // todo: add error handler
                console.error(err);
            }),
            this._getAccounts()
                .then(accounts => {
                if (!(this._accounts.length === accounts.length &&
                    accounts.every(v => accounts.includes(v)))) {
                    this._accounts = accounts;
                    this._onAccountsChanged();
                }
            })
                .catch(err => {
                // todo: add error handler
                // eslint-disable-next-line no-console
                console.error(err);
            }),
        ])
            .then(() => this._eventEmitter.emit('connect', {
            chainId: this._chainId,
        }))
            .catch(err => {
            // todo: add error handler
            // eslint-disable-next-line no-console
            console.error(err);
        });
    }
    // todo this must be ProvideRpcError with a message too
    _onDisconnect(code, data) {
        this._eventEmitter.emit('disconnect', new web3_errors_1.EIP1193ProviderRpcError(code, data));
    }
    _onAccountsChanged() {
        // get chainId and safe to local
        this._eventEmitter.emit('accountsChanged', this._accounts);
    }
}
exports.Eip1193Provider = Eip1193Provider;
//# sourceMappingURL=web3_eip1193_provider.js.map