var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
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
import { ETH_DATA_FORMAT, } from 'web3-types';
import { isNullish } from 'web3-validator';
import { ContractExecutionError, InvalidResponseError, TransactionPollingTimeoutError, TransactionRevertedWithoutReasonError, TransactionRevertInstructionError, TransactionRevertWithCustomError, } from 'web3-errors';
import { ethRpcMethods } from 'web3-rpc-methods';
// eslint-disable-next-line import/no-cycle
import { getTransactionGasPricing } from './get_transaction_gas_pricing.js';
// eslint-disable-next-line import/no-cycle
import { trySendTransaction } from './try_send_transaction.js';
// eslint-disable-next-line import/no-cycle
import { watchTransactionForConfirmations } from './watch_transaction_for_confirmations.js';
import { ALL_EVENTS_ABI } from '../constants.js';
// eslint-disable-next-line import/no-cycle
import { getTransactionError } from './get_transaction_error.js';
// eslint-disable-next-line import/no-cycle
import { getRevertReason } from './get_revert_reason.js';
import { decodeEventABI } from './decoding.js';
export class SendTxHelper {
    constructor({ options, web3Context, promiEvent, returnFormat, }) {
        this.options = {
            checkRevertBeforeSending: true,
        };
        this.options = options;
        this.web3Context = web3Context;
        this.promiEvent = promiEvent;
        this.returnFormat = returnFormat;
    }
    getReceiptWithEvents(data) {
        var _a, _b;
        const result = Object.assign({}, (data !== null && data !== void 0 ? data : {}));
        if (((_a = this.options) === null || _a === void 0 ? void 0 : _a.contractAbi) && result.logs && result.logs.length > 0) {
            result.events = {};
            for (const log of result.logs) {
                const event = decodeEventABI(ALL_EVENTS_ABI, log, (_b = this.options) === null || _b === void 0 ? void 0 : _b.contractAbi, this.returnFormat);
                if (event.event) {
                    result.events[event.event] = event;
                }
            }
        }
        return result;
    }
    checkRevertBeforeSending(tx) {
        return __awaiter(this, void 0, void 0, function* () {
            if (this.options.checkRevertBeforeSending !== false) {
                let formatTx = tx;
                if (isNullish(tx.data) && isNullish(tx.input) && isNullish(tx.gas)) {
                    // eth.call runs into error if data isnt filled and gas is not defined, its a simple transaction so we fill it with 21000
                    formatTx = Object.assign(Object.assign({}, tx), { gas: 21000 });
                }
                const reason = yield getRevertReason(this.web3Context, formatTx, this.options.contractAbi);
                if (reason !== undefined) {
                    throw yield getTransactionError(this.web3Context, tx, undefined, undefined, this.options.contractAbi, reason);
                }
            }
        });
    }
    emitSending(tx) {
        if (this.promiEvent.listenerCount('sending') > 0) {
            this.promiEvent.emit('sending', tx);
        }
    }
    populateGasPrice(_a) {
        return __awaiter(this, arguments, void 0, function* ({ transactionFormatted, transaction, }) {
            var _b;
            let result = transactionFormatted;
            if (!this.web3Context.config.ignoreGasPricing &&
                !((_b = this.options) === null || _b === void 0 ? void 0 : _b.ignoreGasPricing) &&
                isNullish(transactionFormatted.gasPrice) &&
                (isNullish(transaction.maxPriorityFeePerGas) ||
                    isNullish(transaction.maxFeePerGas))) {
                result = Object.assign(Object.assign({}, transactionFormatted), (yield getTransactionGasPricing(transactionFormatted, this.web3Context, ETH_DATA_FORMAT)));
            }
            return result;
        });
    }
    signAndSend(_a) {
        return __awaiter(this, arguments, void 0, function* ({ wallet, tx, }) {
            if (wallet) {
                const signedTransaction = yield wallet.signTransaction(tx);
                return trySendTransaction(this.web3Context, () => __awaiter(this, void 0, void 0, function* () {
                    return ethRpcMethods.sendRawTransaction(this.web3Context.requestManager, signedTransaction.rawTransaction);
                }), signedTransaction.transactionHash);
            }
            return trySendTransaction(this.web3Context, () => __awaiter(this, void 0, void 0, function* () {
                return ethRpcMethods.sendTransaction(this.web3Context.requestManager, tx);
            }));
        });
    }
    emitSent(tx) {
        if (this.promiEvent.listenerCount('sent') > 0) {
            this.promiEvent.emit('sent', tx);
        }
    }
    emitTransactionHash(hash) {
        if (this.promiEvent.listenerCount('transactionHash') > 0) {
            this.promiEvent.emit('transactionHash', hash);
        }
    }
    emitReceipt(receipt) {
        if (this.promiEvent.listenerCount('receipt') > 0) {
            this.promiEvent.emit('receipt', 
            // @ts-expect-error unknown type fix
            receipt);
        }
    }
    handleError(_a) {
        return __awaiter(this, arguments, void 0, function* ({ error, tx }) {
            var _b;
            let _error = error;
            if (_error instanceof ContractExecutionError && this.web3Context.handleRevert) {
                _error = yield getTransactionError(this.web3Context, tx, undefined, undefined, (_b = this.options) === null || _b === void 0 ? void 0 : _b.contractAbi);
            }
            if ((_error instanceof InvalidResponseError ||
                _error instanceof ContractExecutionError ||
                _error instanceof TransactionRevertWithCustomError ||
                _error instanceof TransactionRevertedWithoutReasonError ||
                _error instanceof TransactionRevertInstructionError ||
                _error instanceof TransactionPollingTimeoutError) &&
                this.promiEvent.listenerCount('error') > 0) {
                this.promiEvent.emit('error', _error);
            }
            return _error;
        });
    }
    emitConfirmation({ receipt, transactionHash, customTransactionReceiptSchema, }) {
        if (this.promiEvent.listenerCount('confirmation') > 0) {
            watchTransactionForConfirmations(this.web3Context, this.promiEvent, receipt, transactionHash, this.returnFormat, customTransactionReceiptSchema);
        }
    }
    handleResolve(_a) {
        return __awaiter(this, arguments, void 0, function* ({ receipt, tx }) {
            var _b, _c, _d;
            if ((_b = this.options) === null || _b === void 0 ? void 0 : _b.transactionResolver) {
                return (_c = this.options) === null || _c === void 0 ? void 0 : _c.transactionResolver(receipt);
            }
            if (receipt.status === BigInt(0)) {
                const error = yield getTransactionError(this.web3Context, tx, 
                // @ts-expect-error unknown type fix
                receipt, undefined, (_d = this.options) === null || _d === void 0 ? void 0 : _d.contractAbi);
                if (this.promiEvent.listenerCount('error') > 0) {
                    this.promiEvent.emit('error', error);
                }
                throw error;
            }
            else {
                return receipt;
            }
        });
    }
}
//# sourceMappingURL=send_tx_helper.js.map