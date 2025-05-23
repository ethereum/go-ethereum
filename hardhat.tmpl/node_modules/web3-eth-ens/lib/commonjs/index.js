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
exports.registryAddresses = void 0;
/**
 * The `web3.eth.ens` functions let you interact with ENS. We recommend reading the [ENS documentation](https://docs.ens.domains/) to get deeper insights about the internals of the name service.
 *
 * ## Breaking Changes
 *
 * -   All the API level interfaces returning or accepting `null` in 1.x, use `undefined` in 4.x.
 * -   Functions don't accept a callback anymore.
 * -   Functions that accepted an optional `TransactionConfig` as the last argument, now accept an optional `NonPayableCallOptions`. See `web3-eth-contract` package for more details.
 * -   Removed all non-read methods. If you need modifing resolver or registry, please use https://www.npmjs.com/package/@ensdomains/ensjs
 */
/**
 * This comment _supports3_ [Markdown](https://marked.js.org/)
 */
const config_js_1 = require("./config.js");
Object.defineProperty(exports, "registryAddresses", { enumerable: true, get: function () { return config_js_1.registryAddresses; } });
__exportStar(require("./ens.js"), exports);
//# sourceMappingURL=index.js.map