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
import { ETH_DATA_FORMAT, } from 'web3-types';
import { privateKeyToAddress } from 'web3-eth-accounts';
import { getId } from 'web3-net';
import { isNullish, isNumber, isHexStrict, isAddress } from 'web3-validator';
import { InvalidTransactionWithSender, InvalidTransactionWithReceiver, LocalWalletNotAvailableError, TransactionDataAndInputError, UnableToPopulateNonceError, } from 'web3-errors';
import { bytesToHex, format } from 'web3-utils';
import { NUMBER_DATA_FORMAT } from '../constants.js';
// eslint-disable-next-line import/no-cycle
import { getChainId, getTransactionCount, estimateGas } from '../rpc_method_wrappers.js';
import { detectTransactionType } from './detect_transaction_type.js';
import { transactionSchema } from '../schemas.js';
// eslint-disable-next-line import/no-cycle
import { getTransactionGasPricing } from './get_transaction_gas_pricing.js';
export const getTransactionFromOrToAttr = (attr, web3Context, transaction, privateKey) => {
    if (transaction !== undefined && attr in transaction && transaction[attr] !== undefined) {
        if (typeof transaction[attr] === 'string' && isAddress(transaction[attr])) {
            // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
            return transaction[attr];
        }
        if (!isHexStrict(transaction[attr]) && isNumber(transaction[attr])) {
            if (web3Context.wallet) {
                const account = web3Context.wallet.get(format({ format: 'uint' }, transaction[attr], NUMBER_DATA_FORMAT));
                if (!isNullish(account)) {
                    return account.address;
                }
                throw new LocalWalletNotAvailableError();
            }
            throw new LocalWalletNotAvailableError();
        }
        else {
            throw attr === 'from'
                ? new InvalidTransactionWithSender(transaction.from)
                : // eslint-disable-next-line @typescript-eslint/no-unsafe-call
                    new InvalidTransactionWithReceiver(transaction.to);
        }
    }
    if (attr === 'from') {
        if (!isNullish(privateKey))
            return privateKeyToAddress(privateKey);
        if (!isNullish(web3Context.defaultAccount))
            return web3Context.defaultAccount;
    }
    return undefined;
};
export const getTransactionNonce = (web3Context_1, address_1, ...args_1) => __awaiter(void 0, [web3Context_1, address_1, ...args_1], void 0, function* (web3Context, address, returnFormat = web3Context.defaultReturnFormat) {
    if (isNullish(address)) {
        // TODO if (web3.eth.accounts.wallet) use address from local wallet
        throw new UnableToPopulateNonceError();
    }
    return getTransactionCount(web3Context, address, web3Context.defaultBlock, returnFormat);
});
export const getTransactionType = (transaction, web3Context) => {
    const inferredType = detectTransactionType(transaction, web3Context);
    if (!isNullish(inferredType))
        return inferredType;
    if (!isNullish(web3Context.defaultTransactionType))
        return format({ format: 'uint' }, web3Context.defaultTransactionType, ETH_DATA_FORMAT);
    return undefined;
};
// Keep in mind that the order the properties of populateTransaction get populated matters
// as some of the properties are dependent on others
export function defaultTransactionBuilder(options) {
    return __awaiter(this, void 0, void 0, function* () {
        var _a, _b;
        let populatedTransaction = format(transactionSchema, options.transaction, options.web3Context.defaultReturnFormat);
        if (isNullish(populatedTransaction.from)) {
            populatedTransaction.from = getTransactionFromOrToAttr('from', options.web3Context, undefined, options.privateKey);
        }
        // TODO: Debug why need to typecase getTransactionNonce
        if (isNullish(populatedTransaction.nonce)) {
            populatedTransaction.nonce = yield getTransactionNonce(options.web3Context, populatedTransaction.from, ETH_DATA_FORMAT);
        }
        if (isNullish(populatedTransaction.value)) {
            populatedTransaction.value = '0x0';
        }
        if (!isNullish(populatedTransaction.data)) {
            if (!isNullish(populatedTransaction.input) &&
                populatedTransaction.data !== populatedTransaction.input)
                throw new TransactionDataAndInputError({
                    data: bytesToHex(populatedTransaction.data),
                    input: bytesToHex(populatedTransaction.input),
                });
            if (!populatedTransaction.data.startsWith('0x'))
                populatedTransaction.data = `0x${populatedTransaction.data}`;
        }
        else if (!isNullish(populatedTransaction.input)) {
            if (!populatedTransaction.input.startsWith('0x'))
                populatedTransaction.input = `0x${populatedTransaction.input}`;
        }
        else {
            populatedTransaction.input = '0x';
        }
        if (isNullish(populatedTransaction.common)) {
            if (options.web3Context.defaultCommon) {
                const common = options.web3Context.defaultCommon;
                const chainId = common.customChain.chainId;
                const networkId = common.customChain.networkId;
                const name = common.customChain.name;
                populatedTransaction.common = Object.assign(Object.assign({}, common), { customChain: { chainId, networkId, name } });
            }
            if (isNullish(populatedTransaction.chain)) {
                populatedTransaction.chain = options.web3Context.defaultChain;
            }
            if (isNullish(populatedTransaction.hardfork)) {
                populatedTransaction.hardfork = options.web3Context.defaultHardfork;
            }
        }
        if (isNullish(populatedTransaction.chainId) &&
            isNullish((_a = populatedTransaction.common) === null || _a === void 0 ? void 0 : _a.customChain.chainId)) {
            populatedTransaction.chainId = yield getChainId(options.web3Context, ETH_DATA_FORMAT);
        }
        if (isNullish(populatedTransaction.networkId)) {
            populatedTransaction.networkId =
                (_b = options.web3Context.defaultNetworkId) !== null && _b !== void 0 ? _b : (yield getId(options.web3Context, ETH_DATA_FORMAT));
        }
        if (isNullish(populatedTransaction.gasLimit) && !isNullish(populatedTransaction.gas)) {
            populatedTransaction.gasLimit = populatedTransaction.gas;
        }
        populatedTransaction.type = getTransactionType(populatedTransaction, options.web3Context);
        if (isNullish(populatedTransaction.accessList) &&
            (populatedTransaction.type === '0x1' || populatedTransaction.type === '0x2')) {
            populatedTransaction.accessList = [];
        }
        if (options.fillGasPrice)
            populatedTransaction = Object.assign(Object.assign({}, populatedTransaction), (yield getTransactionGasPricing(populatedTransaction, options.web3Context, ETH_DATA_FORMAT)));
        if (isNullish(populatedTransaction.gas) &&
            isNullish(populatedTransaction.gasLimit) &&
            options.fillGasLimit) {
            const fillGasLimit = yield estimateGas(options.web3Context, populatedTransaction, 'latest', ETH_DATA_FORMAT);
            populatedTransaction = Object.assign(Object.assign({}, populatedTransaction), { gas: format({ format: 'uint' }, fillGasLimit, ETH_DATA_FORMAT) });
        }
        return populatedTransaction;
    });
}
export const transactionBuilder = (options) => __awaiter(void 0, void 0, void 0, function* () {
    var _a;
    return ((_a = options.web3Context.transactionBuilder) !== null && _a !== void 0 ? _a : defaultTransactionBuilder)(Object.assign(Object.assign({}, options), { transaction: options.transaction }));
});
//# sourceMappingURL=transaction_builder.js.map