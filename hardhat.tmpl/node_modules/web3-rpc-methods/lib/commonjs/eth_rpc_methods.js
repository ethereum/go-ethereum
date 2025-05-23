"use strict";
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
exports.getProof = exports.getChainId = exports.requestAccounts = exports.getPendingTransactions = exports.getFeeHistory = exports.submitHashrate = exports.submitWork = exports.getWork = exports.getLogs = exports.getFilterLogs = exports.getFilterChanges = exports.uninstallFilter = exports.newPendingTransactionFilter = exports.newBlockFilter = exports.newFilter = exports.compileSerpent = exports.compileLLL = exports.compileSolidity = exports.getCompilers = exports.getUncleByBlockNumberAndIndex = exports.getUncleByBlockHashAndIndex = exports.getTransactionReceipt = exports.getTransactionByBlockNumberAndIndex = exports.getTransactionByBlockHashAndIndex = exports.getTransactionByHash = exports.getBlockByNumber = exports.getBlockByHash = exports.estimateGas = exports.call = exports.sendRawTransaction = exports.sendTransaction = exports.signTransaction = exports.sign = exports.getCode = exports.getUncleCountByBlockNumber = exports.getUncleCountByBlockHash = exports.getBlockTransactionCountByNumber = exports.getBlockTransactionCountByHash = exports.getTransactionCount = exports.getStorageAt = exports.getBalance = exports.getBlockNumber = exports.getAccounts = exports.getMaxPriorityFeePerGas = exports.getGasPrice = exports.getHashRate = exports.getMining = exports.getCoinbase = exports.getSyncing = exports.getProtocolVersion = void 0;
exports.signTypedData = exports.createAccessList = exports.getNodeInfo = void 0;
const web3_validator_1 = require("web3-validator");
function getProtocolVersion(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_protocolVersion',
            params: [],
        });
    });
}
exports.getProtocolVersion = getProtocolVersion;
function getSyncing(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_syncing',
            params: [],
        });
    });
}
exports.getSyncing = getSyncing;
function getCoinbase(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_coinbase',
            params: [],
        });
    });
}
exports.getCoinbase = getCoinbase;
function getMining(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_mining',
            params: [],
        });
    });
}
exports.getMining = getMining;
function getHashRate(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_hashrate',
            params: [],
        });
    });
}
exports.getHashRate = getHashRate;
function getGasPrice(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_gasPrice',
            params: [],
        });
    });
}
exports.getGasPrice = getGasPrice;
function getMaxPriorityFeePerGas(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_maxPriorityFeePerGas',
            params: [],
        });
    });
}
exports.getMaxPriorityFeePerGas = getMaxPriorityFeePerGas;
function getAccounts(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_accounts',
            params: [],
        });
    });
}
exports.getAccounts = getAccounts;
function getBlockNumber(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_blockNumber',
            params: [],
        });
    });
}
exports.getBlockNumber = getBlockNumber;
function getBalance(requestManager, address, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['address', 'blockNumberOrTag'], [address, blockNumber]);
        return requestManager.send({
            method: 'eth_getBalance',
            params: [address, blockNumber],
        });
    });
}
exports.getBalance = getBalance;
function getStorageAt(requestManager, address, storageSlot, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['address', 'hex', 'blockNumberOrTag'], [address, storageSlot, blockNumber]);
        return requestManager.send({
            method: 'eth_getStorageAt',
            params: [address, storageSlot, blockNumber],
        });
    });
}
exports.getStorageAt = getStorageAt;
function getTransactionCount(requestManager, address, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['address', 'blockNumberOrTag'], [address, blockNumber]);
        return requestManager.send({
            method: 'eth_getTransactionCount',
            params: [address, blockNumber],
        });
    });
}
exports.getTransactionCount = getTransactionCount;
function getBlockTransactionCountByHash(requestManager, blockHash) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['bytes32'], [blockHash]);
        return requestManager.send({
            method: 'eth_getBlockTransactionCountByHash',
            params: [blockHash],
        });
    });
}
exports.getBlockTransactionCountByHash = getBlockTransactionCountByHash;
function getBlockTransactionCountByNumber(requestManager, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['blockNumberOrTag'], [blockNumber]);
        return requestManager.send({
            method: 'eth_getBlockTransactionCountByNumber',
            params: [blockNumber],
        });
    });
}
exports.getBlockTransactionCountByNumber = getBlockTransactionCountByNumber;
function getUncleCountByBlockHash(requestManager, blockHash) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['bytes32'], [blockHash]);
        return requestManager.send({
            method: 'eth_getUncleCountByBlockHash',
            params: [blockHash],
        });
    });
}
exports.getUncleCountByBlockHash = getUncleCountByBlockHash;
function getUncleCountByBlockNumber(requestManager, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['blockNumberOrTag'], [blockNumber]);
        return requestManager.send({
            method: 'eth_getUncleCountByBlockNumber',
            params: [blockNumber],
        });
    });
}
exports.getUncleCountByBlockNumber = getUncleCountByBlockNumber;
function getCode(requestManager, address, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['address', 'blockNumberOrTag'], [address, blockNumber]);
        return requestManager.send({
            method: 'eth_getCode',
            params: [address, blockNumber],
        });
    });
}
exports.getCode = getCode;
function sign(requestManager, address, message) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['address', 'hex'], [address, message]);
        return requestManager.send({
            method: 'eth_sign',
            params: [address, message],
        });
    });
}
exports.sign = sign;
// TODO - Validation should be:
// isTransactionWithSender(transaction)
// ? validateTransactionWithSender(transaction)
// : validateTransactionWithSender(transaction, true) with true being a isPartial flag
function signTransaction(requestManager, transaction) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_signTransaction',
            params: [transaction],
        });
    });
}
exports.signTransaction = signTransaction;
// TODO - Validation should be:
// isTransactionWithSender(transaction)
// ? validateTransactionWithSender(transaction)
// : validateTransactionWithSender(transaction, true) with true being a isPartial flag
function sendTransaction(requestManager, transaction) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_sendTransaction',
            params: [transaction],
        });
    });
}
exports.sendTransaction = sendTransaction;
function sendRawTransaction(requestManager, transaction) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['hex'], [transaction]);
        return requestManager.send({
            method: 'eth_sendRawTransaction',
            params: [transaction],
        });
    });
}
exports.sendRawTransaction = sendRawTransaction;
// TODO - validate transaction
function call(requestManager, transaction, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        // validateTransactionCall(transaction);
        web3_validator_1.validator.validate(['blockNumberOrTag'], [blockNumber]);
        return requestManager.send({
            method: 'eth_call',
            params: [transaction, blockNumber],
        });
    });
}
exports.call = call;
// TODO Not sure how to best validate Partial<TransactionWithSender>
function estimateGas(requestManager, transaction, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['blockNumberOrTag'], [blockNumber]);
        return requestManager.send({
            method: 'eth_estimateGas',
            params: [transaction, blockNumber],
        });
    });
}
exports.estimateGas = estimateGas;
function getBlockByHash(requestManager, blockHash, hydrated) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['bytes32', 'bool'], [blockHash, hydrated]);
        return requestManager.send({
            method: 'eth_getBlockByHash',
            params: [blockHash, hydrated],
        });
    });
}
exports.getBlockByHash = getBlockByHash;
function getBlockByNumber(requestManager, blockNumber, hydrated) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['blockNumberOrTag', 'bool'], [blockNumber, hydrated]);
        return requestManager.send({
            method: 'eth_getBlockByNumber',
            params: [blockNumber, hydrated],
        });
    });
}
exports.getBlockByNumber = getBlockByNumber;
function getTransactionByHash(requestManager, transactionHash) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['bytes32'], [transactionHash]);
        return requestManager.send({
            method: 'eth_getTransactionByHash',
            params: [transactionHash],
        });
    });
}
exports.getTransactionByHash = getTransactionByHash;
function getTransactionByBlockHashAndIndex(requestManager, blockHash, transactionIndex) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['bytes32', 'hex'], [blockHash, transactionIndex]);
        return requestManager.send({
            method: 'eth_getTransactionByBlockHashAndIndex',
            params: [blockHash, transactionIndex],
        });
    });
}
exports.getTransactionByBlockHashAndIndex = getTransactionByBlockHashAndIndex;
function getTransactionByBlockNumberAndIndex(requestManager, blockNumber, transactionIndex) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['blockNumberOrTag', 'hex'], [blockNumber, transactionIndex]);
        return requestManager.send({
            method: 'eth_getTransactionByBlockNumberAndIndex',
            params: [blockNumber, transactionIndex],
        });
    });
}
exports.getTransactionByBlockNumberAndIndex = getTransactionByBlockNumberAndIndex;
function getTransactionReceipt(requestManager, transactionHash) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['bytes32'], [transactionHash]);
        return requestManager.send({
            method: 'eth_getTransactionReceipt',
            params: [transactionHash],
        });
    });
}
exports.getTransactionReceipt = getTransactionReceipt;
function getUncleByBlockHashAndIndex(requestManager, blockHash, uncleIndex) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['bytes32', 'hex'], [blockHash, uncleIndex]);
        return requestManager.send({
            method: 'eth_getUncleByBlockHashAndIndex',
            params: [blockHash, uncleIndex],
        });
    });
}
exports.getUncleByBlockHashAndIndex = getUncleByBlockHashAndIndex;
function getUncleByBlockNumberAndIndex(requestManager, blockNumber, uncleIndex) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['blockNumberOrTag', 'hex'], [blockNumber, uncleIndex]);
        return requestManager.send({
            method: 'eth_getUncleByBlockNumberAndIndex',
            params: [blockNumber, uncleIndex],
        });
    });
}
exports.getUncleByBlockNumberAndIndex = getUncleByBlockNumberAndIndex;
function getCompilers(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_getCompilers',
            params: [],
        });
    });
}
exports.getCompilers = getCompilers;
function compileSolidity(requestManager, code) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['string'], [code]);
        return requestManager.send({
            method: 'eth_compileSolidity',
            params: [code],
        });
    });
}
exports.compileSolidity = compileSolidity;
function compileLLL(requestManager, code) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['string'], [code]);
        return requestManager.send({
            method: 'eth_compileLLL',
            params: [code],
        });
    });
}
exports.compileLLL = compileLLL;
function compileSerpent(requestManager, code) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['string'], [code]);
        return requestManager.send({
            method: 'eth_compileSerpent',
            params: [code],
        });
    });
}
exports.compileSerpent = compileSerpent;
function newFilter(requestManager, filter) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['filter'], [filter]);
        return requestManager.send({
            method: 'eth_newFilter',
            params: [filter],
        });
    });
}
exports.newFilter = newFilter;
function newBlockFilter(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_newBlockFilter',
            params: [],
        });
    });
}
exports.newBlockFilter = newBlockFilter;
function newPendingTransactionFilter(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_newPendingTransactionFilter',
            params: [],
        });
    });
}
exports.newPendingTransactionFilter = newPendingTransactionFilter;
function uninstallFilter(requestManager, filterIdentifier) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['hex'], [filterIdentifier]);
        return requestManager.send({
            method: 'eth_uninstallFilter',
            params: [filterIdentifier],
        });
    });
}
exports.uninstallFilter = uninstallFilter;
function getFilterChanges(requestManager, filterIdentifier) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['hex'], [filterIdentifier]);
        return requestManager.send({
            method: 'eth_getFilterChanges',
            params: [filterIdentifier],
        });
    });
}
exports.getFilterChanges = getFilterChanges;
function getFilterLogs(requestManager, filterIdentifier) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['hex'], [filterIdentifier]);
        return requestManager.send({
            method: 'eth_getFilterLogs',
            params: [filterIdentifier],
        });
    });
}
exports.getFilterLogs = getFilterLogs;
function getLogs(requestManager, filter) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['filter'], [filter]);
        return requestManager.send({
            method: 'eth_getLogs',
            params: [filter],
        });
    });
}
exports.getLogs = getLogs;
function getWork(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_getWork',
            params: [],
        });
    });
}
exports.getWork = getWork;
function submitWork(requestManager, nonce, hash, digest) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['bytes8', 'bytes32', 'bytes32'], [nonce, hash, digest]);
        return requestManager.send({
            method: 'eth_submitWork',
            params: [nonce, hash, digest],
        });
    });
}
exports.submitWork = submitWork;
function submitHashrate(requestManager, hashRate, id) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['bytes32', 'bytes32'], [hashRate, id]);
        return requestManager.send({
            method: 'eth_submitHashrate',
            params: [hashRate, id],
        });
    });
}
exports.submitHashrate = submitHashrate;
function getFeeHistory(requestManager, blockCount, newestBlock, rewardPercentiles) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['hex', 'blockNumberOrTag'], [blockCount, newestBlock]);
        for (const rewardPercentile of rewardPercentiles) {
            web3_validator_1.validator.validate(['number'], [rewardPercentile]);
        }
        return requestManager.send({
            method: 'eth_feeHistory',
            params: [blockCount, newestBlock, rewardPercentiles],
        });
    });
}
exports.getFeeHistory = getFeeHistory;
function getPendingTransactions(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_pendingTransactions',
            params: [],
        });
    });
}
exports.getPendingTransactions = getPendingTransactions;
function requestAccounts(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_requestAccounts',
            params: [],
        });
    });
}
exports.requestAccounts = requestAccounts;
function getChainId(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_chainId',
            params: [],
        });
    });
}
exports.getChainId = getChainId;
function getProof(requestManager, address, storageKeys, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['address', 'bytes32[]', 'blockNumberOrTag'], [address, storageKeys, blockNumber]);
        return requestManager.send({
            method: 'eth_getProof',
            params: [address, storageKeys, blockNumber],
        });
    });
}
exports.getProof = getProof;
function getNodeInfo(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'web3_clientVersion',
            params: [],
        });
    });
}
exports.getNodeInfo = getNodeInfo;
function createAccessList(requestManager, transaction, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        web3_validator_1.validator.validate(['blockNumberOrTag'], [blockNumber]);
        return requestManager.send({
            method: 'eth_createAccessList',
            params: [transaction, blockNumber],
        });
    });
}
exports.createAccessList = createAccessList;
function signTypedData(requestManager, address, typedData, useLegacy = false) {
    return __awaiter(this, void 0, void 0, function* () {
        // TODO Add validation for typedData
        web3_validator_1.validator.validate(['address'], [address]);
        return requestManager.send({
            method: `eth_signTypedData${useLegacy ? '' : '_v4'}`,
            params: [address, typedData],
        });
    });
}
exports.signTypedData = signTypedData;
//# sourceMappingURL=eth_rpc_methods.js.map