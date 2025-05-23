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
import { isNullish } from 'web3-validator';
import { Eip1559NotSupportedError, UnsupportedTransactionTypeError } from 'web3-errors';
import { format } from 'web3-utils';
// eslint-disable-next-line import/no-cycle
import { getBlock, getGasPrice } from '../rpc_method_wrappers.js';
// eslint-disable-next-line import/no-cycle
import { getTransactionType } from './transaction_builder.js';
function getEip1559GasPricing(transaction, web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        var _a, _b, _c, _d;
        const block = yield getBlock(web3Context, web3Context.defaultBlock, false, ETH_DATA_FORMAT);
        if (isNullish(block.baseFeePerGas))
            throw new Eip1559NotSupportedError();
        let gasPrice;
        if (isNullish(transaction.gasPrice) && BigInt(block.baseFeePerGas) === BigInt(0)) {
            gasPrice = yield getGasPrice(web3Context, returnFormat);
        }
        if (!isNullish(transaction.gasPrice) || !isNullish(gasPrice)) {
            const convertedTransactionGasPrice = format({ format: 'uint' }, (_a = transaction.gasPrice) !== null && _a !== void 0 ? _a : gasPrice, returnFormat);
            return {
                maxPriorityFeePerGas: convertedTransactionGasPrice,
                maxFeePerGas: convertedTransactionGasPrice,
            };
        }
        return {
            maxPriorityFeePerGas: format({ format: 'uint' }, (_b = transaction.maxPriorityFeePerGas) !== null && _b !== void 0 ? _b : web3Context.defaultMaxPriorityFeePerGas, returnFormat),
            maxFeePerGas: format({ format: 'uint' }, ((_c = transaction.maxFeePerGas) !== null && _c !== void 0 ? _c : BigInt(block.baseFeePerGas) * BigInt(2) +
                BigInt((_d = transaction.maxPriorityFeePerGas) !== null && _d !== void 0 ? _d : web3Context.defaultMaxPriorityFeePerGas)), returnFormat),
        };
    });
}
export function getTransactionGasPricing(transaction, web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const transactionType = getTransactionType(transaction, web3Context);
        if (!isNullish(transactionType)) {
            if (transactionType.startsWith('-'))
                throw new UnsupportedTransactionTypeError(transactionType);
            // https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2718.md#transactions
            if (Number(transactionType) < 0 || Number(transactionType) > 127)
                throw new UnsupportedTransactionTypeError(transactionType);
            if (isNullish(transaction.gasPrice) &&
                (transactionType === '0x0' || transactionType === '0x1'))
                return {
                    gasPrice: yield getGasPrice(web3Context, returnFormat),
                    maxPriorityFeePerGas: undefined,
                    maxFeePerGas: undefined,
                };
            if (transactionType === '0x2') {
                return Object.assign({ gasPrice: undefined }, (yield getEip1559GasPricing(transaction, web3Context, returnFormat)));
            }
        }
        return undefined;
    });
}
//# sourceMappingURL=get_transaction_gas_pricing.js.map