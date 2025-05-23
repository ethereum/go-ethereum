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
exports.Web3SubscriptionManager = void 0;
const web3_types_1 = require("web3-types");
const web3_errors_1 = require("web3-errors");
const web3_utils_1 = require("web3-utils");
const utils_js_1 = require("./utils.js");
const web3_request_manager_js_1 = require("./web3_request_manager.js");
class Web3SubscriptionManager {
    constructor(requestManager, registeredSubscriptions, tolerateUnlinkedSubscription = false) {
        this.requestManager = requestManager;
        this.registeredSubscriptions = registeredSubscriptions;
        this.tolerateUnlinkedSubscription = tolerateUnlinkedSubscription;
        this._subscriptions = new Map();
        this.requestManager.on(web3_request_manager_js_1.Web3RequestManagerEvent.BEFORE_PROVIDER_CHANGE, () => __awaiter(this, void 0, void 0, function* () {
            yield this.unsubscribe();
        }));
        this.requestManager.on(web3_request_manager_js_1.Web3RequestManagerEvent.PROVIDER_CHANGED, () => {
            this.clear();
            this.listenToProviderEvents();
        });
        this.listenToProviderEvents();
    }
    listenToProviderEvents() {
        const providerAsWebProvider = this.requestManager.provider;
        if (!this.requestManager.provider ||
            (typeof (providerAsWebProvider === null || providerAsWebProvider === void 0 ? void 0 : providerAsWebProvider.supportsSubscriptions) === 'function' &&
                !(providerAsWebProvider === null || providerAsWebProvider === void 0 ? void 0 : providerAsWebProvider.supportsSubscriptions()))) {
            return;
        }
        if (typeof this.requestManager.provider.on === 'function') {
            if (typeof this.requestManager.provider.request === 'function') {
                // Listen to provider messages and data
                this.requestManager.provider.on('message', 
                // eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/no-unsafe-argument
                (message) => this.messageListener(message));
            }
            else {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/no-unsafe-argument
                providerAsWebProvider.on('data', (data) => this.messageListener(data));
            }
        }
    }
    messageListener(data) {
        var _a, _b, _c;
        if (!data) {
            throw new web3_errors_1.SubscriptionError('Should not call messageListener with no data. Type was');
        }
        const subscriptionId = ((_a = data.params) === null || _a === void 0 ? void 0 : _a.subscription) ||
            ((_b = data.data) === null || _b === void 0 ? void 0 : _b.subscription) ||
            ((_c = data.id) === null || _c === void 0 ? void 0 : _c.toString(16));
        // Process if the received data is related to a subscription
        if (subscriptionId) {
            const sub = this._subscriptions.get(subscriptionId);
            sub === null || sub === void 0 ? void 0 : sub.processSubscriptionData(data);
        }
    }
    /**
     * Will create a new subscription
     *
     * @param name - The subscription you want to subscribe to
     * @param args - Optional additional parameters, depending on the subscription type
     * @param returnFormat- ({@link DataFormat} defaults to {@link DEFAULT_RETURN_FORMAT}) - Specifies how the return data from the call should be formatted.
     *
     * Will subscribe to a specific topic (note: name)
     * @returns The subscription object
     */
    subscribe(name_1, args_1) {
        return __awaiter(this, arguments, void 0, function* (name, args, returnFormat = web3_types_1.DEFAULT_RETURN_FORMAT) {
            const Klass = this.registeredSubscriptions[name];
            if (!Klass) {
                throw new web3_errors_1.SubscriptionError('Invalid subscription type');
            }
            // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
            const subscription = new Klass(args !== null && args !== void 0 ? args : undefined, {
                subscriptionManager: this,
                returnFormat,
                // eslint.disable-next-line @typescript-eslint/no-unsafe-any
            });
            yield this.addSubscription(subscription);
            return subscription;
        });
    }
    /**
     * Will returns all subscriptions.
     */
    get subscriptions() {
        return this._subscriptions;
    }
    /**
     *
     * Adds an instance of {@link Web3Subscription} and subscribes to it
     *
     * @param sub - A {@link Web3Subscription} object
     */
    addSubscription(sub) {
        return __awaiter(this, void 0, void 0, function* () {
            if (!this.requestManager.provider) {
                throw new web3_errors_1.ProviderError('Provider not available');
            }
            if (!this.supportsSubscriptions()) {
                throw new web3_errors_1.SubscriptionError('The current provider does not support subscriptions');
            }
            if (sub.id && this._subscriptions.has(sub.id)) {
                throw new web3_errors_1.SubscriptionError(`Subscription with id "${sub.id}" already exists`);
            }
            yield sub.sendSubscriptionRequest();
            if ((0, web3_utils_1.isNullish)(sub.id)) {
                throw new web3_errors_1.SubscriptionError('Subscription is not subscribed yet.');
            }
            this._subscriptions.set(sub.id, sub);
            return sub.id;
        });
    }
    /**
     * Will clear a subscription
     *
     * @param id - The subscription of type {@link Web3Subscription}  to remove
     */
    removeSubscription(sub) {
        return __awaiter(this, void 0, void 0, function* () {
            const { id } = sub;
            if ((0, web3_utils_1.isNullish)(id)) {
                throw new web3_errors_1.SubscriptionError('Subscription is not subscribed yet. Or, had already been unsubscribed but not through the Subscription Manager.');
            }
            if (!this._subscriptions.has(id) && !this.tolerateUnlinkedSubscription) {
                throw new web3_errors_1.SubscriptionError(`Subscription with id "${id.toString()}" does not exists`);
            }
            yield sub.sendUnsubscribeRequest();
            this._subscriptions.delete(id);
            return id;
        });
    }
    /**
     * Will unsubscribe all subscriptions that fulfill the condition
     *
     * @param condition - A function that access and `id` and a `subscription` and return `true` or `false`
     * @returns An array of all the un-subscribed subscriptions
     */
    unsubscribe(condition) {
        return __awaiter(this, void 0, void 0, function* () {
            const result = [];
            for (const [id, sub] of this.subscriptions.entries()) {
                if (!condition || (typeof condition === 'function' && condition({ id, sub }))) {
                    result.push(this.removeSubscription(sub));
                }
            }
            return Promise.all(result);
        });
    }
    /**
     * Clears all subscriptions
     */
    clear() {
        this._subscriptions.clear();
    }
    /**
     * Check whether the current provider supports subscriptions.
     *
     * @returns `true` or `false` depending on if the current provider supports subscriptions
     */
    supportsSubscriptions() {
        return (0, web3_utils_1.isNullish)(this.requestManager.provider)
            ? false
            : (0, utils_js_1.isSupportSubscriptions)(this.requestManager.provider);
    }
}
exports.Web3SubscriptionManager = Web3SubscriptionManager;
//# sourceMappingURL=web3_subscription_manager.js.map