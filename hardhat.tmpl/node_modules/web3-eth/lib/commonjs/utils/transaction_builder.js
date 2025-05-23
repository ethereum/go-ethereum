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
exports.transactionBuilder = exports.getTransactionType = exports.getTransactionNonce = exports.getTransactionFromOrToAttr = void 0;
exports.defaultTransactionBuilder = defaultTransactionBuilder;
const web3_types_1 = require("web3-types");
const web3_eth_accounts_1 = require("web3-eth-accounts");
const web3_net_1 = require("web3-net");
const web3_validator_1 = require("web3-validator");
const web3_errors_1 = require("web3-errors");
const web3_utils_1 = require("web3-utils");
const constants_js_1 = require("../constants.js");
// eslint-disable-next-line import/no-cycle
const rpc_method_wrappers_js_1 = require("../rpc_method_wrappers.js");
const detect_transaction_type_js_1 = require("./detect_transaction_type.js");
const schemas_js_1 = require("../schemas.js");
// eslint-disable-next-line import/no-cycle
const get_transaction_gas_pricing_js_1 = require("./get_transaction_gas_pricing.js");
const getTransactionFromOrToAttr = (attr, web3Context, transaction, privateKey) => {
    if (transaction !== undefined && attr in transaction && transaction[attr] !== undefined) {
        if (typeof transaction[attr] === 'string' && (0, web3_validator_1.isAddress)(transaction[attr])) {
            // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
            return transaction[attr];
        }
        if (!(0, web3_validator_1.isHexStrict)(transaction[attr]) && (0, web3_validator_1.isNumber)(transaction[attr])) {
            if (web3Context.wallet) {
                const account = web3Context.wallet.get((0, web3_utils_1.format)({ format: 'uint' }, transaction[attr], constants_js_1.NUMBER_DATA_FORMAT));
                if (!(0, web3_validator_1.isNullish)(account)) {
                    return account.address;
                }
                throw new web3_errors_1.LocalWalletNotAvailableError();
            }
            throw new web3_errors_1.LocalWalletNotAvailableError();
        }
        else {
            throw attr === 'from'
                ? new web3_errors_1.InvalidTransactionWithSender(transaction.from)
                : // eslint-disable-next-line @typescript-eslint/no-unsafe-call
                    new web3_errors_1.InvalidTransactionWithReceiver(transaction.to);
        }
    }
    if (attr === 'from') {
        if (!(0, web3_validator_1.isNullish)(privateKey))
            return (0, web3_eth_accounts_1.privateKeyToAddress)(privateKey);
        if (!(0, web3_validator_1.isNullish)(web3Context.defaultAccount))
            return web3Context.defaultAccount;
    }
    return undefined;
};
exports.getTransactionFromOrToAttr = getTransactionFromOrToAttr;
const getTransactionNonce = (web3Context_1, address_1, ...args_1) => __awaiter(void 0, [web3Context_1, address_1, ...args_1], void 0, function* (web3Context, address, returnFormat = web3Context.defaultReturnFormat) {
    if ((0, web3_validator_1.isNullish)(address)) {
        // TODO if (web3.eth.accounts.wallet) use address from local wallet
        throw new web3_errors_1.UnableToPopulateNonceError();
    }
    return (0, rpc_method_wrappers_js_1.getTransactionCount)(web3Context, address, web3Context.defaultBlock, returnFormat);
});
exports.getTransactionNonce = getTransactionNonce;
const getTransactionType = (transaction, web3Context) => {
    const inferredType = (0, detect_transaction_type_js_1.detectTransactionType)(transaction, web3Context);
    if (!(0, web3_validator_1.isNullish)(inferredType))
        return inferredType;
    if (!(0, web3_validator_1.isNullish)(web3Context.defaultTransactionType))
        return (0, web3_utils_1.format)({ format: 'uint' }, web3Context.defaultTransactionType, web3_types_1.ETH_DATA_FORMAT);
    return undefined;
};
exports.getTransactionType = getTransactionType;
// Keep in mind that the order the properties of populateTransaction get populated matters
// as some of the properties are dependent on others
function defaultTransactionBuilder(options) {
    return __awaiter(this, void 0, void 0, function* () {
        var _a, _b;
        let populatedTransaction = (0, web3_utils_1.format)(schemas_js_1.transactionSchema, options.transaction, options.web3Context.defaultReturnFormat);
        if ((0, web3_validator_1.isNullish)(populatedTransaction.from)) {
            populatedTransaction.from = (0, exports.getTransactionFromOrToAttr)('from', options.web3Context, undefined, options.privateKey);
        }
        // TODO: Debug why need to typecase getTransactionNonce
        if ((0, web3_validator_1.isNullish)(populatedTransaction.nonce)) {
            populatedTransaction.nonce = yield (0, exports.getTransactionNonce)(options.web3Context, populatedTransaction.from, web3_types_1.ETH_DATA_FORMAT);
        }
        if ((0, web3_validator_1.isNullish)(populatedTransaction.value)) {
            populatedTransaction.value = '0x0';
        }
        if (!(0, web3_validator_1.isNullish)(populatedTransaction.data)) {
            if (!(0, web3_validator_1.isNullish)(populatedTransaction.input) &&
                populatedTransaction.data !== populatedTransaction.input)
                throw new web3_errors_1.TransactionDataAndInputError({
                    data: (0, web3_utils_1.bytesToHex)(populatedTransaction.data),
                    input: (0, web3_utils_1.bytesToHex)(populatedTransaction.input),
                });
            if (!populatedTransaction.data.startsWith('0x'))
                populatedTransaction.data = `0x${populatedTransaction.data}`;
        }
        else if (!(0, web3_validator_1.isNullish)(populatedTransaction.input)) {
            if (!populatedTransaction.input.startsWith('0x'))
                populatedTransaction.input = `0x${populatedTransaction.input}`;
        }
        else {
            populatedTransaction.input = '0x';
        }
        if ((0, web3_validator_1.isNullish)(populatedTransaction.common)) {
            if (options.web3Context.defaultCommon) {
                const common = options.web3Context.defaultCommon;
                const chainId = common.customChain.chainId;
                const networkId = common.customChain.networkId;
                const name = common.customChain.name;
                populatedTransaction.common = Object.assign(Object.assign({}, common), { customChain: { chainId, networkId, name } });
            }
            if ((0, web3_validator_1.isNullish)(populatedTransaction.chain)) {
                populatedTransaction.chain = options.web3Context.defaultChain;
            }
            if ((0, web3_validator_1.isNullish)(populatedTransaction.hardfork)) {
                populatedTransaction.hardfork = options.web3Context.defaultHardfork;
            }
        }
        if ((0, web3_validator_1.isNullish)(populatedTransaction.chainId) &&
            (0, web3_validator_1.isNullish)((_a = populatedTransaction.common) === null || _a === void 0 ? void 0 : _a.customChain.chainId)) {
            populatedTransaction.chainId = yield (0, rpc_method_wrappers_js_1.getChainId)(options.web3Context, web3_types_1.ETH_DATA_FORMAT);
        }
        if ((0, web3_validator_1.isNullish)(populatedTransaction.networkId)) {
            populatedTransaction.networkId =
                (_b = options.web3Context.defaultNetworkId) !== null && _b !== void 0 ? _b : (yield (0, web3_net_1.getId)(options.web3Context, web3_types_1.ETH_DATA_FORMAT));
        }
        if ((0, web3_validator_1.isNullish)(populatedTransaction.gasLimit) && !(0, web3_validator_1.isNullish)(populatedTransaction.gas)) {
            populatedTransaction.gasLimit = populatedTransaction.gas;
        }
        populatedTransaction.type = (0, exports.getTransactionType)(populatedTransaction, options.web3Context);
        if ((0, web3_validator_1.isNullish)(populatedTransaction.accessList) &&
            (populatedTransaction.type === '0x1' || populatedTransaction.type === '0x2')) {
            populatedTransaction.accessList = [];
        }
        if (options.fillGasPrice)
            populatedTransaction = Object.assign(Object.assign({}, populatedTransaction), (yield (0, get_transaction_gas_pricing_js_1.getTransactionGasPricing)(populatedTransaction, options.web3Context, web3_types_1.ETH_DATA_FORMAT)));
        if ((0, web3_validator_1.isNullish)(populatedTransaction.gas) &&
            (0, web3_validator_1.isNullish)(populatedTransaction.gasLimit) &&
            options.fillGasLimit) {
            const fillGasLimit = yield (0, rpc_method_wrappers_js_1.estimateGas)(options.web3Context, populatedTransaction, 'latest', web3_types_1.ETH_DATA_FORMAT);
            populatedTransaction = Object.assign(Object.assign({}, populatedTransaction), { gas: (0, web3_utils_1.format)({ format: 'uint' }, fillGasLimit, web3_types_1.ETH_DATA_FORMAT) });
        }
        return populatedTransaction;
    });
}
const transactionBuilder = (options) => __awaiter(void 0, void 0, void 0, function* () {
    var _a;
    return ((_a = options.web3Context.transactionBuilder) !== null && _a !== void 0 ? _a : defaultTransactionBuilder)(Object.assign(Object.assign({}, options), { transaction: options.transaction }));
});
exports.transactionBuilder = transactionBuilder;
//# sourceMappingURL=transaction_builder.js.map