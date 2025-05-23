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
 * The web3.eth.accounts contains functions to generate Ethereum accounts and sign transactions and data.
 *
 * **_NOTE:_** This package has NOT been audited and might potentially be unsafe. Take precautions to clear memory properly, store the private keys safely, and test transaction receiving and sending functionality properly before using in production!
 *
 *
 * To use this package standalone and use its methods use:
 * ```ts
 * import { create, decrypt } from 'web3-eth-accounts'; // ....
 * ```
 *
 * To use this package within the web3 object use:
 *
 * ```ts
 * import Web3 from 'web3';
 *
 * const web3 = new Web3(Web3.givenProvider || 'ws://some.local-or-remote.node:8546');
 * // now you have access to the accounts class
 * web3.eth.accounts.create();
 * ```
 */
__exportStar(require("./wallet.js"), exports);
__exportStar(require("./account.js"), exports);
__exportStar(require("./types.js"), exports);
__exportStar(require("./schemas.js"), exports);
__exportStar(require("./common/index.js"), exports);
__exportStar(require("./tx/index.js"), exports);
//# sourceMappingURL=index.js.map