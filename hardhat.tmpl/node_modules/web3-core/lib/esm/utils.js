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
import { Web3BaseProvider, } from 'web3-types';
export const isWeb3Provider = (provider) => Web3BaseProvider.isWeb3Provider(provider);
export const isMetaMaskProvider = (provider) => typeof provider !== 'string' &&
    'request' in provider &&
    provider.request.constructor.name === 'AsyncFunction' &&
    'isMetaMask' in provider &&
    provider.isMetaMask;
export const isLegacyRequestProvider = (provider) => typeof provider !== 'string' &&
    'request' in provider &&
    provider.request.constructor.name === 'Function';
export const isEIP1193Provider = (provider) => typeof provider !== 'string' &&
    'request' in provider &&
    provider.request.constructor.name === 'AsyncFunction';
export const isLegacySendProvider = (provider) => typeof provider !== 'string' && 'send' in provider;
export const isLegacySendAsyncProvider = (provider) => typeof provider !== 'string' && 'sendAsync' in provider;
export const isSupportedProvider = (provider) => provider &&
    (isWeb3Provider(provider) ||
        isEIP1193Provider(provider) ||
        isLegacyRequestProvider(provider) ||
        isLegacySendAsyncProvider(provider) ||
        isLegacySendProvider(provider));
export const isSupportSubscriptions = (provider) => {
    if (provider && 'supportsSubscriptions' in provider) {
        return provider.supportsSubscriptions();
    }
    if (provider && typeof provider !== 'string' && 'on' in provider) {
        return true;
    }
    return false;
};
//# sourceMappingURL=utils.js.map