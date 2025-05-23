"use strict";
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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Web3 = void 0;
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
const web3_core_1 = require("web3-core");
const web3_eth_1 = require("web3-eth");
const web3_eth_contract_1 = __importDefault(require("web3-eth-contract"));
const web3_eth_ens_1 = require("web3-eth-ens");
const web3_eth_iban_1 = require("web3-eth-iban");
const web3_eth_personal_1 = require("web3-eth-personal");
const web3_net_1 = require("web3-net");
const utils = __importStar(require("web3-utils"));
const web3_utils_1 = require("web3-utils");
const web3_rpc_providers_1 = require("web3-rpc-providers");
const web3_errors_1 = require("web3-errors");
const abi_js_1 = __importDefault(require("./abi.js"));
const accounts_js_1 = require("./accounts.js");
const version_js_1 = require("./version.js");
const web3_eip6963_js_1 = require("./web3_eip6963.js");
class Web3 extends web3_core_1.Web3Context {
    constructor(providerOrContext = web3_rpc_providers_1.mainnet) {
        var _a;
        if ((0, web3_utils_1.isNullish)(providerOrContext) ||
            (typeof providerOrContext === 'string' && providerOrContext.trim() === '') ||
            (typeof providerOrContext !== 'string' &&
                !(0, web3_core_1.isSupportedProvider)(providerOrContext) &&
                !providerOrContext.provider)) {
            console.warn('NOTE: web3.js is running without provider. You need to pass a provider in order to interact with the network!');
        }
        let contextInitOptions = {};
        if (typeof providerOrContext === 'string' ||
            (0, web3_core_1.isSupportedProvider)(providerOrContext)) {
            contextInitOptions.provider = providerOrContext;
        }
        else if (providerOrContext) {
            contextInitOptions = providerOrContext;
        }
        else {
            contextInitOptions = {};
        }
        contextInitOptions.registeredSubscriptions = Object.assign(Object.assign({}, web3_eth_1.registeredSubscriptions), ((_a = contextInitOptions.registeredSubscriptions) !== null && _a !== void 0 ? _a : {}));
        super(contextInitOptions);
        const accounts = (0, accounts_js_1.initAccountsForContext)(this);
        // Init protected properties
        this._wallet = accounts.wallet;
        this._accountProvider = accounts;
        this.utils = utils;
        // Have to use local alias to initiate contract context
        // eslint-disable-next-line @typescript-eslint/no-this-alias
        const self = this;
        class ContractBuilder extends web3_eth_contract_1.default {
            constructor(jsonInterface, addressOrOptionsOrContext, optionsOrContextOrReturnFormat, contextOrReturnFormat, returnFormat) {
                if ((0, web3_utils_1.isContractInitOptions)(addressOrOptionsOrContext) &&
                    (0, web3_utils_1.isContractInitOptions)(optionsOrContextOrReturnFormat)) {
                    throw new web3_errors_1.InvalidMethodParamsError('Should not provide options at both 2nd and 3rd parameters');
                }
                let address;
                let options = {};
                let context;
                let dataFormat;
                // add validation so its not a breaking change
                if (!(0, web3_utils_1.isNullish)(addressOrOptionsOrContext) &&
                    typeof addressOrOptionsOrContext !== 'object' &&
                    typeof addressOrOptionsOrContext !== 'string') {
                    throw new web3_errors_1.InvalidMethodParamsError();
                }
                if (typeof addressOrOptionsOrContext === 'string') {
                    address = addressOrOptionsOrContext;
                }
                if ((0, web3_utils_1.isContractInitOptions)(addressOrOptionsOrContext)) {
                    options = addressOrOptionsOrContext;
                }
                else if ((0, web3_utils_1.isContractInitOptions)(optionsOrContextOrReturnFormat)) {
                    options = optionsOrContextOrReturnFormat;
                }
                else {
                    options = {};
                }
                if (addressOrOptionsOrContext instanceof web3_core_1.Web3Context) {
                    context = addressOrOptionsOrContext;
                }
                else if (optionsOrContextOrReturnFormat instanceof web3_core_1.Web3Context) {
                    context = optionsOrContextOrReturnFormat;
                }
                else if (contextOrReturnFormat instanceof web3_core_1.Web3Context) {
                    context = contextOrReturnFormat;
                }
                else {
                    context = self.getContextObject();
                }
                if (returnFormat) {
                    dataFormat = returnFormat;
                }
                else if ((0, web3_utils_1.isDataFormat)(optionsOrContextOrReturnFormat)) {
                    dataFormat = optionsOrContextOrReturnFormat;
                }
                else if ((0, web3_utils_1.isDataFormat)(contextOrReturnFormat)) {
                    dataFormat = contextOrReturnFormat;
                }
                super(jsonInterface, address, options, context, dataFormat);
                super.subscribeToContextEvents(self);
                // eslint-disable-next-line no-use-before-define
                if (!(0, web3_utils_1.isNullish)(eth)) {
                    // eslint-disable-next-line no-use-before-define
                    const TxMiddleware = eth.getTransactionMiddleware();
                    if (!(0, web3_utils_1.isNullish)(TxMiddleware)) {
                        super.setTransactionMiddleware(TxMiddleware);
                    }
                }
            }
        }
        const eth = self.use(web3_eth_1.Web3Eth);
        // Eth Module
        this.eth = Object.assign(eth, {
            // ENS module
            ens: self.use(web3_eth_ens_1.ENS, web3_eth_ens_1.registryAddresses.main), // registry address defaults to main network
            // Iban helpers
            Iban: // registry address defaults to main network
            web3_eth_iban_1.Iban,
            net: self.use(web3_net_1.Net),
            personal: self.use(web3_eth_personal_1.Personal),
            // Contract helper and module
            Contract: ContractBuilder,
            // ABI Helpers
            abi: abi_js_1.default,
            // Accounts helper
            accounts,
        });
    }
}
exports.Web3 = Web3;
Web3.version = version_js_1.Web3PkgInfo.version;
Web3.utils = utils;
Web3.requestEIP6963Providers = web3_eip6963_js_1.requestEIP6963Providers;
Web3.onNewProviderDiscovered = web3_eip6963_js_1.onNewProviderDiscovered;
Web3.modules = {
    Web3Eth: web3_eth_1.Web3Eth,
    Iban: web3_eth_iban_1.Iban,
    Net: web3_net_1.Net,
    ENS: web3_eth_ens_1.ENS,
    Personal: web3_eth_personal_1.Personal,
};
exports.default = Web3;
//# sourceMappingURL=web3.js.map