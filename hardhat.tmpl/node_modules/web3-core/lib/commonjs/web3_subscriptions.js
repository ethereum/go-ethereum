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
Object.defineProperty(exports, "__esModule", { value: true });
exports.Web3Subscription = void 0;
// eslint-disable-next-line max-classes-per-file
const web3_types_1 = require("web3-types");
const web3_utils_1 = require("web3-utils");
// eslint-disable-next-line import/no-cycle
const web3_subscription_manager_js_1 = require("./web3_subscription_manager.js");
const web3_event_emitter_js_1 = require("./web3_event_emitter.js");
class Web3Subscription extends web3_event_emitter_js_1.Web3EventEmitter {
    constructor(args, options) {
        var _a;
        super();
        this.args = args;
        const { requestManager } = options;
        const { subscriptionManager } = options;
        if (requestManager) {
            // eslint-disable-next-line deprecation/deprecation
            this._subscriptionManager = new web3_subscription_manager_js_1.Web3SubscriptionManager(requestManager, {}, true);
        }
        else {
            this._subscriptionManager = subscriptionManager;
        }
        this._returnFormat = (_a = options === null || options === void 0 ? void 0 : options.returnFormat) !== null && _a !== void 0 ? _a : web3_types_1.DEFAULT_RETURN_FORMAT;
    }
    get id() {
        return this._id;
    }
    get lastBlock() {
        return this._lastBlock;
    }
    subscribe() {
        return __awaiter(this, void 0, void 0, function* () {
            return this._subscriptionManager.addSubscription(this);
        });
    }
    processSubscriptionData(data) {
        var _a, _b;
        if (data === null || data === void 0 ? void 0 : data.data) {
            // for EIP-1193 provider
            this._processSubscriptionResult((_b = (_a = data === null || data === void 0 ? void 0 : data.data) === null || _a === void 0 ? void 0 : _a.result) !== null && _b !== void 0 ? _b : data === null || data === void 0 ? void 0 : data.data);
        }
        else if (data &&
            web3_utils_1.jsonRpc.isResponseWithNotification(data)) {
            this._processSubscriptionResult(data === null || data === void 0 ? void 0 : data.params.result);
        }
    }
    sendSubscriptionRequest() {
        return __awaiter(this, void 0, void 0, function* () {
            this._id = yield this._subscriptionManager.requestManager.send({
                method: 'eth_subscribe',
                params: this._buildSubscriptionParams(),
            });
            this.emit('connected', this._id);
            return this._id;
        });
    }
    get returnFormat() {
        return this._returnFormat;
    }
    get subscriptionManager() {
        return this._subscriptionManager;
    }
    resubscribe() {
        return __awaiter(this, void 0, void 0, function* () {
            yield this.unsubscribe();
            yield this.subscribe();
        });
    }
    unsubscribe() {
        return __awaiter(this, void 0, void 0, function* () {
            if (!this.id) {
                return;
            }
            yield this._subscriptionManager.removeSubscription(this);
        });
    }
    sendUnsubscribeRequest() {
        return __awaiter(this, void 0, void 0, function* () {
            yield this._subscriptionManager.requestManager.send({
                method: 'eth_unsubscribe',
                params: [this.id],
            });
            this._id = undefined;
        });
    }
    // eslint-disable-next-line class-methods-use-this
    formatSubscriptionResult(data) {
        return data;
    }
    _processSubscriptionResult(data) {
        this.emit('data', this.formatSubscriptionResult(data));
    }
    _processSubscriptionError(error) {
        this.emit('error', error);
    }
    // eslint-disable-next-line class-methods-use-this
    _buildSubscriptionParams() {
        // This should be overridden in the subclass
        throw new Error('Implement in the child class');
    }
}
exports.Web3Subscription = Web3Subscription;
//# sourceMappingURL=web3_subscriptions.js.map