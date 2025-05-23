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
import { Web3RequestManager } from 'web3-core';
import {
	Address,
	BlockNumberOrTag,
	Filter,
	HexString32Bytes,
	HexString8Bytes,
	HexStringBytes,
	TransactionCallAPI,
	TransactionWithSenderAPI,
	Uint,
	Uint256,
	Web3EthExecutionAPI,
	Eip712TypedData,
} from 'web3-types';
import { validator } from 'web3-validator';

export async function getProtocolVersion(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_protocolVersion',
		params: [],
	});
}

export async function getSyncing(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_syncing',
		params: [],
	});
}

export async function getCoinbase(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_coinbase',
		params: [],
	});
}

export async function getMining(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_mining',
		params: [],
	});
}

export async function getHashRate(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_hashrate',
		params: [],
	});
}

export async function getGasPrice(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_gasPrice',
		params: [],
	});
}

export async function getMaxPriorityFeePerGas(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_maxPriorityFeePerGas',
		params: [],
	});
}

export async function getAccounts(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_accounts',
		params: [],
	});
}

export async function getBlockNumber(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_blockNumber',
		params: [],
	});
}

export async function getBalance(
	requestManager: Web3RequestManager,
	address: Address,
	blockNumber: BlockNumberOrTag,
) {
	validator.validate(['address', 'blockNumberOrTag'], [address, blockNumber]);

	return requestManager.send({
		method: 'eth_getBalance',
		params: [address, blockNumber],
	});
}

export async function getStorageAt(
	requestManager: Web3RequestManager,
	address: Address,
	storageSlot: Uint256,
	blockNumber: BlockNumberOrTag,
) {
	validator.validate(['address', 'hex', 'blockNumberOrTag'], [address, storageSlot, blockNumber]);

	return requestManager.send({
		method: 'eth_getStorageAt',
		params: [address, storageSlot, blockNumber],
	});
}

export async function getTransactionCount(
	requestManager: Web3RequestManager,
	address: Address,
	blockNumber: BlockNumberOrTag,
) {
	validator.validate(['address', 'blockNumberOrTag'], [address, blockNumber]);

	return requestManager.send({
		method: 'eth_getTransactionCount',
		params: [address, blockNumber],
	});
}

export async function getBlockTransactionCountByHash(
	requestManager: Web3RequestManager,
	blockHash: HexString32Bytes,
) {
	validator.validate(['bytes32'], [blockHash]);

	return requestManager.send({
		method: 'eth_getBlockTransactionCountByHash',
		params: [blockHash],
	});
}

export async function getBlockTransactionCountByNumber(
	requestManager: Web3RequestManager,
	blockNumber: BlockNumberOrTag,
) {
	validator.validate(['blockNumberOrTag'], [blockNumber]);

	return requestManager.send({
		method: 'eth_getBlockTransactionCountByNumber',
		params: [blockNumber],
	});
}

export async function getUncleCountByBlockHash(
	requestManager: Web3RequestManager,
	blockHash: HexString32Bytes,
) {
	validator.validate(['bytes32'], [blockHash]);

	return requestManager.send({
		method: 'eth_getUncleCountByBlockHash',
		params: [blockHash],
	});
}

export async function getUncleCountByBlockNumber(
	requestManager: Web3RequestManager,
	blockNumber: BlockNumberOrTag,
) {
	validator.validate(['blockNumberOrTag'], [blockNumber]);

	return requestManager.send({
		method: 'eth_getUncleCountByBlockNumber',
		params: [blockNumber],
	});
}

export async function getCode(
	requestManager: Web3RequestManager,
	address: Address,
	blockNumber: BlockNumberOrTag,
) {
	validator.validate(['address', 'blockNumberOrTag'], [address, blockNumber]);

	return requestManager.send({
		method: 'eth_getCode',
		params: [address, blockNumber],
	});
}

export async function sign(
	requestManager: Web3RequestManager,
	address: Address,
	message: HexStringBytes,
) {
	validator.validate(['address', 'hex'], [address, message]);

	return requestManager.send({
		method: 'eth_sign',
		params: [address, message],
	});
}

// TODO - Validation should be:
// isTransactionWithSender(transaction)
// ? validateTransactionWithSender(transaction)
// : validateTransactionWithSender(transaction, true) with true being a isPartial flag
export async function signTransaction(
	requestManager: Web3RequestManager,
	transaction: TransactionWithSenderAPI | Partial<TransactionWithSenderAPI>,
) {
	return requestManager.send({
		method: 'eth_signTransaction',
		params: [transaction],
	});
}

// TODO - Validation should be:
// isTransactionWithSender(transaction)
// ? validateTransactionWithSender(transaction)
// : validateTransactionWithSender(transaction, true) with true being a isPartial flag
export async function sendTransaction(
	requestManager: Web3RequestManager,
	transaction: TransactionWithSenderAPI | Partial<TransactionWithSenderAPI>,
) {
	return requestManager.send({
		method: 'eth_sendTransaction',
		params: [transaction],
	});
}

export async function sendRawTransaction(
	requestManager: Web3RequestManager,
	transaction: HexStringBytes,
) {
	validator.validate(['hex'], [transaction]);

	return requestManager.send({
		method: 'eth_sendRawTransaction',
		params: [transaction],
	});
}

// TODO - validate transaction
export async function call(
	requestManager: Web3RequestManager,
	transaction: TransactionCallAPI,
	blockNumber: BlockNumberOrTag,
) {
	// validateTransactionCall(transaction);
	validator.validate(['blockNumberOrTag'], [blockNumber]);

	return requestManager.send({
		method: 'eth_call',
		params: [transaction, blockNumber],
	});
}

// TODO Not sure how to best validate Partial<TransactionWithSender>
export async function estimateGas<TransactionType = TransactionWithSenderAPI>(
	requestManager: Web3RequestManager,
	transaction: Partial<TransactionType>,
	blockNumber: BlockNumberOrTag,
) {
	validator.validate(['blockNumberOrTag'], [blockNumber]);

	return requestManager.send({
		method: 'eth_estimateGas',
		params: [transaction, blockNumber],
	});
}

export async function getBlockByHash(
	requestManager: Web3RequestManager,
	blockHash: HexString32Bytes,
	hydrated: boolean,
) {
	validator.validate(['bytes32', 'bool'], [blockHash, hydrated]);

	return requestManager.send({
		method: 'eth_getBlockByHash',
		params: [blockHash, hydrated],
	});
}

export async function getBlockByNumber(
	requestManager: Web3RequestManager,
	blockNumber: BlockNumberOrTag,
	hydrated: boolean,
) {
	validator.validate(['blockNumberOrTag', 'bool'], [blockNumber, hydrated]);

	return requestManager.send({
		method: 'eth_getBlockByNumber',
		params: [blockNumber, hydrated],
	});
}

export async function getTransactionByHash(
	requestManager: Web3RequestManager,
	transactionHash: HexString32Bytes,
) {
	validator.validate(['bytes32'], [transactionHash]);

	return requestManager.send({
		method: 'eth_getTransactionByHash',
		params: [transactionHash],
	});
}

export async function getTransactionByBlockHashAndIndex(
	requestManager: Web3RequestManager,
	blockHash: HexString32Bytes,
	transactionIndex: Uint,
) {
	validator.validate(['bytes32', 'hex'], [blockHash, transactionIndex]);

	return requestManager.send({
		method: 'eth_getTransactionByBlockHashAndIndex',
		params: [blockHash, transactionIndex],
	});
}

export async function getTransactionByBlockNumberAndIndex(
	requestManager: Web3RequestManager,
	blockNumber: BlockNumberOrTag,
	transactionIndex: Uint,
) {
	validator.validate(['blockNumberOrTag', 'hex'], [blockNumber, transactionIndex]);

	return requestManager.send({
		method: 'eth_getTransactionByBlockNumberAndIndex',
		params: [blockNumber, transactionIndex],
	});
}

export async function getTransactionReceipt(
	requestManager: Web3RequestManager,
	transactionHash: HexString32Bytes,
) {
	validator.validate(['bytes32'], [transactionHash]);

	return requestManager.send({
		method: 'eth_getTransactionReceipt',
		params: [transactionHash],
	});
}

export async function getUncleByBlockHashAndIndex(
	requestManager: Web3RequestManager,
	blockHash: HexString32Bytes,
	uncleIndex: Uint,
) {
	validator.validate(['bytes32', 'hex'], [blockHash, uncleIndex]);

	return requestManager.send({
		method: 'eth_getUncleByBlockHashAndIndex',
		params: [blockHash, uncleIndex],
	});
}

export async function getUncleByBlockNumberAndIndex(
	requestManager: Web3RequestManager,
	blockNumber: BlockNumberOrTag,
	uncleIndex: Uint,
) {
	validator.validate(['blockNumberOrTag', 'hex'], [blockNumber, uncleIndex]);

	return requestManager.send({
		method: 'eth_getUncleByBlockNumberAndIndex',
		params: [blockNumber, uncleIndex],
	});
}

export async function getCompilers(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_getCompilers',
		params: [],
	});
}

export async function compileSolidity(requestManager: Web3RequestManager, code: string) {
	validator.validate(['string'], [code]);

	return requestManager.send({
		method: 'eth_compileSolidity',
		params: [code],
	});
}

export async function compileLLL(requestManager: Web3RequestManager, code: string) {
	validator.validate(['string'], [code]);

	return requestManager.send({
		method: 'eth_compileLLL',
		params: [code],
	});
}

export async function compileSerpent(requestManager: Web3RequestManager, code: string) {
	validator.validate(['string'], [code]);

	return requestManager.send({
		method: 'eth_compileSerpent',
		params: [code],
	});
}

export async function newFilter(requestManager: Web3RequestManager, filter: Filter) {
	validator.validate(['filter'], [filter]);

	return requestManager.send({
		method: 'eth_newFilter',
		params: [filter],
	});
}

export async function newBlockFilter(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_newBlockFilter',
		params: [],
	});
}

export async function newPendingTransactionFilter(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_newPendingTransactionFilter',
		params: [],
	});
}

export async function uninstallFilter(requestManager: Web3RequestManager, filterIdentifier: Uint) {
	validator.validate(['hex'], [filterIdentifier]);

	return requestManager.send({
		method: 'eth_uninstallFilter',
		params: [filterIdentifier],
	});
}

export async function getFilterChanges(requestManager: Web3RequestManager, filterIdentifier: Uint) {
	validator.validate(['hex'], [filterIdentifier]);

	return requestManager.send({
		method: 'eth_getFilterChanges',
		params: [filterIdentifier],
	});
}

export async function getFilterLogs(requestManager: Web3RequestManager, filterIdentifier: Uint) {
	validator.validate(['hex'], [filterIdentifier]);

	return requestManager.send({
		method: 'eth_getFilterLogs',
		params: [filterIdentifier],
	});
}

export async function getLogs(requestManager: Web3RequestManager, filter: Filter) {
	validator.validate(['filter'], [filter]);

	return requestManager.send({
		method: 'eth_getLogs',
		params: [filter],
	});
}

export async function getWork(requestManager: Web3RequestManager) {
	return requestManager.send({
		method: 'eth_getWork',
		params: [],
	});
}

export async function submitWork(
	requestManager: Web3RequestManager,
	nonce: HexString8Bytes,
	hash: HexString32Bytes,
	digest: HexString32Bytes,
) {
	validator.validate(['bytes8', 'bytes32', 'bytes32'], [nonce, hash, digest]);

	return requestManager.send({
		method: 'eth_submitWork',
		params: [nonce, hash, digest],
	});
}

export async function submitHashrate(
	requestManager: Web3RequestManager,
	hashRate: HexString32Bytes,
	id: HexString32Bytes,
) {
	validator.validate(['bytes32', 'bytes32'], [hashRate, id]);

	return requestManager.send({
		method: 'eth_submitHashrate',
		params: [hashRate, id],
	});
}

export async function getFeeHistory(
	requestManager: Web3RequestManager,
	blockCount: Uint,
	newestBlock: BlockNumberOrTag,
	rewardPercentiles: number[],
) {
	validator.validate(['hex', 'blockNumberOrTag'], [blockCount, newestBlock]);

	for (const rewardPercentile of rewardPercentiles) {
		validator.validate(['number'], [rewardPercentile]);
	}

	return requestManager.send({
		method: 'eth_feeHistory',
		params: [blockCount, newestBlock, rewardPercentiles],
	});
}

export async function getPendingTransactions(
	requestManager: Web3RequestManager<Web3EthExecutionAPI>,
) {
	return requestManager.send({
		method: 'eth_pendingTransactions',
		params: [],
	});
}

export async function requestAccounts(requestManager: Web3RequestManager<Web3EthExecutionAPI>) {
	return requestManager.send({
		method: 'eth_requestAccounts',
		params: [],
	});
}

export async function getChainId(requestManager: Web3RequestManager<Web3EthExecutionAPI>) {
	return requestManager.send({
		method: 'eth_chainId',
		params: [],
	});
}

export async function getProof(
	requestManager: Web3RequestManager<Web3EthExecutionAPI>,
	address: Address,
	storageKeys: HexString32Bytes[],
	blockNumber: BlockNumberOrTag,
) {
	validator.validate(
		['address', 'bytes32[]', 'blockNumberOrTag'],
		[address, storageKeys, blockNumber],
	);

	return requestManager.send({
		method: 'eth_getProof',
		params: [address, storageKeys, blockNumber],
	});
}

export async function getNodeInfo(requestManager: Web3RequestManager<Web3EthExecutionAPI>) {
	return requestManager.send({
		method: 'web3_clientVersion',
		params: [],
	});
}

export async function createAccessList(
	requestManager: Web3RequestManager,
	transaction: TransactionWithSenderAPI | Partial<TransactionWithSenderAPI>,
	blockNumber: BlockNumberOrTag,
) {
	validator.validate(['blockNumberOrTag'], [blockNumber]);

	return requestManager.send({
		method: 'eth_createAccessList',
		params: [transaction, blockNumber],
	});
}

export async function signTypedData(
	requestManager: Web3RequestManager,
	address: Address,
	typedData: Eip712TypedData,
	useLegacy = false,
): Promise<string> {
	// TODO Add validation for typedData
	validator.validate(['address'], [address]);

	return requestManager.send({
		method: `eth_signTypedData${useLegacy ? '' : '_v4'}`,
		params: [address, typedData],
	});
}
