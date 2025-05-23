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
exports.ContractSubscriptionManager = void 0;
const web3_core_1 = require("web3-core");
const web3_types_1 = require("web3-types");
/**
 * Similar to `Web3SubscriptionManager` but has a reference to the Contract that uses
 */
class ContractSubscriptionManager extends web3_core_1.Web3SubscriptionManager {
    /**
     *
     * @param - Web3SubscriptionManager
     * @param - parentContract
     *
     * @example
     * ```ts
     * const requestManager = new Web3RequestManager("ws://localhost:8545");
     * const contract = new Contract(...)
     * const subscriptionManager = new Web3SubscriptionManager(requestManager, {}, contract);
     * ```
     */
    constructor(self, parentContract) {
        super(self.requestManager, self.registeredSubscriptions);
        this.parentContract = parentContract;
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
        const _super = Object.create(null, {
            subscribe: { get: () => super.subscribe }
        });
        return __awaiter(this, arguments, void 0, function* (name, args, returnFormat = web3_types_1.DEFAULT_RETURN_FORMAT) {
            return _super.subscribe.call(this, name, args !== null && args !== void 0 ? args : this.parentContract.options, returnFormat);
        });
    }
}
exports.ContractSubscriptionManager = ContractSubscriptionManager;
//# sourceMappingURL=contract-subscription-manager.js.map