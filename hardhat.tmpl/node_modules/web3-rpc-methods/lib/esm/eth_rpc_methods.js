var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
import { validator } from 'web3-validator';
export function getProtocolVersion(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_protocolVersion',
            params: [],
        });
    });
}
export function getSyncing(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_syncing',
            params: [],
        });
    });
}
export function getCoinbase(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_coinbase',
            params: [],
        });
    });
}
export function getMining(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_mining',
            params: [],
        });
    });
}
export function getHashRate(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_hashrate',
            params: [],
        });
    });
}
export function getGasPrice(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_gasPrice',
            params: [],
        });
    });
}
export function getMaxPriorityFeePerGas(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_maxPriorityFeePerGas',
            params: [],
        });
    });
}
export function getAccounts(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_accounts',
            params: [],
        });
    });
}
export function getBlockNumber(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_blockNumber',
            params: [],
        });
    });
}
export function getBalance(requestManager, address, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['address', 'blockNumberOrTag'], [address, blockNumber]);
        return requestManager.send({
            method: 'eth_getBalance',
            params: [address, blockNumber],
        });
    });
}
export function getStorageAt(requestManager, address, storageSlot, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['address', 'hex', 'blockNumberOrTag'], [address, storageSlot, blockNumber]);
        return requestManager.send({
            method: 'eth_getStorageAt',
            params: [address, storageSlot, blockNumber],
        });
    });
}
export function getTransactionCount(requestManager, address, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['address', 'blockNumberOrTag'], [address, blockNumber]);
        return requestManager.send({
            method: 'eth_getTransactionCount',
            params: [address, blockNumber],
        });
    });
}
export function getBlockTransactionCountByHash(requestManager, blockHash) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['bytes32'], [blockHash]);
        return requestManager.send({
            method: 'eth_getBlockTransactionCountByHash',
            params: [blockHash],
        });
    });
}
export function getBlockTransactionCountByNumber(requestManager, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['blockNumberOrTag'], [blockNumber]);
        return requestManager.send({
            method: 'eth_getBlockTransactionCountByNumber',
            params: [blockNumber],
        });
    });
}
export function getUncleCountByBlockHash(requestManager, blockHash) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['bytes32'], [blockHash]);
        return requestManager.send({
            method: 'eth_getUncleCountByBlockHash',
            params: [blockHash],
        });
    });
}
export function getUncleCountByBlockNumber(requestManager, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['blockNumberOrTag'], [blockNumber]);
        return requestManager.send({
            method: 'eth_getUncleCountByBlockNumber',
            params: [blockNumber],
        });
    });
}
export function getCode(requestManager, address, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['address', 'blockNumberOrTag'], [address, blockNumber]);
        return requestManager.send({
            method: 'eth_getCode',
            params: [address, blockNumber],
        });
    });
}
export function sign(requestManager, address, message) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['address', 'hex'], [address, message]);
        return requestManager.send({
            method: 'eth_sign',
            params: [address, message],
        });
    });
}
// TODO - Validation should be:
// isTransactionWithSender(transaction)
// ? validateTransactionWithSender(transaction)
// : validateTransactionWithSender(transaction, true) with true being a isPartial flag
export function signTransaction(requestManager, transaction) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_signTransaction',
            params: [transaction],
        });
    });
}
// TODO - Validation should be:
// isTransactionWithSender(transaction)
// ? validateTransactionWithSender(transaction)
// : validateTransactionWithSender(transaction, true) with true being a isPartial flag
export function sendTransaction(requestManager, transaction) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_sendTransaction',
            params: [transaction],
        });
    });
}
export function sendRawTransaction(requestManager, transaction) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['hex'], [transaction]);
        return requestManager.send({
            method: 'eth_sendRawTransaction',
            params: [transaction],
        });
    });
}
// TODO - validate transaction
export function call(requestManager, transaction, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        // validateTransactionCall(transaction);
        validator.validate(['blockNumberOrTag'], [blockNumber]);
        return requestManager.send({
            method: 'eth_call',
            params: [transaction, blockNumber],
        });
    });
}
// TODO Not sure how to best validate Partial<TransactionWithSender>
export function estimateGas(requestManager, transaction, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['blockNumberOrTag'], [blockNumber]);
        return requestManager.send({
            method: 'eth_estimateGas',
            params: [transaction, blockNumber],
        });
    });
}
export function getBlockByHash(requestManager, blockHash, hydrated) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['bytes32', 'bool'], [blockHash, hydrated]);
        return requestManager.send({
            method: 'eth_getBlockByHash',
            params: [blockHash, hydrated],
        });
    });
}
export function getBlockByNumber(requestManager, blockNumber, hydrated) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['blockNumberOrTag', 'bool'], [blockNumber, hydrated]);
        return requestManager.send({
            method: 'eth_getBlockByNumber',
            params: [blockNumber, hydrated],
        });
    });
}
export function getTransactionByHash(requestManager, transactionHash) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['bytes32'], [transactionHash]);
        return requestManager.send({
            method: 'eth_getTransactionByHash',
            params: [transactionHash],
        });
    });
}
export function getTransactionByBlockHashAndIndex(requestManager, blockHash, transactionIndex) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['bytes32', 'hex'], [blockHash, transactionIndex]);
        return requestManager.send({
            method: 'eth_getTransactionByBlockHashAndIndex',
            params: [blockHash, transactionIndex],
        });
    });
}
export function getTransactionByBlockNumberAndIndex(requestManager, blockNumber, transactionIndex) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['blockNumberOrTag', 'hex'], [blockNumber, transactionIndex]);
        return requestManager.send({
            method: 'eth_getTransactionByBlockNumberAndIndex',
            params: [blockNumber, transactionIndex],
        });
    });
}
export function getTransactionReceipt(requestManager, transactionHash) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['bytes32'], [transactionHash]);
        return requestManager.send({
            method: 'eth_getTransactionReceipt',
            params: [transactionHash],
        });
    });
}
export function getUncleByBlockHashAndIndex(requestManager, blockHash, uncleIndex) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['bytes32', 'hex'], [blockHash, uncleIndex]);
        return requestManager.send({
            method: 'eth_getUncleByBlockHashAndIndex',
            params: [blockHash, uncleIndex],
        });
    });
}
export function getUncleByBlockNumberAndIndex(requestManager, blockNumber, uncleIndex) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['blockNumberOrTag', 'hex'], [blockNumber, uncleIndex]);
        return requestManager.send({
            method: 'eth_getUncleByBlockNumberAndIndex',
            params: [blockNumber, uncleIndex],
        });
    });
}
export function getCompilers(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_getCompilers',
            params: [],
        });
    });
}
export function compileSolidity(requestManager, code) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['string'], [code]);
        return requestManager.send({
            method: 'eth_compileSolidity',
            params: [code],
        });
    });
}
export function compileLLL(requestManager, code) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['string'], [code]);
        return requestManager.send({
            method: 'eth_compileLLL',
            params: [code],
        });
    });
}
export function compileSerpent(requestManager, code) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['string'], [code]);
        return requestManager.send({
            method: 'eth_compileSerpent',
            params: [code],
        });
    });
}
export function newFilter(requestManager, filter) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['filter'], [filter]);
        return requestManager.send({
            method: 'eth_newFilter',
            params: [filter],
        });
    });
}
export function newBlockFilter(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_newBlockFilter',
            params: [],
        });
    });
}
export function newPendingTransactionFilter(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_newPendingTransactionFilter',
            params: [],
        });
    });
}
export function uninstallFilter(requestManager, filterIdentifier) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['hex'], [filterIdentifier]);
        return requestManager.send({
            method: 'eth_uninstallFilter',
            params: [filterIdentifier],
        });
    });
}
export function getFilterChanges(requestManager, filterIdentifier) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['hex'], [filterIdentifier]);
        return requestManager.send({
            method: 'eth_getFilterChanges',
            params: [filterIdentifier],
        });
    });
}
export function getFilterLogs(requestManager, filterIdentifier) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['hex'], [filterIdentifier]);
        return requestManager.send({
            method: 'eth_getFilterLogs',
            params: [filterIdentifier],
        });
    });
}
export function getLogs(requestManager, filter) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['filter'], [filter]);
        return requestManager.send({
            method: 'eth_getLogs',
            params: [filter],
        });
    });
}
export function getWork(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_getWork',
            params: [],
        });
    });
}
export function submitWork(requestManager, nonce, hash, digest) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['bytes8', 'bytes32', 'bytes32'], [nonce, hash, digest]);
        return requestManager.send({
            method: 'eth_submitWork',
            params: [nonce, hash, digest],
        });
    });
}
export function submitHashrate(requestManager, hashRate, id) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['bytes32', 'bytes32'], [hashRate, id]);
        return requestManager.send({
            method: 'eth_submitHashrate',
            params: [hashRate, id],
        });
    });
}
export function getFeeHistory(requestManager, blockCount, newestBlock, rewardPercentiles) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['hex', 'blockNumberOrTag'], [blockCount, newestBlock]);
        for (const rewardPercentile of rewardPercentiles) {
            validator.validate(['number'], [rewardPercentile]);
        }
        return requestManager.send({
            method: 'eth_feeHistory',
            params: [blockCount, newestBlock, rewardPercentiles],
        });
    });
}
export function getPendingTransactions(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_pendingTransactions',
            params: [],
        });
    });
}
export function requestAccounts(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_requestAccounts',
            params: [],
        });
    });
}
export function getChainId(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'eth_chainId',
            params: [],
        });
    });
}
export function getProof(requestManager, address, storageKeys, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['address', 'bytes32[]', 'blockNumberOrTag'], [address, storageKeys, blockNumber]);
        return requestManager.send({
            method: 'eth_getProof',
            params: [address, storageKeys, blockNumber],
        });
    });
}
export function getNodeInfo(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'web3_clientVersion',
            params: [],
        });
    });
}
export function createAccessList(requestManager, transaction, blockNumber) {
    return __awaiter(this, void 0, void 0, function* () {
        validator.validate(['blockNumberOrTag'], [blockNumber]);
        return requestManager.send({
            method: 'eth_createAccessList',
            params: [transaction, blockNumber],
        });
    });
}
export function signTypedData(requestManager, address, typedData, useLegacy = false) {
    return __awaiter(this, void 0, void 0, function* () {
        // TODO Add validation for typedData
        validator.validate(['address'], [address]);
        return requestManager.send({
            method: `eth_signTypedData${useLegacy ? '' : '_v4'}`,
            params: [address, typedData],
        });
    });
}
//# sourceMappingURL=eth_rpc_methods.js.map