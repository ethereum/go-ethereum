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
Object.defineProperty(exports, "__esModule", { value: true });
exports.isSupportSubscriptions = exports.isSupportedProvider = exports.isLegacySendAsyncProvider = exports.isLegacySendProvider = exports.isEIP1193Provider = exports.isLegacyRequestProvider = exports.isMetaMaskProvider = exports.isWeb3Provider = void 0;
const web3_types_1 = require("web3-types");
const isWeb3Provider = (provider) => web3_types_1.Web3BaseProvider.isWeb3Provider(provider);
exports.isWeb3Provider = isWeb3Provider;
const isMetaMaskProvider = (provider) => typeof provider !== 'string' &&
    'request' in provider &&
    provider.request.constructor.name === 'AsyncFunction' &&
    'isMetaMask' in provider &&
    provider.isMetaMask;
exports.isMetaMaskProvider = isMetaMaskProvider;
const isLegacyRequestProvider = (provider) => typeof provider !== 'string' &&
    'request' in provider &&
    provider.request.constructor.name === 'Function';
exports.isLegacyRequestProvider = isLegacyRequestProvider;
const isEIP1193Provider = (provider) => typeof provider !== 'string' &&
    'request' in provider &&
    provider.request.constructor.name === 'AsyncFunction';
exports.isEIP1193Provider = isEIP1193Provider;
const isLegacySendProvider = (provider) => typeof provider !== 'string' && 'send' in provider;
exports.isLegacySendProvider = isLegacySendProvider;
const isLegacySendAsyncProvider = (provider) => typeof provider !== 'string' && 'sendAsync' in provider;
exports.isLegacySendAsyncProvider = isLegacySendAsyncProvider;
const isSupportedProvider = (provider) => provider &&
    ((0, exports.isWeb3Provider)(provider) ||
        (0, exports.isEIP1193Provider)(provider) ||
        (0, exports.isLegacyRequestProvider)(provider) ||
        (0, exports.isLegacySendAsyncProvider)(provider) ||
        (0, exports.isLegacySendProvider)(provider));
exports.isSupportedProvider = isSupportedProvider;
const isSupportSubscriptions = (provider) => {
    if (provider && 'supportsSubscriptions' in provider) {
        return provider.supportsSubscriptions();
    }
    if (provider && typeof provider !== 'string' && 'on' in provider) {
        return true;
    }
    return false;
};
exports.isSupportSubscriptions = isSupportSubscriptions;
//# sourceMappingURL=utils.js.map