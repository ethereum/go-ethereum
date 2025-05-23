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
import { format } from 'web3-utils';
import { netRpcMethods } from 'web3-rpc-methods';
export function getId(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield netRpcMethods.getId(web3Context.requestManager);
        return format({ format: 'uint' }, response, returnFormat);
    });
}
export function getPeerCount(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield netRpcMethods.getPeerCount(web3Context.requestManager);
        // Data returned is number in hex format
        return format({ format: 'uint' }, response, returnFormat);
    });
}
export const isListening = (web3Context) => __awaiter(void 0, void 0, void 0, function* () { return netRpcMethods.isListening(web3Context.requestManager); });
//# sourceMappingURL=rpc_method_wrappers.js.map