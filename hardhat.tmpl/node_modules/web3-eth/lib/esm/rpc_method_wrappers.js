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
var __rest = (this && this.__rest) || function (s, e) {
    var t = {};
    for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p) && e.indexOf(p) < 0)
        t[p] = s[p];
    if (s != null && typeof Object.getOwnPropertySymbols === "function")
        for (var i = 0, p = Object.getOwnPropertySymbols(s); i < p.length; i++) {
            if (e.indexOf(p[i]) < 0 && Object.prototype.propertyIsEnumerable.call(s, p[i]))
                t[p[i]] = s[p[i]];
        }
    return t;
};
// Disabling because returnTypes must be last param to match 1.x params
/* eslint-disable default-param-last */
import { ETH_DATA_FORMAT, DEFAULT_RETURN_FORMAT, } from 'web3-types';
import { Web3PromiEvent } from 'web3-core';
import { format, hexToBytes, bytesToUint8Array, numberToHex } from 'web3-utils';
import { TransactionFactory } from 'web3-eth-accounts';
import { isBlockTag, isBytes, isNullish, isString } from 'web3-validator';
import { SignatureError } from 'web3-errors';
import { ethRpcMethods } from 'web3-rpc-methods';
import { decodeSignedTransaction } from './utils/decode_signed_transaction.js';
import { accountSchema, blockSchema, feeHistorySchema, logSchema, transactionReceiptSchema, accessListResultSchema, SignatureObjectSchema, } from './schemas.js';
// eslint-disable-next-line import/no-cycle
import { getTransactionFromOrToAttr } from './utils/transaction_builder.js';
import { formatTransaction } from './utils/format_transaction.js';
// eslint-disable-next-line import/no-cycle
import { trySendTransaction } from './utils/try_send_transaction.js';
// eslint-disable-next-line import/no-cycle
import { waitForTransactionReceipt } from './utils/wait_for_transaction_receipt.js';
import { NUMBER_DATA_FORMAT } from './constants.js';
// eslint-disable-next-line import/no-cycle
import { SendTxHelper } from './utils/send_tx_helper.js';
/**
 * View additional documentations here: {@link Web3Eth.getProtocolVersion}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export const getProtocolVersion = (web3Context) => __awaiter(void 0, void 0, void 0, function* () { return ethRpcMethods.getProtocolVersion(web3Context.requestManager); });
// TODO Add returnFormat parameter
/**
 * View additional documentations here: {@link Web3Eth.isSyncing}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export const isSyncing = (web3Context) => __awaiter(void 0, void 0, void 0, function* () { return ethRpcMethods.getSyncing(web3Context.requestManager); });
// TODO consider adding returnFormat parameter (to format address as bytes)
/**
 * View additional documentations here: {@link Web3Eth.getCoinbase}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export const getCoinbase = (web3Context) => __awaiter(void 0, void 0, void 0, function* () { return ethRpcMethods.getCoinbase(web3Context.requestManager); });
/**
 * View additional documentations here: {@link Web3Eth.isMining}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export const isMining = (web3Context) => __awaiter(void 0, void 0, void 0, function* () { return ethRpcMethods.getMining(web3Context.requestManager); });
/**
 * View additional documentations here: {@link Web3Eth.getHashRate}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getHashRate(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield ethRpcMethods.getHashRate(web3Context.requestManager);
        return format({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getGasPrice}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getGasPrice(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield ethRpcMethods.getGasPrice(web3Context.requestManager);
        return format({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getMaxPriorityFeePerGas}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getMaxPriorityFeePerGas(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield ethRpcMethods.getMaxPriorityFeePerGas(web3Context.requestManager);
        return format({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getBlockNumber}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getBlockNumber(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield ethRpcMethods.getBlockNumber(web3Context.requestManager);
        return format({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getBalance}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getBalance(web3Context_1, address_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, address, blockNumber = web3Context.defaultBlock, returnFormat) {
        const blockNumberFormatted = isBlockTag(blockNumber)
            ? blockNumber
            : format({ format: 'uint' }, blockNumber, ETH_DATA_FORMAT);
        const response = yield ethRpcMethods.getBalance(web3Context.requestManager, address, blockNumberFormatted);
        return format({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getStorageAt}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getStorageAt(web3Context_1, address_1, storageSlot_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, address, storageSlot, blockNumber = web3Context.defaultBlock, returnFormat) {
        const storageSlotFormatted = format({ format: 'uint' }, storageSlot, ETH_DATA_FORMAT);
        const blockNumberFormatted = isBlockTag(blockNumber)
            ? blockNumber
            : format({ format: 'uint' }, blockNumber, ETH_DATA_FORMAT);
        const response = yield ethRpcMethods.getStorageAt(web3Context.requestManager, address, storageSlotFormatted, blockNumberFormatted);
        return format({ format: 'bytes' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getCode}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getCode(web3Context_1, address_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, address, blockNumber = web3Context.defaultBlock, returnFormat) {
        const blockNumberFormatted = isBlockTag(blockNumber)
            ? blockNumber
            : format({ format: 'uint' }, blockNumber, ETH_DATA_FORMAT);
        const response = yield ethRpcMethods.getCode(web3Context.requestManager, address, blockNumberFormatted);
        return format({ format: 'bytes' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getBlock}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getBlock(web3Context_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, block = web3Context.defaultBlock, hydrated = false, returnFormat) {
        var _a;
        let response;
        if (isBytes(block)) {
            const blockHashFormatted = format({ format: 'bytes32' }, block, ETH_DATA_FORMAT);
            response = yield ethRpcMethods.getBlockByHash(web3Context.requestManager, blockHashFormatted, hydrated);
        }
        else {
            const blockNumberFormatted = isBlockTag(block)
                ? block
                : format({ format: 'uint' }, block, ETH_DATA_FORMAT);
            response = yield ethRpcMethods.getBlockByNumber(web3Context.requestManager, blockNumberFormatted, hydrated);
        }
        const res = format(blockSchema, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
        if (!isNullish(res)) {
            const result = Object.assign(Object.assign({}, res), { transactions: (_a = res.transactions) !== null && _a !== void 0 ? _a : [] });
            return result;
        }
        return res;
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getBlockTransactionCount}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getBlockTransactionCount(web3Context_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, block = web3Context.defaultBlock, returnFormat) {
        let response;
        if (isBytes(block)) {
            const blockHashFormatted = format({ format: 'bytes32' }, block, ETH_DATA_FORMAT);
            response = yield ethRpcMethods.getBlockTransactionCountByHash(web3Context.requestManager, blockHashFormatted);
        }
        else {
            const blockNumberFormatted = isBlockTag(block)
                ? block
                : format({ format: 'uint' }, block, ETH_DATA_FORMAT);
            response = yield ethRpcMethods.getBlockTransactionCountByNumber(web3Context.requestManager, blockNumberFormatted);
        }
        return format({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getBlockUncleCount}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getBlockUncleCount(web3Context_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, block = web3Context.defaultBlock, returnFormat) {
        let response;
        if (isBytes(block)) {
            const blockHashFormatted = format({ format: 'bytes32' }, block, ETH_DATA_FORMAT);
            response = yield ethRpcMethods.getUncleCountByBlockHash(web3Context.requestManager, blockHashFormatted);
        }
        else {
            const blockNumberFormatted = isBlockTag(block)
                ? block
                : format({ format: 'uint' }, block, ETH_DATA_FORMAT);
            response = yield ethRpcMethods.getUncleCountByBlockNumber(web3Context.requestManager, blockNumberFormatted);
        }
        return format({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getUncle}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getUncle(web3Context_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, block = web3Context.defaultBlock, uncleIndex, returnFormat) {
        const uncleIndexFormatted = format({ format: 'uint' }, uncleIndex, ETH_DATA_FORMAT);
        let response;
        if (isBytes(block)) {
            const blockHashFormatted = format({ format: 'bytes32' }, block, ETH_DATA_FORMAT);
            response = yield ethRpcMethods.getUncleByBlockHashAndIndex(web3Context.requestManager, blockHashFormatted, uncleIndexFormatted);
        }
        else {
            const blockNumberFormatted = isBlockTag(block)
                ? block
                : format({ format: 'uint' }, block, ETH_DATA_FORMAT);
            response = yield ethRpcMethods.getUncleByBlockNumberAndIndex(web3Context.requestManager, blockNumberFormatted, uncleIndexFormatted);
        }
        return format(blockSchema, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getTransaction}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getTransaction(web3Context_1, transactionHash_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, transactionHash, returnFormat = web3Context.defaultReturnFormat) {
        const transactionHashFormatted = format({ format: 'bytes32' }, transactionHash, DEFAULT_RETURN_FORMAT);
        const response = yield ethRpcMethods.getTransactionByHash(web3Context.requestManager, transactionHashFormatted);
        return isNullish(response)
            ? response
            : formatTransaction(response, returnFormat, {
                transactionSchema: web3Context.config.customTransactionSchema,
                fillInputAndData: true,
            });
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getPendingTransactions}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getPendingTransactions(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield ethRpcMethods.getPendingTransactions(web3Context.requestManager);
        return response.map(transaction => formatTransaction(transaction, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat, {
            transactionSchema: web3Context.config.customTransactionSchema,
            fillInputAndData: true,
        }));
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getTransactionFromBlock}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getTransactionFromBlock(web3Context_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, block = web3Context.defaultBlock, transactionIndex, returnFormat) {
        const transactionIndexFormatted = format({ format: 'uint' }, transactionIndex, ETH_DATA_FORMAT);
        let response;
        if (isBytes(block)) {
            const blockHashFormatted = format({ format: 'bytes32' }, block, ETH_DATA_FORMAT);
            response = yield ethRpcMethods.getTransactionByBlockHashAndIndex(web3Context.requestManager, blockHashFormatted, transactionIndexFormatted);
        }
        else {
            const blockNumberFormatted = isBlockTag(block)
                ? block
                : format({ format: 'uint' }, block, ETH_DATA_FORMAT);
            response = yield ethRpcMethods.getTransactionByBlockNumberAndIndex(web3Context.requestManager, blockNumberFormatted, transactionIndexFormatted);
        }
        return isNullish(response)
            ? response
            : formatTransaction(response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat, {
                transactionSchema: web3Context.config.customTransactionSchema,
                fillInputAndData: true,
            });
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getTransactionReceipt}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getTransactionReceipt(web3Context, transactionHash, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const transactionHashFormatted = format({ format: 'bytes32' }, transactionHash, DEFAULT_RETURN_FORMAT);
        let response;
        try {
            response = yield ethRpcMethods.getTransactionReceipt(web3Context.requestManager, transactionHashFormatted);
        }
        catch (error) {
            // geth indexing error, we poll until transactions stopped indexing
            if (typeof error === 'object' &&
                !isNullish(error) &&
                'message' in error &&
                error.message === 'transaction indexing is in progress') {
                console.warn('Transaction indexing is in progress.');
            }
            else {
                throw error;
            }
        }
        return isNullish(response)
            ? response
            : format(transactionReceiptSchema, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getTransactionCount}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getTransactionCount(web3Context_1, address_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, address, blockNumber = web3Context.defaultBlock, returnFormat) {
        const blockNumberFormatted = isBlockTag(blockNumber)
            ? blockNumber
            : format({ format: 'uint' }, blockNumber, ETH_DATA_FORMAT);
        const response = yield ethRpcMethods.getTransactionCount(web3Context.requestManager, address, blockNumberFormatted);
        return format({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.sendTransaction}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function sendTransaction(web3Context, transactionObj, returnFormat, options = { checkRevertBeforeSending: true }, transactionMiddleware) {
    const promiEvent = new Web3PromiEvent((resolve, reject) => {
        setImmediate(() => {
            (() => __awaiter(this, void 0, void 0, function* () {
                const sendTxHelper = new SendTxHelper({
                    web3Context,
                    promiEvent,
                    options,
                    returnFormat,
                });
                let transaction = Object.assign({}, transactionObj);
                if (!isNullish(transactionMiddleware)) {
                    transaction = yield transactionMiddleware.processTransaction(transaction);
                }
                let transactionFormatted = formatTransaction(Object.assign(Object.assign({}, transaction), { from: getTransactionFromOrToAttr('from', web3Context, transaction), to: getTransactionFromOrToAttr('to', web3Context, transaction) }), ETH_DATA_FORMAT, {
                    transactionSchema: web3Context.config.customTransactionSchema,
                });
                try {
                    transactionFormatted = (yield sendTxHelper.populateGasPrice({
                        transaction,
                        transactionFormatted,
                    }));
                    yield sendTxHelper.checkRevertBeforeSending(transactionFormatted);
                    sendTxHelper.emitSending(transactionFormatted);
                    let wallet;
                    if (web3Context.wallet && !isNullish(transactionFormatted.from)) {
                        wallet = web3Context.wallet.get(transactionFormatted.from);
                    }
                    const transactionHash = yield sendTxHelper.signAndSend({
                        wallet,
                        tx: transactionFormatted,
                    });
                    const transactionHashFormatted = format({ format: 'bytes32' }, transactionHash, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
                    sendTxHelper.emitSent(transactionFormatted);
                    sendTxHelper.emitTransactionHash(transactionHashFormatted);
                    const transactionReceipt = yield waitForTransactionReceipt(web3Context, transactionHash, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
                    const transactionReceiptFormatted = sendTxHelper.getReceiptWithEvents(format(transactionReceiptSchema, transactionReceipt, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat));
                    sendTxHelper.emitReceipt(transactionReceiptFormatted);
                    resolve(yield sendTxHelper.handleResolve({
                        receipt: transactionReceiptFormatted,
                        tx: transactionFormatted,
                    }));
                    sendTxHelper.emitConfirmation({
                        receipt: transactionReceiptFormatted,
                        transactionHash,
                    });
                }
                catch (error) {
                    reject(yield sendTxHelper.handleError({
                        error,
                        tx: transactionFormatted,
                    }));
                }
            }))();
        });
    });
    return promiEvent;
}
/**
 * View additional documentations here: {@link Web3Eth.sendSignedTransaction}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function sendSignedTransaction(web3Context, signedTransaction, returnFormat, options = { checkRevertBeforeSending: true }) {
    // TODO - Promise returned in function argument where a void return was expected
    // eslint-disable-next-line @typescript-eslint/no-misused-promises
    const promiEvent = new Web3PromiEvent((resolve, reject) => {
        setImmediate(() => {
            (() => __awaiter(this, void 0, void 0, function* () {
                const sendTxHelper = new SendTxHelper({
                    web3Context,
                    promiEvent,
                    options,
                    returnFormat,
                });
                // Formatting signedTransaction to be send to RPC endpoint
                const signedTransactionFormattedHex = format({ format: 'bytes' }, signedTransaction, ETH_DATA_FORMAT);
                const unSerializedTransaction = TransactionFactory.fromSerializedData(bytesToUint8Array(hexToBytes(signedTransactionFormattedHex)));
                const unSerializedTransactionWithFrom = Object.assign(Object.assign({}, unSerializedTransaction.toJSON()), { 
                    // Some providers will default `from` to address(0) causing the error
                    // reported from `eth_call` to not be the reason the user's tx failed
                    // e.g. `eth_call` will return an Out of Gas error for a failed
                    // smart contract execution contract, because the sender, address(0),
                    // has no balance to pay for the gas of the transaction execution
                    from: unSerializedTransaction.getSenderAddress().toString() });
                try {
                    const { v, r, s } = unSerializedTransactionWithFrom, txWithoutSigParams = __rest(unSerializedTransactionWithFrom, ["v", "r", "s"]);
                    yield sendTxHelper.checkRevertBeforeSending(txWithoutSigParams);
                    sendTxHelper.emitSending(signedTransactionFormattedHex);
                    const transactionHash = yield trySendTransaction(web3Context, () => __awaiter(this, void 0, void 0, function* () {
                        return ethRpcMethods.sendRawTransaction(web3Context.requestManager, signedTransactionFormattedHex);
                    }));
                    sendTxHelper.emitSent(signedTransactionFormattedHex);
                    const transactionHashFormatted = format({ format: 'bytes32' }, transactionHash, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
                    sendTxHelper.emitTransactionHash(transactionHashFormatted);
                    const transactionReceipt = yield waitForTransactionReceipt(web3Context, transactionHash, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
                    const transactionReceiptFormatted = sendTxHelper.getReceiptWithEvents(format(transactionReceiptSchema, transactionReceipt, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat));
                    sendTxHelper.emitReceipt(transactionReceiptFormatted);
                    resolve(yield sendTxHelper.handleResolve({
                        receipt: transactionReceiptFormatted,
                        tx: unSerializedTransactionWithFrom,
                    }));
                    sendTxHelper.emitConfirmation({
                        receipt: transactionReceiptFormatted,
                        transactionHash,
                    });
                }
                catch (error) {
                    reject(yield sendTxHelper.handleError({
                        error,
                        tx: unSerializedTransactionWithFrom,
                    }));
                }
            }))();
        });
    });
    return promiEvent;
}
/**
 * View additional documentations here: {@link Web3Eth.sign}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function sign(web3Context_1, message_1, addressOrIndex_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, message, addressOrIndex, returnFormat = web3Context.defaultReturnFormat) {
        var _a;
        const messageFormatted = format({ format: 'bytes' }, message, DEFAULT_RETURN_FORMAT);
        if ((_a = web3Context.wallet) === null || _a === void 0 ? void 0 : _a.get(addressOrIndex)) {
            const wallet = web3Context.wallet.get(addressOrIndex);
            const signed = wallet.sign(messageFormatted);
            return format(SignatureObjectSchema, signed, returnFormat);
        }
        if (typeof addressOrIndex === 'number') {
            throw new SignatureError(message, 'RPC method "eth_sign" does not support index signatures');
        }
        const response = yield ethRpcMethods.sign(web3Context.requestManager, addressOrIndex, messageFormatted);
        return format({ format: 'bytes' }, response, returnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.signTransaction}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function signTransaction(web3Context_1, transaction_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, transaction, returnFormat = web3Context.defaultReturnFormat) {
        const response = yield ethRpcMethods.signTransaction(web3Context.requestManager, formatTransaction(transaction, ETH_DATA_FORMAT, {
            transactionSchema: web3Context.config.customTransactionSchema,
        }));
        // Some clients only return the encoded signed transaction (e.g. Ganache)
        // while clients such as Geth return the desired SignedTransactionInfoAPI object
        return isString(response)
            ? decodeSignedTransaction(response, returnFormat, {
                fillInputAndData: true,
            })
            : {
                raw: format({ format: 'bytes' }, response.raw, returnFormat),
                tx: formatTransaction(response.tx, returnFormat, {
                    transactionSchema: web3Context.config.customTransactionSchema,
                    fillInputAndData: true,
                }),
            };
    });
}
// TODO Decide what to do with transaction.to
// https://github.com/ChainSafe/web3.js/pull/4525#issuecomment-982330076
/**
 * View additional documentations here: {@link Web3Eth.call}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function call(web3Context_1, transaction_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, transaction, blockNumber = web3Context.defaultBlock, returnFormat = web3Context.defaultReturnFormat) {
        const blockNumberFormatted = isBlockTag(blockNumber)
            ? blockNumber
            : format({ format: 'uint' }, blockNumber, ETH_DATA_FORMAT);
        const response = yield ethRpcMethods.call(web3Context.requestManager, formatTransaction(transaction, ETH_DATA_FORMAT, {
            transactionSchema: web3Context.config.customTransactionSchema,
        }), blockNumberFormatted);
        return format({ format: 'bytes' }, response, returnFormat);
    });
}
// TODO - Investigate whether response is padded as 1.x docs suggest
/**
 * View additional documentations here: {@link Web3Eth.estimateGas}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function estimateGas(web3Context_1, transaction_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, transaction, blockNumber = web3Context.defaultBlock, returnFormat) {
        const transactionFormatted = formatTransaction(transaction, ETH_DATA_FORMAT, {
            transactionSchema: web3Context.config.customTransactionSchema,
        });
        const blockNumberFormatted = isBlockTag(blockNumber)
            ? blockNumber
            : format({ format: 'uint' }, blockNumber, ETH_DATA_FORMAT);
        const response = yield ethRpcMethods.estimateGas(web3Context.requestManager, transactionFormatted, blockNumberFormatted);
        return format({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
// TODO - Add input formatting to filter
/**
 * View additional documentations here: {@link Web3Eth.getPastLogs}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getLogs(web3Context, filter, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        // format type bigint or number toBlock and fromBlock to hexstring.
        let { toBlock, fromBlock } = filter;
        if (!isNullish(toBlock)) {
            if (typeof toBlock === 'number' || typeof toBlock === 'bigint') {
                toBlock = numberToHex(toBlock);
            }
        }
        if (!isNullish(fromBlock)) {
            if (typeof fromBlock === 'number' || typeof fromBlock === 'bigint') {
                fromBlock = numberToHex(fromBlock);
            }
        }
        const formattedFilter = Object.assign(Object.assign({}, filter), { fromBlock, toBlock });
        const response = yield ethRpcMethods.getLogs(web3Context.requestManager, formattedFilter);
        const result = response.map(res => {
            if (typeof res === 'string') {
                return res;
            }
            return format(logSchema, res, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
        });
        return result;
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getChainId}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getChainId(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield ethRpcMethods.getChainId(web3Context.requestManager);
        return format({ format: 'uint' }, 
        // Response is number in hex formatted string
        response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getProof}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getProof(web3Context_1, address_1, storageKeys_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, address, storageKeys, blockNumber = web3Context.defaultBlock, returnFormat) {
        const storageKeysFormatted = storageKeys.map(storageKey => format({ format: 'bytes' }, storageKey, ETH_DATA_FORMAT));
        const blockNumberFormatted = isBlockTag(blockNumber)
            ? blockNumber
            : format({ format: 'uint' }, blockNumber, ETH_DATA_FORMAT);
        const response = yield ethRpcMethods.getProof(web3Context.requestManager, address, storageKeysFormatted, blockNumberFormatted);
        return format(accountSchema, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
// TODO Throwing an error with Geth, but not Infura
// TODO gasUsedRatio and reward not formatting
/**
 * View additional documentations here: {@link Web3Eth.getFeeHistory}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function getFeeHistory(web3Context_1, blockCount_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, blockCount, newestBlock = web3Context.defaultBlock, rewardPercentiles, returnFormat) {
        const blockCountFormatted = format({ format: 'uint' }, blockCount, ETH_DATA_FORMAT);
        const newestBlockFormatted = isBlockTag(newestBlock)
            ? newestBlock
            : format({ format: 'uint' }, newestBlock, ETH_DATA_FORMAT);
        const rewardPercentilesFormatted = format({
            type: 'array',
            items: {
                format: 'uint',
            },
        }, rewardPercentiles, NUMBER_DATA_FORMAT);
        const response = yield ethRpcMethods.getFeeHistory(web3Context.requestManager, blockCountFormatted, newestBlockFormatted, rewardPercentilesFormatted);
        return format(feeHistorySchema, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.createAccessList}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function createAccessList(web3Context_1, transaction_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, transaction, blockNumber = web3Context.defaultBlock, returnFormat) {
        const blockNumberFormatted = isBlockTag(blockNumber)
            ? blockNumber
            : format({ format: 'uint' }, blockNumber, ETH_DATA_FORMAT);
        const response = (yield ethRpcMethods.createAccessList(web3Context.requestManager, formatTransaction(transaction, ETH_DATA_FORMAT, {
            transactionSchema: web3Context.config.customTransactionSchema,
        }), blockNumberFormatted));
        return format(accessListResultSchema, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.signTypedData}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
export function signTypedData(web3Context, address, typedData, useLegacy, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield ethRpcMethods.signTypedData(web3Context.requestManager, address, typedData, useLegacy);
        return format({ format: 'bytes' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
//# sourceMappingURL=rpc_method_wrappers.js.map