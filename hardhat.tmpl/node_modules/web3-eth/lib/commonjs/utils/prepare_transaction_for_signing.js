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
exports.prepareTransactionForSigning = void 0;
const web3_types_1 = require("web3-types");
const web3_utils_1 = require("web3-utils");
const web3_eth_accounts_1 = require("web3-eth-accounts");
const web3_validator_1 = require("web3-validator");
const validation_js_1 = require("../validation.js");
const format_transaction_js_1 = require("./format_transaction.js");
const transaction_builder_js_1 = require("./transaction_builder.js");
const getEthereumjsTxDataFromTransaction = (transaction) => {
    var _a, _b;
    return (Object.assign(Object.assign({}, transaction), { nonce: transaction.nonce, gasPrice: transaction.gasPrice, gasLimit: (_a = transaction.gasLimit) !== null && _a !== void 0 ? _a : transaction.gas, to: transaction.to, value: transaction.value, data: (_b = transaction.data) !== null && _b !== void 0 ? _b : transaction.input, type: transaction.type, chainId: transaction.chainId, accessList: transaction.accessList, maxPriorityFeePerGas: transaction.maxPriorityFeePerGas, maxFeePerGas: transaction.maxFeePerGas }));
};
const getEthereumjsTransactionOptions = (transaction, web3Context) => {
    var _a, _b, _c, _d, _e, _f, _g, _h, _j, _k, _l, _m, _o, _p, _q, _r, _s, _t;
    const hasTransactionSigningOptions = (!(0, web3_validator_1.isNullish)(transaction.chain) && !(0, web3_validator_1.isNullish)(transaction.hardfork)) ||
        !(0, web3_validator_1.isNullish)(transaction.common);
    let common;
    if (!hasTransactionSigningOptions) {
        // if defaultcommon is specified, use that.
        if (web3Context.defaultCommon) {
            common = Object.assign({}, web3Context.defaultCommon);
            if ((0, web3_validator_1.isNullish)(common.hardfork))
                common.hardfork = (_a = transaction.hardfork) !== null && _a !== void 0 ? _a : web3Context.defaultHardfork;
            if ((0, web3_validator_1.isNullish)(common.baseChain))
                common.baseChain = web3Context.defaultChain;
        }
        else {
            common = web3_eth_accounts_1.Common.custom({
                name: 'custom-network',
                chainId: (0, web3_utils_1.toNumber)(transaction.chainId),
                networkId: !(0, web3_validator_1.isNullish)(transaction.networkId)
                    ? (0, web3_utils_1.toNumber)(transaction.networkId)
                    : undefined,
                defaultHardfork: (_b = transaction.hardfork) !== null && _b !== void 0 ? _b : web3Context.defaultHardfork,
            }, {
                baseChain: web3Context.defaultChain,
            });
        }
    }
    else {
        const name = (_f = (_e = (_d = (_c = transaction === null || transaction === void 0 ? void 0 : transaction.common) === null || _c === void 0 ? void 0 : _c.customChain) === null || _d === void 0 ? void 0 : _d.name) !== null && _e !== void 0 ? _e : transaction.chain) !== null && _f !== void 0 ? _f : 'custom-network';
        const chainId = (0, web3_utils_1.toNumber)((_j = (_h = (_g = transaction === null || transaction === void 0 ? void 0 : transaction.common) === null || _g === void 0 ? void 0 : _g.customChain) === null || _h === void 0 ? void 0 : _h.chainId) !== null && _j !== void 0 ? _j : transaction === null || transaction === void 0 ? void 0 : transaction.chainId);
        const networkId = (0, web3_utils_1.toNumber)((_m = (_l = (_k = transaction === null || transaction === void 0 ? void 0 : transaction.common) === null || _k === void 0 ? void 0 : _k.customChain) === null || _l === void 0 ? void 0 : _l.networkId) !== null && _m !== void 0 ? _m : transaction === null || transaction === void 0 ? void 0 : transaction.networkId);
        const defaultHardfork = (_q = (_p = (_o = transaction === null || transaction === void 0 ? void 0 : transaction.common) === null || _o === void 0 ? void 0 : _o.hardfork) !== null && _p !== void 0 ? _p : transaction === null || transaction === void 0 ? void 0 : transaction.hardfork) !== null && _q !== void 0 ? _q : web3Context.defaultHardfork;
        const baseChain = (_t = (_s = (_r = transaction.common) === null || _r === void 0 ? void 0 : _r.baseChain) !== null && _s !== void 0 ? _s : transaction.chain) !== null && _t !== void 0 ? _t : web3Context.defaultChain;
        if (chainId && networkId && name) {
            common = web3_eth_accounts_1.Common.custom({
                name,
                chainId,
                networkId,
                defaultHardfork,
            }, {
                baseChain,
            });
        }
    }
    return { common };
};
const prepareTransactionForSigning = (transaction_1, web3Context_1, privateKey_1, ...args_1) => __awaiter(void 0, [transaction_1, web3Context_1, privateKey_1, ...args_1], void 0, function* (transaction, web3Context, privateKey, fillGasPrice = false, fillGasLimit = true) {
    const populatedTransaction = (yield (0, transaction_builder_js_1.transactionBuilder)({
        transaction,
        web3Context,
        privateKey,
        fillGasPrice,
        fillGasLimit,
    }));
    const formattedTransaction = (0, format_transaction_js_1.formatTransaction)(populatedTransaction, web3_types_1.ETH_DATA_FORMAT, {
        transactionSchema: web3Context.config.customTransactionSchema,
    });
    (0, validation_js_1.validateTransactionForSigning)(formattedTransaction, undefined, {
        transactionSchema: web3Context.config.customTransactionSchema,
    });
    return web3_eth_accounts_1.TransactionFactory.fromTxData(getEthereumjsTxDataFromTransaction(formattedTransaction), getEthereumjsTransactionOptions(formattedTransaction, web3Context));
});
exports.prepareTransactionForSigning = prepareTransactionForSigning;
//# sourceMappingURL=prepare_transaction_for_signing.js.map