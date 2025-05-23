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
exports.SendTxHelper = exports.trySendTransaction = exports.waitForTransactionReceipt = exports.getTransactionFromOrToAttr = exports.transactionBuilder = exports.detectTransactionType = void 0;
/**
 * The `web3-eth` package allows you to interact with an Ethereum blockchain and Ethereum smart contracts.
 *
 * To use this package standalone and use its methods use:
 * ```ts
 * import { Web3Context } from 'web3-core';
 * import { BlockTags } from 'web3-types';
 * import { DEFAULT_RETURN_FORMAT } from 'web3-types';
 * import { getBalance} from 'web3-eth';
 *
 * getBalance(
 *      new Web3Context('http://127.0.0.1:8545'),
 *      '0x407d73d8a49eeb85d32cf465507dd71d507100c1',
 *      BlockTags.LATEST,
 *      DEFAULT_RETURN_FORMAT
 * ).then(console.log);
 * > 1000000000000n
 * ```
 *
 * To use this package within the `web3` object use:
 * ```ts
 * import Web3 from 'web3';
 *
 * const web3 = new Web3(Web3.givenProvider || 'ws://some.local-or-remote.node:8546');
 * web3.eth.getBalance('0x407d73d8a49eeb85d32cf465507dd71d507100c1').then(console.log);
 * > 1000000000000n
 *```
 *
 * With `web3-eth` you can also subscribe (if supported by provider) to events in the Ethereum Blockchain, using the `subscribe` function. See more at the {@link Web3Eth.subscribe} function.
 */
/**
 *
 */
require("setimmediate");
const web3_eth_js_1 = require("./web3_eth.js");
__exportStar(require("./web3_eth.js"), exports);
__exportStar(require("./utils/decoding.js"), exports);
__exportStar(require("./schemas.js"), exports);
__exportStar(require("./constants.js"), exports);
__exportStar(require("./types.js"), exports);
__exportStar(require("./validation.js"), exports);
__exportStar(require("./rpc_method_wrappers.js"), exports);
__exportStar(require("./utils/format_transaction.js"), exports);
__exportStar(require("./utils/prepare_transaction_for_signing.js"), exports);
__exportStar(require("./web3_subscriptions.js"), exports);
var detect_transaction_type_js_1 = require("./utils/detect_transaction_type.js");
Object.defineProperty(exports, "detectTransactionType", { enumerable: true, get: function () { return detect_transaction_type_js_1.detectTransactionType; } });
var transaction_builder_js_1 = require("./utils/transaction_builder.js");
Object.defineProperty(exports, "transactionBuilder", { enumerable: true, get: function () { return transaction_builder_js_1.transactionBuilder; } });
Object.defineProperty(exports, "getTransactionFromOrToAttr", { enumerable: true, get: function () { return transaction_builder_js_1.getTransactionFromOrToAttr; } });
var wait_for_transaction_receipt_js_1 = require("./utils/wait_for_transaction_receipt.js");
Object.defineProperty(exports, "waitForTransactionReceipt", { enumerable: true, get: function () { return wait_for_transaction_receipt_js_1.waitForTransactionReceipt; } });
var try_send_transaction_js_1 = require("./utils/try_send_transaction.js");
Object.defineProperty(exports, "trySendTransaction", { enumerable: true, get: function () { return try_send_transaction_js_1.trySendTransaction; } });
var send_tx_helper_js_1 = require("./utils/send_tx_helper.js");
Object.defineProperty(exports, "SendTxHelper", { enumerable: true, get: function () { return send_tx_helper_js_1.SendTxHelper; } });
exports.default = web3_eth_js_1.Web3Eth;
//# sourceMappingURL=index.js.map