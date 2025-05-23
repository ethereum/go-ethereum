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
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __exportStar = (this && this.__exportStar) || function(m, exports) {
    for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports, p)) __createBinding(exports, m, p);
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.txUtils = exports.BaseTransaction = exports.TransactionFactory = exports.Transaction = exports.AccessListEIP2930Transaction = exports.FeeMarketEIP1559Transaction = void 0;
// @ethereumjs/tx version 4.1.1
var eip1559Transaction_js_1 = require("./eip1559Transaction.js");
Object.defineProperty(exports, "FeeMarketEIP1559Transaction", { enumerable: true, get: function () { return eip1559Transaction_js_1.FeeMarketEIP1559Transaction; } });
var eip2930Transaction_js_1 = require("./eip2930Transaction.js");
Object.defineProperty(exports, "AccessListEIP2930Transaction", { enumerable: true, get: function () { return eip2930Transaction_js_1.AccessListEIP2930Transaction; } });
var legacyTransaction_js_1 = require("./legacyTransaction.js");
Object.defineProperty(exports, "Transaction", { enumerable: true, get: function () { return legacyTransaction_js_1.Transaction; } });
var transactionFactory_js_1 = require("./transactionFactory.js");
Object.defineProperty(exports, "TransactionFactory", { enumerable: true, get: function () { return transactionFactory_js_1.TransactionFactory; } });
var baseTransaction_js_1 = require("./baseTransaction.js");
Object.defineProperty(exports, "BaseTransaction", { enumerable: true, get: function () { return baseTransaction_js_1.BaseTransaction; } });
exports.txUtils = __importStar(require("./utils.js"));
__exportStar(require("./types.js"), exports);
//# sourceMappingURL=index.js.map