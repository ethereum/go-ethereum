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

/* eslint-disable max-classes-per-file */

import {
	Bytes,
	HexString,
	Numbers,
	TransactionReceipt,
	Web3ValidationErrorObject,
} from 'web3-types';
import {
	ERR_RAW_TX_UNDEFINED,
	ERR_TX,
	ERR_TX_BLOCK_TIMEOUT,
	ERR_TX_CONTRACT_NOT_STORED,
	ERR_TX_CHAIN_ID_MISMATCH,
	ERR_TX_DATA_AND_INPUT,
	ERR_TX_GAS_MISMATCH,
	ERR_TX_CHAIN_MISMATCH,
	ERR_TX_HARDFORK_MISMATCH,
	ERR_TX_INVALID_CALL,
	ERR_TX_INVALID_CHAIN_INFO,
	ERR_TX_INVALID_FEE_MARKET_GAS,
	ERR_TX_INVALID_FEE_MARKET_GAS_PRICE,
	ERR_TX_INVALID_LEGACY_FEE_MARKET,
	ERR_TX_INVALID_LEGACY_GAS,
	ERR_TX_INVALID_NONCE_OR_CHAIN_ID,
	ERR_TX_INVALID_OBJECT,
	ERR_TX_INVALID_SENDER,
	ERR_TX_INVALID_RECEIVER,
	ERR_TX_LOCAL_WALLET_NOT_AVAILABLE,
	ERR_TX_MISSING_CHAIN_INFO,
	ERR_TX_MISSING_CUSTOM_CHAIN,
	ERR_TX_MISSING_CUSTOM_CHAIN_ID,
	ERR_TX_MISSING_GAS,
	ERR_TX_NO_CONTRACT_ADDRESS,
	ERR_TX_NOT_FOUND,
	ERR_TX_OUT_OF_GAS,
	ERR_TX_POLLING_TIMEOUT,
	ERR_TX_RECEIPT_MISSING_BLOCK_NUMBER,
	ERR_TX_RECEIPT_MISSING_OR_BLOCKHASH_NULL,
	ERR_TX_REVERT_INSTRUCTION,
	ERR_TX_REVERT_TRANSACTION,
	ERR_TX_REVERT_WITHOUT_REASON,
	ERR_TX_SEND_TIMEOUT,
	ERR_TX_SIGNING,
	ERR_TX_UNABLE_TO_POPULATE_NONCE,
	ERR_TX_UNSUPPORTED_EIP_1559,
	ERR_TX_UNSUPPORTED_TYPE,
	ERR_TX_REVERT_TRANSACTION_CUSTOM_ERROR,
	ERR_TX_INVALID_PROPERTIES_FOR_TYPE,
	ERR_TX_MISSING_GAS_INNER_ERROR,
	ERR_TX_GAS_MISMATCH_INNER_ERROR,
} from '../error_codes.js';
import { InvalidValueError, BaseWeb3Error } from '../web3_error_base.js';

export class TransactionError<ReceiptType = TransactionReceipt> extends BaseWeb3Error {
	public code = ERR_TX;

	public constructor(message: string, public receipt?: ReceiptType) {
		super(message);
	}

	public toJSON() {
		return { ...super.toJSON(), receipt: this.receipt };
	}
}

export class RevertInstructionError extends BaseWeb3Error {
	public code = ERR_TX_REVERT_INSTRUCTION;

	public constructor(public reason: string, public signature: string) {
		super(`Your request got reverted with the following reason string: ${reason}`);
	}

	public toJSON() {
		return { ...super.toJSON(), reason: this.reason, signature: this.signature };
	}
}

export class TransactionRevertInstructionError<
	ReceiptType = TransactionReceipt,
> extends BaseWeb3Error {
	public code = ERR_TX_REVERT_TRANSACTION;

	public constructor(
		public reason: string,
		public signature?: string,
		public receipt?: ReceiptType,
		public data?: string,
	) {
		super(
			`Transaction has been reverted by the EVM${
				receipt === undefined ? '' : `:\n ${BaseWeb3Error.convertToString(receipt)}`
			}`,
		);
	}

	public toJSON() {
		return {
			...super.toJSON(),
			reason: this.reason,
			signature: this.signature,
			receipt: this.receipt,
			data: this.data,
		};
	}
}

/**
 * This error is used when a transaction to a smart contract fails and
 * a custom user error (https://blog.soliditylang.org/2021/04/21/custom-errors/)
 * is able to be parsed from the revert reason
 */
export class TransactionRevertWithCustomError<
	ReceiptType = TransactionReceipt,
> extends TransactionRevertInstructionError<ReceiptType> {
	public code = ERR_TX_REVERT_TRANSACTION_CUSTOM_ERROR;

	public constructor(
		public reason: string,
		public customErrorName: string,
		public customErrorDecodedSignature: string,
		public customErrorArguments: Record<string, unknown>,
		public signature?: string,
		public receipt?: ReceiptType,
		public data?: string,
	) {
		super(reason);
	}

	public toJSON() {
		return {
			...super.toJSON(),
			reason: this.reason,
			customErrorName: this.customErrorName,
			customErrorDecodedSignature: this.customErrorDecodedSignature,
			customErrorArguments: this.customErrorArguments,
			signature: this.signature,
			receipt: this.receipt,
			data: this.data,
		};
	}
}

export class NoContractAddressFoundError extends TransactionError {
	public constructor(receipt: TransactionReceipt) {
		super("The transaction receipt didn't contain a contract address.", receipt);
		this.code = ERR_TX_NO_CONTRACT_ADDRESS;
	}

	public toJSON() {
		return { ...super.toJSON(), receipt: this.receipt };
	}
}

export class ContractCodeNotStoredError extends TransactionError {
	public constructor(receipt: TransactionReceipt) {
		super("The contract code couldn't be stored, please check your gas limit.", receipt);
		this.code = ERR_TX_CONTRACT_NOT_STORED;
	}
}

export class TransactionRevertedWithoutReasonError<
	ReceiptType = TransactionReceipt,
> extends TransactionError<ReceiptType> {
	public constructor(receipt?: ReceiptType) {
		super(
			`Transaction has been reverted by the EVM${
				receipt === undefined ? '' : `:\n ${BaseWeb3Error.convertToString(receipt)}`
			}`,
			receipt,
		);
		this.code = ERR_TX_REVERT_WITHOUT_REASON;
	}
}

export class TransactionOutOfGasError extends TransactionError {
	public constructor(receipt: TransactionReceipt) {
		super(
			`Transaction ran out of gas. Please provide more gas:\n ${JSON.stringify(
				receipt,
				undefined,
				2,
			)}`,
			receipt,
		);
		this.code = ERR_TX_OUT_OF_GAS;
	}
}

export class UndefinedRawTransactionError extends TransactionError {
	public constructor() {
		super(`Raw transaction undefined`);
		this.code = ERR_RAW_TX_UNDEFINED;
	}
}
export class TransactionNotFound extends TransactionError {
	public constructor() {
		super('Transaction not found');
		this.code = ERR_TX_NOT_FOUND;
	}
}

export class InvalidTransactionWithSender extends InvalidValueError {
	public code = ERR_TX_INVALID_SENDER;

	public constructor(value: unknown) {
		super(value, 'invalid transaction with invalid sender');
	}
}
export class InvalidTransactionWithReceiver extends InvalidValueError {
	public code = ERR_TX_INVALID_RECEIVER;

	public constructor(value: unknown) {
		super(value, 'invalid transaction with invalid receiver');
	}
}
export class InvalidTransactionCall extends InvalidValueError {
	public code = ERR_TX_INVALID_CALL;

	public constructor(value: unknown) {
		super(value, 'invalid transaction call');
	}
}

export class MissingCustomChainError extends InvalidValueError {
	public code = ERR_TX_MISSING_CUSTOM_CHAIN;

	public constructor() {
		super(
			'MissingCustomChainError',
			'If tx.common is provided it must have tx.common.customChain',
		);
	}
}

export class MissingCustomChainIdError extends InvalidValueError {
	public code = ERR_TX_MISSING_CUSTOM_CHAIN_ID;

	public constructor() {
		super(
			'MissingCustomChainIdError',
			'If tx.common is provided it must have tx.common.customChain and tx.common.customChain.chainId',
		);
	}
}

export class ChainIdMismatchError extends InvalidValueError {
	public code = ERR_TX_CHAIN_ID_MISMATCH;

	public constructor(value: { txChainId: unknown; customChainId: unknown }) {
		super(
			JSON.stringify(value),
			// https://github.com/ChainSafe/web3.js/blob/8783f4d64e424456bdc53b34ef1142d0a7cee4d7/packages/web3-eth-accounts/src/index.js#L176
			'Chain Id doesnt match in tx.chainId tx.common.customChain.chainId',
		);
	}
}

export class ChainMismatchError extends InvalidValueError {
	public code = ERR_TX_CHAIN_MISMATCH;

	public constructor(value: { txChain: unknown; baseChain: unknown }) {
		super(JSON.stringify(value), 'Chain doesnt match in tx.chain tx.common.basechain');
	}
}

export class HardforkMismatchError extends InvalidValueError {
	public code = ERR_TX_HARDFORK_MISMATCH;

	public constructor(value: { txHardfork: unknown; commonHardfork: unknown }) {
		super(JSON.stringify(value), 'hardfork doesnt match in tx.hardfork tx.common.hardfork');
	}
}

export class CommonOrChainAndHardforkError extends InvalidValueError {
	public code = ERR_TX_INVALID_CHAIN_INFO;

	public constructor() {
		super(
			'CommonOrChainAndHardforkError',
			'Please provide the common object or the chain and hardfork property but not all together.',
		);
	}
}

export class MissingChainOrHardforkError extends InvalidValueError {
	public code = ERR_TX_MISSING_CHAIN_INFO;

	public constructor(value: { chain: string | undefined; hardfork: string | undefined }) {
		super(
			'MissingChainOrHardforkError',
			`When specifying chain and hardfork, both values must be defined. Received "chain": ${
				value.chain ?? 'undefined'
			}, "hardfork": ${value.hardfork ?? 'undefined'}`,
		);
	}
}

export class MissingGasInnerError extends BaseWeb3Error {
	public code = ERR_TX_MISSING_GAS_INNER_ERROR;

	public constructor() {
		super(
			'Missing properties in transaction, either define "gas" and "gasPrice" for type 0 transactions or "gas", "maxPriorityFeePerGas" and "maxFeePerGas" for type 2 transactions',
		);
	}
}

export class MissingGasError extends InvalidValueError {
	public code = ERR_TX_MISSING_GAS;

	public constructor(value: {
		gas: Numbers | undefined;
		gasPrice: Numbers | undefined;
		maxPriorityFeePerGas: Numbers | undefined;
		maxFeePerGas: Numbers | undefined;
	}) {
		super(
			`gas: ${value.gas ?? 'undefined'}, gasPrice: ${
				value.gasPrice ?? 'undefined'
			}, maxPriorityFeePerGas: ${value.maxPriorityFeePerGas ?? 'undefined'}, maxFeePerGas: ${
				value.maxFeePerGas ?? 'undefined'
			}`,
			'"gas" is missing',
		);
		this.cause = new MissingGasInnerError();
	}
}

export class TransactionGasMismatchInnerError extends BaseWeb3Error {
	public code = ERR_TX_GAS_MISMATCH_INNER_ERROR;

	public constructor() {
		super(
			'Missing properties in transaction, either define "gas" and "gasPrice" for type 0 transactions or "gas", "maxPriorityFeePerGas" and "maxFeePerGas" for type 2 transactions, not both',
		);
	}
}

export class TransactionGasMismatchError extends InvalidValueError {
	public code = ERR_TX_GAS_MISMATCH;

	public constructor(value: {
		gas: Numbers | undefined;
		gasPrice: Numbers | undefined;
		maxPriorityFeePerGas: Numbers | undefined;
		maxFeePerGas: Numbers | undefined;
	}) {
		super(
			`gas: ${value.gas ?? 'undefined'}, gasPrice: ${
				value.gasPrice ?? 'undefined'
			}, maxPriorityFeePerGas: ${value.maxPriorityFeePerGas ?? 'undefined'}, maxFeePerGas: ${
				value.maxFeePerGas ?? 'undefined'
			}`,
			'transaction must specify legacy or fee market gas properties, not both',
		);
		this.cause = new TransactionGasMismatchInnerError();
	}
}

export class InvalidGasOrGasPrice extends InvalidValueError {
	public code = ERR_TX_INVALID_LEGACY_GAS;

	public constructor(value: { gas: Numbers | undefined; gasPrice: Numbers | undefined }) {
		super(
			`gas: ${value.gas ?? 'undefined'}, gasPrice: ${value.gasPrice ?? 'undefined'}`,
			'Gas or gasPrice is lower than 0',
		);
	}
}

export class InvalidMaxPriorityFeePerGasOrMaxFeePerGas extends InvalidValueError {
	public code = ERR_TX_INVALID_FEE_MARKET_GAS;

	public constructor(value: {
		maxPriorityFeePerGas: Numbers | undefined;
		maxFeePerGas: Numbers | undefined;
	}) {
		super(
			`maxPriorityFeePerGas: ${value.maxPriorityFeePerGas ?? 'undefined'}, maxFeePerGas: ${
				value.maxFeePerGas ?? 'undefined'
			}`,
			'maxPriorityFeePerGas or maxFeePerGas is lower than 0',
		);
	}
}

export class Eip1559GasPriceError extends InvalidValueError {
	public code = ERR_TX_INVALID_FEE_MARKET_GAS_PRICE;

	public constructor(value: unknown) {
		super(value, "eip-1559 transactions don't support gasPrice");
	}
}

export class UnsupportedFeeMarketError extends InvalidValueError {
	public code = ERR_TX_INVALID_LEGACY_FEE_MARKET;

	public constructor(value: {
		maxPriorityFeePerGas: Numbers | undefined;
		maxFeePerGas: Numbers | undefined;
	}) {
		super(
			`maxPriorityFeePerGas: ${value.maxPriorityFeePerGas ?? 'undefined'}, maxFeePerGas: ${
				value.maxFeePerGas ?? 'undefined'
			}`,
			"pre-eip-1559 transaction don't support maxFeePerGas/maxPriorityFeePerGas",
		);
	}
}

export class InvalidTransactionObjectError extends InvalidValueError {
	public code = ERR_TX_INVALID_OBJECT;

	public constructor(value: unknown) {
		super(value, 'invalid transaction object');
	}
}

export class InvalidNonceOrChainIdError extends InvalidValueError {
	public code = ERR_TX_INVALID_NONCE_OR_CHAIN_ID;

	public constructor(value: { nonce: Numbers | undefined; chainId: Numbers | undefined }) {
		super(
			`nonce: ${value.nonce ?? 'undefined'}, chainId: ${value.chainId ?? 'undefined'}`,
			'Nonce or chainId is lower than 0',
		);
	}
}

export class UnableToPopulateNonceError extends InvalidValueError {
	public code = ERR_TX_UNABLE_TO_POPULATE_NONCE;

	public constructor() {
		super('UnableToPopulateNonceError', 'unable to populate nonce, no from address available');
	}
}

export class Eip1559NotSupportedError extends InvalidValueError {
	public code = ERR_TX_UNSUPPORTED_EIP_1559;

	public constructor() {
		super('Eip1559NotSupportedError', "Network doesn't support eip-1559");
	}
}

export class UnsupportedTransactionTypeError extends InvalidValueError {
	public code = ERR_TX_UNSUPPORTED_TYPE;

	public constructor(value: unknown) {
		super(value, 'unsupported transaction type');
	}
}

export class TransactionDataAndInputError extends InvalidValueError {
	public code = ERR_TX_DATA_AND_INPUT;

	public constructor(value: { data: HexString | undefined; input: HexString | undefined }) {
		super(
			`data: ${value.data ?? 'undefined'}, input: ${value.input ?? 'undefined'}`,
			'You can\'t have "data" and "input" as properties of transactions at the same time, please use either "data" or "input" instead.',
		);
	}
}

export class TransactionSendTimeoutError extends BaseWeb3Error {
	public code = ERR_TX_SEND_TIMEOUT;

	public constructor(value: { numberOfSeconds: number; transactionHash?: Bytes }) {
		super(
			`The connected Ethereum Node did not respond within ${
				value.numberOfSeconds
			} seconds, please make sure your transaction was properly sent and you are connected to a healthy Node. Be aware that transaction might still be pending or mined!\n\tTransaction Hash: ${
				value.transactionHash ? value.transactionHash.toString() : 'not available'
			}`,
		);
	}
}

function transactionTimeoutHint(transactionHash?: Bytes) {
	return `Please make sure your transaction was properly sent and there are no previous pending transaction for the same account. However, be aware that it might still be mined!\n\tTransaction Hash: ${
		transactionHash ? transactionHash.toString() : 'not available'
	}`;
}

export class TransactionPollingTimeoutError extends BaseWeb3Error {
	public code = ERR_TX_POLLING_TIMEOUT;

	public constructor(value: { numberOfSeconds: number; transactionHash: Bytes }) {
		super(
			`Transaction was not mined within ${
				value.numberOfSeconds
			} seconds. ${transactionTimeoutHint(value.transactionHash)}`,
		);
	}
}

export class TransactionBlockTimeoutError extends BaseWeb3Error {
	public code = ERR_TX_BLOCK_TIMEOUT;

	public constructor(value: {
		starterBlockNumber: number;
		numberOfBlocks: number;
		transactionHash?: Bytes;
	}) {
		super(
			`Transaction started at ${value.starterBlockNumber} but was not mined within ${
				value.numberOfBlocks
			} blocks. ${transactionTimeoutHint(value.transactionHash)}`,
		);
	}
}

export class TransactionMissingReceiptOrBlockHashError extends InvalidValueError {
	public code = ERR_TX_RECEIPT_MISSING_OR_BLOCKHASH_NULL;

	public constructor(value: {
		receipt: TransactionReceipt;
		blockHash: Bytes;
		transactionHash: Bytes;
	}) {
		super(
			`receipt: ${JSON.stringify(
				value.receipt,
			)}, blockHash: ${value.blockHash?.toString()}, transactionHash: ${value.transactionHash?.toString()}`,
			`Receipt missing or blockHash null`,
		);
	}
}

export class TransactionReceiptMissingBlockNumberError extends InvalidValueError {
	public code = ERR_TX_RECEIPT_MISSING_BLOCK_NUMBER;

	public constructor(value: { receipt: TransactionReceipt }) {
		super(`receipt: ${JSON.stringify(value.receipt)}`, `Receipt missing block number`);
	}
}

export class TransactionSigningError extends BaseWeb3Error {
	public code = ERR_TX_SIGNING;
	public constructor(errorDetails: string) {
		super(`Invalid signature. "${errorDetails}"`);
	}
}

export class LocalWalletNotAvailableError extends InvalidValueError {
	public code = ERR_TX_LOCAL_WALLET_NOT_AVAILABLE;

	public constructor() {
		super(
			'LocalWalletNotAvailableError',
			`Attempted to index account in local wallet, but no wallet is available`,
		);
	}
}
export class InvalidPropertiesForTransactionTypeError extends BaseWeb3Error {
	public code = ERR_TX_INVALID_PROPERTIES_FOR_TYPE;

	public constructor(
		validationError: Web3ValidationErrorObject[],
		txType: '0x0' | '0x1' | '0x2',
	) {
		const invalidPropertyNames: string[] = [];
		validationError.forEach(error => invalidPropertyNames.push(error.keyword));
		super(
			`The following properties are invalid for the transaction type ${txType}: ${invalidPropertyNames.join(
				', ',
			)}`,
		);
	}
}
