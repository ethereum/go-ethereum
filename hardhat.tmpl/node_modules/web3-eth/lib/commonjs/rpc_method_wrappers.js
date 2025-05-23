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
Object.defineProperty(exports, "__esModule", { value: true });
exports.isMining = exports.getCoinbase = exports.isSyncing = exports.getProtocolVersion = void 0;
exports.getHashRate = getHashRate;
exports.getGasPrice = getGasPrice;
exports.getMaxPriorityFeePerGas = getMaxPriorityFeePerGas;
exports.getBlockNumber = getBlockNumber;
exports.getBalance = getBalance;
exports.getStorageAt = getStorageAt;
exports.getCode = getCode;
exports.getBlock = getBlock;
exports.getBlockTransactionCount = getBlockTransactionCount;
exports.getBlockUncleCount = getBlockUncleCount;
exports.getUncle = getUncle;
exports.getTransaction = getTransaction;
exports.getPendingTransactions = getPendingTransactions;
exports.getTransactionFromBlock = getTransactionFromBlock;
exports.getTransactionReceipt = getTransactionReceipt;
exports.getTransactionCount = getTransactionCount;
exports.sendTransaction = sendTransaction;
exports.sendSignedTransaction = sendSignedTransaction;
exports.sign = sign;
exports.signTransaction = signTransaction;
exports.call = call;
exports.estimateGas = estimateGas;
exports.getLogs = getLogs;
exports.getChainId = getChainId;
exports.getProof = getProof;
exports.getFeeHistory = getFeeHistory;
exports.createAccessList = createAccessList;
exports.signTypedData = signTypedData;
// Disabling because returnTypes must be last param to match 1.x params
/* eslint-disable default-param-last */
const web3_types_1 = require("web3-types");
const web3_core_1 = require("web3-core");
const web3_utils_1 = require("web3-utils");
const web3_eth_accounts_1 = require("web3-eth-accounts");
const web3_validator_1 = require("web3-validator");
const web3_errors_1 = require("web3-errors");
const web3_rpc_methods_1 = require("web3-rpc-methods");
const decode_signed_transaction_js_1 = require("./utils/decode_signed_transaction.js");
const schemas_js_1 = require("./schemas.js");
// eslint-disable-next-line import/no-cycle
const transaction_builder_js_1 = require("./utils/transaction_builder.js");
const format_transaction_js_1 = require("./utils/format_transaction.js");
// eslint-disable-next-line import/no-cycle
const try_send_transaction_js_1 = require("./utils/try_send_transaction.js");
// eslint-disable-next-line import/no-cycle
const wait_for_transaction_receipt_js_1 = require("./utils/wait_for_transaction_receipt.js");
const constants_js_1 = require("./constants.js");
// eslint-disable-next-line import/no-cycle
const send_tx_helper_js_1 = require("./utils/send_tx_helper.js");
/**
 * View additional documentations here: {@link Web3Eth.getProtocolVersion}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
const getProtocolVersion = (web3Context) => __awaiter(void 0, void 0, void 0, function* () { return web3_rpc_methods_1.ethRpcMethods.getProtocolVersion(web3Context.requestManager); });
exports.getProtocolVersion = getProtocolVersion;
// TODO Add returnFormat parameter
/**
 * View additional documentations here: {@link Web3Eth.isSyncing}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
const isSyncing = (web3Context) => __awaiter(void 0, void 0, void 0, function* () { return web3_rpc_methods_1.ethRpcMethods.getSyncing(web3Context.requestManager); });
exports.isSyncing = isSyncing;
// TODO consider adding returnFormat parameter (to format address as bytes)
/**
 * View additional documentations here: {@link Web3Eth.getCoinbase}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
const getCoinbase = (web3Context) => __awaiter(void 0, void 0, void 0, function* () { return web3_rpc_methods_1.ethRpcMethods.getCoinbase(web3Context.requestManager); });
exports.getCoinbase = getCoinbase;
/**
 * View additional documentations here: {@link Web3Eth.isMining}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
const isMining = (web3Context) => __awaiter(void 0, void 0, void 0, function* () { return web3_rpc_methods_1.ethRpcMethods.getMining(web3Context.requestManager); });
exports.isMining = isMining;
/**
 * View additional documentations here: {@link Web3Eth.getHashRate}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getHashRate(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.ethRpcMethods.getHashRate(web3Context.requestManager);
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getGasPrice}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getGasPrice(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.ethRpcMethods.getGasPrice(web3Context.requestManager);
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getMaxPriorityFeePerGas}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getMaxPriorityFeePerGas(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.ethRpcMethods.getMaxPriorityFeePerGas(web3Context.requestManager);
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getBlockNumber}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getBlockNumber(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.ethRpcMethods.getBlockNumber(web3Context.requestManager);
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getBalance}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getBalance(web3Context_1, address_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, address, blockNumber = web3Context.defaultBlock, returnFormat) {
        const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(blockNumber)
            ? blockNumber
            : (0, web3_utils_1.format)({ format: 'uint' }, blockNumber, web3_types_1.ETH_DATA_FORMAT);
        const response = yield web3_rpc_methods_1.ethRpcMethods.getBalance(web3Context.requestManager, address, blockNumberFormatted);
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getStorageAt}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getStorageAt(web3Context_1, address_1, storageSlot_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, address, storageSlot, blockNumber = web3Context.defaultBlock, returnFormat) {
        const storageSlotFormatted = (0, web3_utils_1.format)({ format: 'uint' }, storageSlot, web3_types_1.ETH_DATA_FORMAT);
        const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(blockNumber)
            ? blockNumber
            : (0, web3_utils_1.format)({ format: 'uint' }, blockNumber, web3_types_1.ETH_DATA_FORMAT);
        const response = yield web3_rpc_methods_1.ethRpcMethods.getStorageAt(web3Context.requestManager, address, storageSlotFormatted, blockNumberFormatted);
        return (0, web3_utils_1.format)({ format: 'bytes' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getCode}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getCode(web3Context_1, address_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, address, blockNumber = web3Context.defaultBlock, returnFormat) {
        const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(blockNumber)
            ? blockNumber
            : (0, web3_utils_1.format)({ format: 'uint' }, blockNumber, web3_types_1.ETH_DATA_FORMAT);
        const response = yield web3_rpc_methods_1.ethRpcMethods.getCode(web3Context.requestManager, address, blockNumberFormatted);
        return (0, web3_utils_1.format)({ format: 'bytes' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getBlock}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getBlock(web3Context_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, block = web3Context.defaultBlock, hydrated = false, returnFormat) {
        var _a;
        let response;
        if ((0, web3_validator_1.isBytes)(block)) {
            const blockHashFormatted = (0, web3_utils_1.format)({ format: 'bytes32' }, block, web3_types_1.ETH_DATA_FORMAT);
            response = yield web3_rpc_methods_1.ethRpcMethods.getBlockByHash(web3Context.requestManager, blockHashFormatted, hydrated);
        }
        else {
            const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(block)
                ? block
                : (0, web3_utils_1.format)({ format: 'uint' }, block, web3_types_1.ETH_DATA_FORMAT);
            response = yield web3_rpc_methods_1.ethRpcMethods.getBlockByNumber(web3Context.requestManager, blockNumberFormatted, hydrated);
        }
        const res = (0, web3_utils_1.format)(schemas_js_1.blockSchema, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
        if (!(0, web3_validator_1.isNullish)(res)) {
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
function getBlockTransactionCount(web3Context_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, block = web3Context.defaultBlock, returnFormat) {
        let response;
        if ((0, web3_validator_1.isBytes)(block)) {
            const blockHashFormatted = (0, web3_utils_1.format)({ format: 'bytes32' }, block, web3_types_1.ETH_DATA_FORMAT);
            response = yield web3_rpc_methods_1.ethRpcMethods.getBlockTransactionCountByHash(web3Context.requestManager, blockHashFormatted);
        }
        else {
            const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(block)
                ? block
                : (0, web3_utils_1.format)({ format: 'uint' }, block, web3_types_1.ETH_DATA_FORMAT);
            response = yield web3_rpc_methods_1.ethRpcMethods.getBlockTransactionCountByNumber(web3Context.requestManager, blockNumberFormatted);
        }
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getBlockUncleCount}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getBlockUncleCount(web3Context_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, block = web3Context.defaultBlock, returnFormat) {
        let response;
        if ((0, web3_validator_1.isBytes)(block)) {
            const blockHashFormatted = (0, web3_utils_1.format)({ format: 'bytes32' }, block, web3_types_1.ETH_DATA_FORMAT);
            response = yield web3_rpc_methods_1.ethRpcMethods.getUncleCountByBlockHash(web3Context.requestManager, blockHashFormatted);
        }
        else {
            const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(block)
                ? block
                : (0, web3_utils_1.format)({ format: 'uint' }, block, web3_types_1.ETH_DATA_FORMAT);
            response = yield web3_rpc_methods_1.ethRpcMethods.getUncleCountByBlockNumber(web3Context.requestManager, blockNumberFormatted);
        }
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getUncle}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getUncle(web3Context_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, block = web3Context.defaultBlock, uncleIndex, returnFormat) {
        const uncleIndexFormatted = (0, web3_utils_1.format)({ format: 'uint' }, uncleIndex, web3_types_1.ETH_DATA_FORMAT);
        let response;
        if ((0, web3_validator_1.isBytes)(block)) {
            const blockHashFormatted = (0, web3_utils_1.format)({ format: 'bytes32' }, block, web3_types_1.ETH_DATA_FORMAT);
            response = yield web3_rpc_methods_1.ethRpcMethods.getUncleByBlockHashAndIndex(web3Context.requestManager, blockHashFormatted, uncleIndexFormatted);
        }
        else {
            const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(block)
                ? block
                : (0, web3_utils_1.format)({ format: 'uint' }, block, web3_types_1.ETH_DATA_FORMAT);
            response = yield web3_rpc_methods_1.ethRpcMethods.getUncleByBlockNumberAndIndex(web3Context.requestManager, blockNumberFormatted, uncleIndexFormatted);
        }
        return (0, web3_utils_1.format)(schemas_js_1.blockSchema, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getTransaction}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getTransaction(web3Context_1, transactionHash_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, transactionHash, returnFormat = web3Context.defaultReturnFormat) {
        const transactionHashFormatted = (0, web3_utils_1.format)({ format: 'bytes32' }, transactionHash, web3_types_1.DEFAULT_RETURN_FORMAT);
        const response = yield web3_rpc_methods_1.ethRpcMethods.getTransactionByHash(web3Context.requestManager, transactionHashFormatted);
        return (0, web3_validator_1.isNullish)(response)
            ? response
            : (0, format_transaction_js_1.formatTransaction)(response, returnFormat, {
                transactionSchema: web3Context.config.customTransactionSchema,
                fillInputAndData: true,
            });
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getPendingTransactions}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getPendingTransactions(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.ethRpcMethods.getPendingTransactions(web3Context.requestManager);
        return response.map(transaction => (0, format_transaction_js_1.formatTransaction)(transaction, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat, {
            transactionSchema: web3Context.config.customTransactionSchema,
            fillInputAndData: true,
        }));
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getTransactionFromBlock}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getTransactionFromBlock(web3Context_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, block = web3Context.defaultBlock, transactionIndex, returnFormat) {
        const transactionIndexFormatted = (0, web3_utils_1.format)({ format: 'uint' }, transactionIndex, web3_types_1.ETH_DATA_FORMAT);
        let response;
        if ((0, web3_validator_1.isBytes)(block)) {
            const blockHashFormatted = (0, web3_utils_1.format)({ format: 'bytes32' }, block, web3_types_1.ETH_DATA_FORMAT);
            response = yield web3_rpc_methods_1.ethRpcMethods.getTransactionByBlockHashAndIndex(web3Context.requestManager, blockHashFormatted, transactionIndexFormatted);
        }
        else {
            const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(block)
                ? block
                : (0, web3_utils_1.format)({ format: 'uint' }, block, web3_types_1.ETH_DATA_FORMAT);
            response = yield web3_rpc_methods_1.ethRpcMethods.getTransactionByBlockNumberAndIndex(web3Context.requestManager, blockNumberFormatted, transactionIndexFormatted);
        }
        return (0, web3_validator_1.isNullish)(response)
            ? response
            : (0, format_transaction_js_1.formatTransaction)(response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat, {
                transactionSchema: web3Context.config.customTransactionSchema,
                fillInputAndData: true,
            });
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getTransactionReceipt}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getTransactionReceipt(web3Context, transactionHash, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const transactionHashFormatted = (0, web3_utils_1.format)({ format: 'bytes32' }, transactionHash, web3_types_1.DEFAULT_RETURN_FORMAT);
        let response;
        try {
            response = yield web3_rpc_methods_1.ethRpcMethods.getTransactionReceipt(web3Context.requestManager, transactionHashFormatted);
        }
        catch (error) {
            // geth indexing error, we poll until transactions stopped indexing
            if (typeof error === 'object' &&
                !(0, web3_validator_1.isNullish)(error) &&
                'message' in error &&
                error.message === 'transaction indexing is in progress') {
                console.warn('Transaction indexing is in progress.');
            }
            else {
                throw error;
            }
        }
        return (0, web3_validator_1.isNullish)(response)
            ? response
            : (0, web3_utils_1.format)(schemas_js_1.transactionReceiptSchema, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getTransactionCount}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getTransactionCount(web3Context_1, address_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, address, blockNumber = web3Context.defaultBlock, returnFormat) {
        const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(blockNumber)
            ? blockNumber
            : (0, web3_utils_1.format)({ format: 'uint' }, blockNumber, web3_types_1.ETH_DATA_FORMAT);
        const response = yield web3_rpc_methods_1.ethRpcMethods.getTransactionCount(web3Context.requestManager, address, blockNumberFormatted);
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.sendTransaction}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function sendTransaction(web3Context, transactionObj, returnFormat, options = { checkRevertBeforeSending: true }, transactionMiddleware) {
    const promiEvent = new web3_core_1.Web3PromiEvent((resolve, reject) => {
        setImmediate(() => {
            (() => __awaiter(this, void 0, void 0, function* () {
                const sendTxHelper = new send_tx_helper_js_1.SendTxHelper({
                    web3Context,
                    promiEvent,
                    options,
                    returnFormat,
                });
                let transaction = Object.assign({}, transactionObj);
                if (!(0, web3_validator_1.isNullish)(transactionMiddleware)) {
                    transaction = yield transactionMiddleware.processTransaction(transaction);
                }
                let transactionFormatted = (0, format_transaction_js_1.formatTransaction)(Object.assign(Object.assign({}, transaction), { from: (0, transaction_builder_js_1.getTransactionFromOrToAttr)('from', web3Context, transaction), to: (0, transaction_builder_js_1.getTransactionFromOrToAttr)('to', web3Context, transaction) }), web3_types_1.ETH_DATA_FORMAT, {
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
                    if (web3Context.wallet && !(0, web3_validator_1.isNullish)(transactionFormatted.from)) {
                        wallet = web3Context.wallet.get(transactionFormatted.from);
                    }
                    const transactionHash = yield sendTxHelper.signAndSend({
                        wallet,
                        tx: transactionFormatted,
                    });
                    const transactionHashFormatted = (0, web3_utils_1.format)({ format: 'bytes32' }, transactionHash, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
                    sendTxHelper.emitSent(transactionFormatted);
                    sendTxHelper.emitTransactionHash(transactionHashFormatted);
                    const transactionReceipt = yield (0, wait_for_transaction_receipt_js_1.waitForTransactionReceipt)(web3Context, transactionHash, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
                    const transactionReceiptFormatted = sendTxHelper.getReceiptWithEvents((0, web3_utils_1.format)(schemas_js_1.transactionReceiptSchema, transactionReceipt, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat));
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
function sendSignedTransaction(web3Context, signedTransaction, returnFormat, options = { checkRevertBeforeSending: true }) {
    // TODO - Promise returned in function argument where a void return was expected
    // eslint-disable-next-line @typescript-eslint/no-misused-promises
    const promiEvent = new web3_core_1.Web3PromiEvent((resolve, reject) => {
        setImmediate(() => {
            (() => __awaiter(this, void 0, void 0, function* () {
                const sendTxHelper = new send_tx_helper_js_1.SendTxHelper({
                    web3Context,
                    promiEvent,
                    options,
                    returnFormat,
                });
                // Formatting signedTransaction to be send to RPC endpoint
                const signedTransactionFormattedHex = (0, web3_utils_1.format)({ format: 'bytes' }, signedTransaction, web3_types_1.ETH_DATA_FORMAT);
                const unSerializedTransaction = web3_eth_accounts_1.TransactionFactory.fromSerializedData((0, web3_utils_1.bytesToUint8Array)((0, web3_utils_1.hexToBytes)(signedTransactionFormattedHex)));
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
                    const transactionHash = yield (0, try_send_transaction_js_1.trySendTransaction)(web3Context, () => __awaiter(this, void 0, void 0, function* () {
                        return web3_rpc_methods_1.ethRpcMethods.sendRawTransaction(web3Context.requestManager, signedTransactionFormattedHex);
                    }));
                    sendTxHelper.emitSent(signedTransactionFormattedHex);
                    const transactionHashFormatted = (0, web3_utils_1.format)({ format: 'bytes32' }, transactionHash, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
                    sendTxHelper.emitTransactionHash(transactionHashFormatted);
                    const transactionReceipt = yield (0, wait_for_transaction_receipt_js_1.waitForTransactionReceipt)(web3Context, transactionHash, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
                    const transactionReceiptFormatted = sendTxHelper.getReceiptWithEvents((0, web3_utils_1.format)(schemas_js_1.transactionReceiptSchema, transactionReceipt, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat));
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
function sign(web3Context_1, message_1, addressOrIndex_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, message, addressOrIndex, returnFormat = web3Context.defaultReturnFormat) {
        var _a;
        const messageFormatted = (0, web3_utils_1.format)({ format: 'bytes' }, message, web3_types_1.DEFAULT_RETURN_FORMAT);
        if ((_a = web3Context.wallet) === null || _a === void 0 ? void 0 : _a.get(addressOrIndex)) {
            const wallet = web3Context.wallet.get(addressOrIndex);
            const signed = wallet.sign(messageFormatted);
            return (0, web3_utils_1.format)(schemas_js_1.SignatureObjectSchema, signed, returnFormat);
        }
        if (typeof addressOrIndex === 'number') {
            throw new web3_errors_1.SignatureError(message, 'RPC method "eth_sign" does not support index signatures');
        }
        const response = yield web3_rpc_methods_1.ethRpcMethods.sign(web3Context.requestManager, addressOrIndex, messageFormatted);
        return (0, web3_utils_1.format)({ format: 'bytes' }, response, returnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.signTransaction}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function signTransaction(web3Context_1, transaction_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, transaction, returnFormat = web3Context.defaultReturnFormat) {
        const response = yield web3_rpc_methods_1.ethRpcMethods.signTransaction(web3Context.requestManager, (0, format_transaction_js_1.formatTransaction)(transaction, web3_types_1.ETH_DATA_FORMAT, {
            transactionSchema: web3Context.config.customTransactionSchema,
        }));
        // Some clients only return the encoded signed transaction (e.g. Ganache)
        // while clients such as Geth return the desired SignedTransactionInfoAPI object
        return (0, web3_validator_1.isString)(response)
            ? (0, decode_signed_transaction_js_1.decodeSignedTransaction)(response, returnFormat, {
                fillInputAndData: true,
            })
            : {
                raw: (0, web3_utils_1.format)({ format: 'bytes' }, response.raw, returnFormat),
                tx: (0, format_transaction_js_1.formatTransaction)(response.tx, returnFormat, {
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
function call(web3Context_1, transaction_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, transaction, blockNumber = web3Context.defaultBlock, returnFormat = web3Context.defaultReturnFormat) {
        const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(blockNumber)
            ? blockNumber
            : (0, web3_utils_1.format)({ format: 'uint' }, blockNumber, web3_types_1.ETH_DATA_FORMAT);
        const response = yield web3_rpc_methods_1.ethRpcMethods.call(web3Context.requestManager, (0, format_transaction_js_1.formatTransaction)(transaction, web3_types_1.ETH_DATA_FORMAT, {
            transactionSchema: web3Context.config.customTransactionSchema,
        }), blockNumberFormatted);
        return (0, web3_utils_1.format)({ format: 'bytes' }, response, returnFormat);
    });
}
// TODO - Investigate whether response is padded as 1.x docs suggest
/**
 * View additional documentations here: {@link Web3Eth.estimateGas}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function estimateGas(web3Context_1, transaction_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, transaction, blockNumber = web3Context.defaultBlock, returnFormat) {
        const transactionFormatted = (0, format_transaction_js_1.formatTransaction)(transaction, web3_types_1.ETH_DATA_FORMAT, {
            transactionSchema: web3Context.config.customTransactionSchema,
        });
        const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(blockNumber)
            ? blockNumber
            : (0, web3_utils_1.format)({ format: 'uint' }, blockNumber, web3_types_1.ETH_DATA_FORMAT);
        const response = yield web3_rpc_methods_1.ethRpcMethods.estimateGas(web3Context.requestManager, transactionFormatted, blockNumberFormatted);
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
// TODO - Add input formatting to filter
/**
 * View additional documentations here: {@link Web3Eth.getPastLogs}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getLogs(web3Context, filter, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        // format type bigint or number toBlock and fromBlock to hexstring.
        let { toBlock, fromBlock } = filter;
        if (!(0, web3_validator_1.isNullish)(toBlock)) {
            if (typeof toBlock === 'number' || typeof toBlock === 'bigint') {
                toBlock = (0, web3_utils_1.numberToHex)(toBlock);
            }
        }
        if (!(0, web3_validator_1.isNullish)(fromBlock)) {
            if (typeof fromBlock === 'number' || typeof fromBlock === 'bigint') {
                fromBlock = (0, web3_utils_1.numberToHex)(fromBlock);
            }
        }
        const formattedFilter = Object.assign(Object.assign({}, filter), { fromBlock, toBlock });
        const response = yield web3_rpc_methods_1.ethRpcMethods.getLogs(web3Context.requestManager, formattedFilter);
        const result = response.map(res => {
            if (typeof res === 'string') {
                return res;
            }
            return (0, web3_utils_1.format)(schemas_js_1.logSchema, res, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
        });
        return result;
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getChainId}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getChainId(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.ethRpcMethods.getChainId(web3Context.requestManager);
        return (0, web3_utils_1.format)({ format: 'uint' }, 
        // Response is number in hex formatted string
        response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getProof}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getProof(web3Context_1, address_1, storageKeys_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, address, storageKeys, blockNumber = web3Context.defaultBlock, returnFormat) {
        const storageKeysFormatted = storageKeys.map(storageKey => (0, web3_utils_1.format)({ format: 'bytes' }, storageKey, web3_types_1.ETH_DATA_FORMAT));
        const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(blockNumber)
            ? blockNumber
            : (0, web3_utils_1.format)({ format: 'uint' }, blockNumber, web3_types_1.ETH_DATA_FORMAT);
        const response = yield web3_rpc_methods_1.ethRpcMethods.getProof(web3Context.requestManager, address, storageKeysFormatted, blockNumberFormatted);
        return (0, web3_utils_1.format)(schemas_js_1.accountSchema, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
// TODO Throwing an error with Geth, but not Infura
// TODO gasUsedRatio and reward not formatting
/**
 * View additional documentations here: {@link Web3Eth.getFeeHistory}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function getFeeHistory(web3Context_1, blockCount_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, blockCount, newestBlock = web3Context.defaultBlock, rewardPercentiles, returnFormat) {
        const blockCountFormatted = (0, web3_utils_1.format)({ format: 'uint' }, blockCount, web3_types_1.ETH_DATA_FORMAT);
        const newestBlockFormatted = (0, web3_validator_1.isBlockTag)(newestBlock)
            ? newestBlock
            : (0, web3_utils_1.format)({ format: 'uint' }, newestBlock, web3_types_1.ETH_DATA_FORMAT);
        const rewardPercentilesFormatted = (0, web3_utils_1.format)({
            type: 'array',
            items: {
                format: 'uint',
            },
        }, rewardPercentiles, constants_js_1.NUMBER_DATA_FORMAT);
        const response = yield web3_rpc_methods_1.ethRpcMethods.getFeeHistory(web3Context.requestManager, blockCountFormatted, newestBlockFormatted, rewardPercentilesFormatted);
        return (0, web3_utils_1.format)(schemas_js_1.feeHistorySchema, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.createAccessList}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function createAccessList(web3Context_1, transaction_1) {
    return __awaiter(this, arguments, void 0, function* (web3Context, transaction, blockNumber = web3Context.defaultBlock, returnFormat) {
        const blockNumberFormatted = (0, web3_validator_1.isBlockTag)(blockNumber)
            ? blockNumber
            : (0, web3_utils_1.format)({ format: 'uint' }, blockNumber, web3_types_1.ETH_DATA_FORMAT);
        const response = (yield web3_rpc_methods_1.ethRpcMethods.createAccessList(web3Context.requestManager, (0, format_transaction_js_1.formatTransaction)(transaction, web3_types_1.ETH_DATA_FORMAT, {
            transactionSchema: web3Context.config.customTransactionSchema,
        }), blockNumberFormatted));
        return (0, web3_utils_1.format)(schemas_js_1.accessListResultSchema, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.signTypedData}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 */
function signTypedData(web3Context, address, typedData, useLegacy, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.ethRpcMethods.signTypedData(web3Context.requestManager, address, typedData, useLegacy);
        return (0, web3_utils_1.format)({ format: 'bytes' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
//# sourceMappingURL=rpc_method_wrappers.js.map