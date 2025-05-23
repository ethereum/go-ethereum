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
 * The web3-net package allows you to interact with an Ethereum node’s network properties.
 *
 * ```ts
 * import Net from 'web3-net';
 *
 * const net = new Net(Net.givenProvider || 'ws://some.local-or-remote.node:8546');
 * // or using the web3 umbrella package
 * import Web3 from 'web3';
 * const web3 = new Web3(Web3.givenProvider || 'ws://some.local-or-remote.node:8546');
 *
 * // -> web3.eth.net
 *
 * // get the ID of the network
 * await web3.eth.net.getId();
 * > 5777n
 *
 * // get the peer count
 * await web3.eth.net.getPeerCount();
 * > 0n
 *
 * // Check if the node is listening for peers
 * await web3.eth.net.isListening();
 * > true
 * ```
 */
/**
 *
 */
const net_js_1 = require("./net.js");
__exportStar(require("./net.js"), exports);
__exportStar(require("./rpc_method_wrappers.js"), exports);
exports.default = net_js_1.Net;
//# sourceMappingURL=index.js.map