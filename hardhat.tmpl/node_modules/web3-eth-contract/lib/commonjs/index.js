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
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __exportStar = (this && this.__exportStar) || function(m, exports) {
    for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports, p)) __createBinding(exports, m, p);
};
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * The `web3.eth.Contract` object makes it easy to interact with smart contracts on the Ethereum blockchain.
 * When you create a new contract object you give it the JSON interface of the respective smart contract and
 * web3 will auto convert all calls into low level ABI calls over RPC for you.
 * This allows you to interact with smart contracts as if they were JavaScript objects.
 *
 * To use it standalone:
 *
 * ```ts
 * const Contract = require('web3-eth-contract');
 *
 * // set provider for all later instances to use
 * Contract.setProvider('ws://localhost:8546');
 *
 * const contract = new Contract(jsonInterface, address);
 *
 * contract.methods.somFunc().send({from: ....})
 * .on('receipt', function(){
 *    ...
 * });
 * ```
 */
/**
 * This comment _supports3_ [Markdown](https://marked.js.org/)
 */
const contract_js_1 = require("./contract.js");
__exportStar(require("./encoding.js"), exports);
__exportStar(require("./contract.js"), exports);
__exportStar(require("./contract-deployer-method-class.js"), exports);
__exportStar(require("./contract_log_subscription.js"), exports);
__exportStar(require("./types.js"), exports);
__exportStar(require("./utils.js"), exports);
exports.default = contract_js_1.Contract;
//# sourceMappingURL=index.js.map