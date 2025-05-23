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
import {
	Address,
	HexString32Bytes,
	Uint,
	HexStringBytes,
	HexStringSingleByte,
	HexString256Bytes,
	FeeHistoryBase,
	HexString8Bytes,
	Uint256,
	BlockNumberOrTag,
	Filter,
	AccessList,
	TransactionHash,
	TransactionReceiptBase,
	BlockBase,
	LogBase,
} from '../eth_types.js';
import { HexString } from '../primitives_types.js';

// The types are generated manually by referring to following doc
// https://github.com/ethereum/execution-apis
// These types follow closely to the v1.0.0-alpha.9 Ethereum spec
export interface TransactionCallAPI {
	readonly from?: Address;
	readonly to: Address;
	readonly gas?: Uint;
	readonly gasPrice?: Uint;
	readonly value?: Uint;
	readonly data?: HexStringBytes;
	readonly type?: HexStringSingleByte;
	readonly maxFeePerGas?: Uint;
	readonly maxPriorityFeePerGas?: Uint;
	readonly accessList?: AccessList;
}

export interface BaseTransactionAPI {
	// eslint-disable-next-line @typescript-eslint/ban-types
	readonly to?: Address | null;
	readonly type: HexStringSingleByte;
	readonly nonce: Uint;
	readonly gas: Uint;
	readonly value: Uint;
	// TODO - https://github.com/ethereum/execution-apis/pull/201
	readonly input: HexStringBytes;
	readonly data?: HexStringBytes;
	readonly chainId?: Uint;
	readonly hash?: HexString32Bytes;
}

export interface Transaction1559UnsignedAPI extends BaseTransactionAPI {
	readonly maxFeePerGas: Uint;
	readonly maxPriorityFeePerGas: Uint;
	readonly accessList: AccessList;
	readonly gasPrice: Uint;
}

export interface Transaction1559SignedAPI extends Transaction1559UnsignedAPI {
	readonly yParity: Uint;
	readonly r: Uint;
	readonly s: Uint;
	readonly v?: never;
}

export interface Transaction2930UnsignedAPI extends BaseTransactionAPI {
	readonly gasPrice: Uint;
	readonly accessList: AccessList;
	readonly maxFeePerGas?: never;
	readonly maxPriorityFeePerGas?: never;
}

export interface Transaction2930SignedAPI extends Transaction2930UnsignedAPI {
	readonly yParity: Uint;
	readonly r: Uint;
	readonly s: Uint;
	readonly v?: never;
}

export interface TransactionLegacyUnsignedAPI extends BaseTransactionAPI {
	readonly gasPrice: Uint;
	readonly accessList?: never;
	readonly maxFeePerGas?: never;
	readonly maxPriorityFeePerGas?: never;
}

export interface TransactionLegacySignedAPI extends TransactionLegacyUnsignedAPI {
	readonly v: Uint;
	readonly r: Uint;
	readonly s: Uint;
}

// https://github.com/ethereum/execution-apis/blob/main/src/schemas/transaction.yaml#L144
export type TransactionUnsignedAPI =
	| Transaction1559UnsignedAPI
	| Transaction2930UnsignedAPI
	| TransactionLegacyUnsignedAPI;

// https://github.com/ethereum/execution-apis/blob/main/src/schemas/transaction.yaml#L211
export type TransactionSignedAPI =
	| Transaction1559SignedAPI
	| Transaction2930SignedAPI
	| TransactionLegacySignedAPI;

// https://github.com/ethereum/execution-apis/blob/main/src/schemas/transaction.yaml#L216
export type TransactionInfoAPI = TransactionSignedAPI & {
	readonly blockHash?: HexString32Bytes;
	readonly blockNumber?: Uint;
	readonly from: Address;
	readonly hash: HexString32Bytes;
	readonly transactionIndex?: Uint;
};

export interface SignedTransactionInfoAPI {
	raw: HexStringBytes;
	tx: TransactionSignedAPI;
}

// https://github.com/ethereum/execution-apis/blob/main/src/schemas/transaction.yaml#L244
export type TransactionWithSenderAPI = TransactionUnsignedAPI & { from: Address };

// https://github.com/ethereum/execution-apis/blob/main/src/schemas/block.yaml#L2
export type BlockAPI = BlockBase<
	HexString32Bytes,
	HexString,
	Uint,
	HexStringBytes,
	TransactionHash[] | TransactionInfoAPI[],
	HexString256Bytes
>;

// https://github.com/ethereum/execution-apis/blob/main/src/schemas/receipt.yaml#L2
export type LogAPI = LogBase<Uint, HexString32Bytes>;

// https://github.com/ethereum/execution-apis/blob/main/src/schemas/receipt.yaml#L36
export type TransactionReceiptAPI = TransactionReceiptBase<
	Uint,
	HexString32Bytes,
	HexString256Bytes,
	LogAPI
>;

// https://github.com/ethereum/execution-apis/blob/main/src/schemas/client.yaml#L2
export type SyncingStatusAPI =
	| { startingBlock: Uint; currentBlock: Uint; highestBlock: Uint }
	| boolean;

// https://github.com/ethereum/execution-apis/blob/main/src/eth/fee_market.yaml#L42
export type FeeHistoryResultAPI = FeeHistoryBase<Uint>;

// https://github.com/ethereum/execution-apis/blob/main/src/schemas/filter.yaml#L2
export type FilterResultsAPI = HexString32Bytes[] | LogAPI[];

export interface CompileResultAPI {
	readonly code: HexStringBytes;
	readonly info: {
		readonly source: string;
		readonly language: string;
		readonly languageVersion: string;
		readonly compilerVersion: string;
		readonly abiDefinition: Record<string, unknown>[];
		readonly userDoc: {
			readonly methods: Record<string, unknown>;
		};
		readonly developerDoc: {
			readonly methods: Record<string, unknown>;
		};
	};
}

/* eslint-disable camelcase */
export type EthExecutionAPI = {
	// https://github.com/ethereum/execution-apis/blob/main/src/eth/block.yaml
	eth_getBlockByHash: (blockHash: HexString32Bytes, hydrated: boolean) => BlockAPI;
	eth_getBlockByNumber: (blockNumber: BlockNumberOrTag, hydrated: boolean) => BlockAPI;
	eth_getBlockTransactionCountByHash: (blockHash: HexString32Bytes) => Uint;
	eth_getBlockTransactionCountByNumber: (blockNumber: BlockNumberOrTag) => Uint;
	eth_getUncleCountByBlockHash: (blockHash: HexString32Bytes) => Uint;
	eth_getUncleCountByBlockNumber: (blockNumber: BlockNumberOrTag) => Uint;
	eth_getUncleByBlockHashAndIndex: (blockHash: HexString32Bytes, uncleIndex: Uint) => BlockAPI;
	eth_getUncleByBlockNumberAndIndex: (
		blockNumber: BlockNumberOrTag,
		uncleIndex: Uint,
	) => BlockAPI;

	// https://github.com/ethereum/execution-apis/blob/main/src/eth/transaction.yaml
	eth_getTransactionByHash: (transactionHash: HexString32Bytes) => TransactionInfoAPI | undefined;
	eth_getTransactionByBlockHashAndIndex: (
		blockHash: HexString32Bytes,
		transactionIndex: Uint,
	) => TransactionInfoAPI | undefined;
	eth_getTransactionByBlockNumberAndIndex: (
		blockNumber: BlockNumberOrTag,
		transactionIndex: Uint,
	) => TransactionInfoAPI | undefined;
	eth_getTransactionReceipt: (
		transactionHash: HexString32Bytes,
	) => TransactionReceiptAPI | undefined;

	// https://github.com/ethereum/execution-apis/blob/main/src/eth/client.yaml
	eth_protocolVersion: () => string;
	eth_syncing: () => SyncingStatusAPI;
	eth_coinbase: () => Address;
	eth_accounts: () => Address[];
	eth_blockNumber: () => Uint;

	// https://github.com/ethereum/execution-apis/blob/main/src/eth/execute.yaml
	eth_call: (transaction: TransactionCallAPI, blockNumber: BlockNumberOrTag) => HexStringBytes;
	eth_estimateGas: (
		transaction: Partial<TransactionWithSenderAPI>,
		blockNumber: BlockNumberOrTag,
	) => Uint;

	// https://github.com/ethereum/execution-apis/blob/main/src/eth/fee_market.yaml
	eth_gasPrice: () => Uint;
	eth_feeHistory: (
		blockCount: Uint,
		newestBlock: BlockNumberOrTag,
		rewardPercentiles: number[],
	) => FeeHistoryResultAPI;
	eth_maxPriorityFeePerGas: () => Uint;

	// https://github.com/ethereum/execution-apis/blob/main/src/eth/filter.yaml
	eth_newFilter: (filter: Filter) => Uint;
	eth_newBlockFilter: () => Uint;
	eth_newPendingTransactionFilter: () => Uint;
	eth_uninstallFilter: (filterIdentifier: Uint) => boolean;
	eth_getFilterChanges: (filterIdentifier: Uint) => FilterResultsAPI;
	eth_getFilterLogs: (filterIdentifier: Uint) => FilterResultsAPI;
	eth_getLogs: (filter: Filter) => FilterResultsAPI;

	// https://github.com/ethereum/execution-apis/blob/main/src/eth/mining.yaml
	eth_mining: () => boolean;
	eth_hashrate: () => Uint;
	eth_getWork: () => [HexString32Bytes, HexString32Bytes, HexString32Bytes];
	eth_submitWork: (
		nonce: HexString8Bytes,
		hash: HexString32Bytes,
		digest: HexString32Bytes,
	) => boolean;
	eth_submitHashrate: (hashRate: HexString32Bytes, id: HexString32Bytes) => boolean;

	// https://github.com/ethereum/execution-apis/blob/main/src/eth/sign.yaml
	eth_sign: (address: Address, message: HexStringBytes) => HexString256Bytes;
	eth_signTransaction: (
		transaction: TransactionWithSenderAPI | Partial<TransactionWithSenderAPI>,
	) => HexStringBytes | SignedTransactionInfoAPI;

	// https://github.com/ethereum/execution-apis/blob/main/src/eth/state.yaml
	eth_getBalance: (address: Address, blockNumber: BlockNumberOrTag) => Uint;
	eth_getStorageAt: (
		address: Address,
		storageSlot: Uint256,
		blockNumber: BlockNumberOrTag,
	) => HexStringBytes;
	eth_getTransactionCount: (address: Address, blockNumber: BlockNumberOrTag) => Uint;
	eth_getCode: (address: Address, blockNumber: BlockNumberOrTag) => HexStringBytes;

	// https://github.com/ethereum/execution-apis/blob/main/src/eth/submit.yaml
	eth_sendTransaction: (
		transaction: TransactionWithSenderAPI | Partial<TransactionWithSenderAPI>,
	) => HexString32Bytes;
	eth_sendRawTransaction: (transaction: HexStringBytes) => HexString32Bytes;

	// https://geth.ethereum.org/docs/rpc/pubsub
	eth_subscribe: (
		...params:
			| ['newHeads']
			| ['newPendingTransactions']
			| ['syncing']
			| ['logs', { address?: HexString; topics?: HexString[] }]
	) => HexString;
	eth_unsubscribe: (subscriptionId: HexString) => HexString;
	eth_clearSubscriptions: (keepSyncing?: boolean) => void;
	// Non-supported by execution-apis specs
	eth_getCompilers: () => string[];
	eth_compileSolidity: (code: string) => CompileResultAPI;
	eth_compileLLL: (code: string) => HexStringBytes;
	eth_compileSerpent: (code: string) => HexStringBytes;
};
