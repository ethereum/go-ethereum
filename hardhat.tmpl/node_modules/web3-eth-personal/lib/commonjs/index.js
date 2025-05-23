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
 * The `web3-eth-personal` package allows you to interact with the Ethereum node’s accounts.
 *
 * **_NOTE:_**  Many of these functions send sensitive information like passwords. Never call these functions over a unsecured Websocket or HTTP provider, as your password will be sent in plain text!
 *
 * import Personal from 'web3-eth-personal';
 *
 * const personal = new Personal('http://localhost:8545');
 *
 * or using the web3 umbrella package
 *
 * import Personal from 'web3-eth-personal';
 * const web3 = new Web3('http://localhost:8545');
 * // web3.eth.personal
 */
/**
 * This comment _supports3_ [Markdown](https://marked.js.org/)
 */
const personal_js_1 = require("./personal.js");
__exportStar(require("./personal.js"), exports);
exports.default = personal_js_1.Personal;
//# sourceMappingURL=index.js.map