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
exports.onNewProviderDiscovered = exports.requestEIP6963Providers = exports.web3ProvidersMapUpdated = exports.eip6963ProvidersMap = exports.Eip6963EventName = void 0;
var Eip6963EventName;
(function (Eip6963EventName) {
    Eip6963EventName["eip6963announceProvider"] = "eip6963:announceProvider";
    Eip6963EventName["eip6963requestProvider"] = "eip6963:requestProvider";
})(Eip6963EventName || (exports.Eip6963EventName = Eip6963EventName = {}));
exports.eip6963ProvidersMap = new Map();
exports.web3ProvidersMapUpdated = 'web3:providersMapUpdated';
const requestEIP6963Providers = () => __awaiter(void 0, void 0, void 0, function* () {
    return new Promise((resolve, reject) => {
        if (typeof window === 'undefined') {
            reject(new Error('window object not available, EIP-6963 is intended to be used within a browser'));
        }
        window.addEventListener(Eip6963EventName.eip6963announceProvider, ((event) => {
            exports.eip6963ProvidersMap.set(event.detail.info.uuid, event.detail);
            const newEvent = new CustomEvent(exports.web3ProvidersMapUpdated, { detail: exports.eip6963ProvidersMap });
            window.dispatchEvent(newEvent);
            resolve(exports.eip6963ProvidersMap);
        }));
        window.dispatchEvent(new Event(Eip6963EventName.eip6963requestProvider));
    });
});
exports.requestEIP6963Providers = requestEIP6963Providers;
const onNewProviderDiscovered = (callback) => {
    if (typeof window === 'undefined') {
        throw new Error('window object not available, EIP-6963 is intended to be used within a browser');
    }
    window.addEventListener(exports.web3ProvidersMapUpdated, callback);
};
exports.onNewProviderDiscovered = onNewProviderDiscovered;
//# sourceMappingURL=web3_eip6963.js.map