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
exports.isListening = exports.getPeerCount = exports.getId = void 0;
const web3_utils_1 = require("web3-utils");
const web3_rpc_methods_1 = require("web3-rpc-methods");
function getId(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.netRpcMethods.getId(web3Context.requestManager);
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat);
    });
}
exports.getId = getId;
function getPeerCount(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.netRpcMethods.getPeerCount(web3Context.requestManager);
        // Data returned is number in hex format
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat);
    });
}
exports.getPeerCount = getPeerCount;
const isListening = (web3Context) => __awaiter(void 0, void 0, void 0, function* () { return web3_rpc_methods_1.netRpcMethods.isListening(web3Context.requestManager); });
exports.isListening = isListening;
//# sourceMappingURL=rpc_method_wrappers.js.map