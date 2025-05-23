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
// eslint-disable-next-line max-classes-per-file
import { Web3Context, isSupportedProvider, } from 'web3-core';
import { Web3Eth, registeredSubscriptions } from 'web3-eth';
import Contract from 'web3-eth-contract';
import { ENS, registryAddresses } from 'web3-eth-ens';
import { Iban } from 'web3-eth-iban';
import { Personal } from 'web3-eth-personal';
import { Net } from 'web3-net';
import * as utils from 'web3-utils';
import { isNullish, isDataFormat, isContractInitOptions } from 'web3-utils';
import { mainnet } from 'web3-rpc-providers';
import { InvalidMethodParamsError } from 'web3-errors';
import abi from './abi.js';
import { initAccountsForContext } from './accounts.js';
import { Web3PkgInfo } from './version.js';
import { onNewProviderDiscovered, requestEIP6963Providers } from './web3_eip6963.js';
export class Web3 extends Web3Context {
    constructor(providerOrContext = mainnet) {
        var _a;
        if (isNullish(providerOrContext) ||
            (typeof providerOrContext === 'string' && providerOrContext.trim() === '') ||
            (typeof providerOrContext !== 'string' &&
                !isSupportedProvider(providerOrContext) &&
                !providerOrContext.provider)) {
            console.warn('NOTE: web3.js is running without provider. You need to pass a provider in order to interact with the network!');
        }
        let contextInitOptions = {};
        if (typeof providerOrContext === 'string' ||
            isSupportedProvider(providerOrContext)) {
            contextInitOptions.provider = providerOrContext;
        }
        else if (providerOrContext) {
            contextInitOptions = providerOrContext;
        }
        else {
            contextInitOptions = {};
        }
        contextInitOptions.registeredSubscriptions = Object.assign(Object.assign({}, registeredSubscriptions), ((_a = contextInitOptions.registeredSubscriptions) !== null && _a !== void 0 ? _a : {}));
        super(contextInitOptions);
        const accounts = initAccountsForContext(this);
        // Init protected properties
        this._wallet = accounts.wallet;
        this._accountProvider = accounts;
        this.utils = utils;
        // Have to use local alias to initiate contract context
        // eslint-disable-next-line @typescript-eslint/no-this-alias
        const self = this;
        class ContractBuilder extends Contract {
            constructor(jsonInterface, addressOrOptionsOrContext, optionsOrContextOrReturnFormat, contextOrReturnFormat, returnFormat) {
                if (isContractInitOptions(addressOrOptionsOrContext) &&
                    isContractInitOptions(optionsOrContextOrReturnFormat)) {
                    throw new InvalidMethodParamsError('Should not provide options at both 2nd and 3rd parameters');
                }
                let address;
                let options = {};
                let context;
                let dataFormat;
                // add validation so its not a breaking change
                if (!isNullish(addressOrOptionsOrContext) &&
                    typeof addressOrOptionsOrContext !== 'object' &&
                    typeof addressOrOptionsOrContext !== 'string') {
                    throw new InvalidMethodParamsError();
                }
                if (typeof addressOrOptionsOrContext === 'string') {
                    address = addressOrOptionsOrContext;
                }
                if (isContractInitOptions(addressOrOptionsOrContext)) {
                    options = addressOrOptionsOrContext;
                }
                else if (isContractInitOptions(optionsOrContextOrReturnFormat)) {
                    options = optionsOrContextOrReturnFormat;
                }
                else {
                    options = {};
                }
                if (addressOrOptionsOrContext instanceof Web3Context) {
                    context = addressOrOptionsOrContext;
                }
                else if (optionsOrContextOrReturnFormat instanceof Web3Context) {
                    context = optionsOrContextOrReturnFormat;
                }
                else if (contextOrReturnFormat instanceof Web3Context) {
                    context = contextOrReturnFormat;
                }
                else {
                    context = self.getContextObject();
                }
                if (returnFormat) {
                    dataFormat = returnFormat;
                }
                else if (isDataFormat(optionsOrContextOrReturnFormat)) {
                    dataFormat = optionsOrContextOrReturnFormat;
                }
                else if (isDataFormat(contextOrReturnFormat)) {
                    dataFormat = contextOrReturnFormat;
                }
                super(jsonInterface, address, options, context, dataFormat);
                super.subscribeToContextEvents(self);
                // eslint-disable-next-line no-use-before-define
                if (!isNullish(eth)) {
                    // eslint-disable-next-line no-use-before-define
                    const TxMiddleware = eth.getTransactionMiddleware();
                    if (!isNullish(TxMiddleware)) {
                        super.setTransactionMiddleware(TxMiddleware);
                    }
                }
            }
        }
        const eth = self.use(Web3Eth);
        // Eth Module
        this.eth = Object.assign(eth, {
            // ENS module
            ens: self.use(ENS, registryAddresses.main), // registry address defaults to main network
            // Iban helpers
            Iban,
            net: self.use(Net),
            personal: self.use(Personal),
            // Contract helper and module
            Contract: ContractBuilder,
            // ABI Helpers
            abi,
            // Accounts helper
            accounts,
        });
    }
}
Web3.version = Web3PkgInfo.version;
Web3.utils = utils;
Web3.requestEIP6963Providers = requestEIP6963Providers;
Web3.onNewProviderDiscovered = onNewProviderDiscovered;
Web3.modules = {
    Web3Eth,
    Iban,
    Net,
    ENS,
    Personal,
};
export default Web3;
//# sourceMappingURL=web3.js.map