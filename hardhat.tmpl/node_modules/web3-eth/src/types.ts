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
	ContractExecutionError,
	TransactionRevertedWithoutReasonError,
	TransactionRevertInstructionError,
	TransactionRevertWithCustomError,
	InvalidResponseError,
	TransactionPollingTimeoutError,
} from 'web3-errors';
import {
	FormatType,
	ETH_DATA_FORMAT,
	DataFormat,
	Bytes,
	ContractAbi,
	HexString,
	Numbers,
	Transaction,
	TransactionReceipt,
	TransactionWithFromAndToLocalWalletIndex,
	TransactionWithFromLocalWalletIndex,
	TransactionWithToLocalWalletIndex,
} from 'web3-types';
import { Schema } from 'web3-validator';

export type InternalTransaction = FormatType<Transaction, typeof ETH_DATA_FORMAT>;

export type SendTransactionEventsBase<ReturnFormat extends DataFormat, TxType> = {
	sending: FormatType<TxType, typeof ETH_DATA_FORMAT>;
	sent: FormatType<TxType, typeof ETH_DATA_FORMAT>;
	transactionHash: FormatType<Bytes, ReturnFormat>;
	receipt: FormatType<TransactionReceipt, ReturnFormat>;
	confirmation: {
		confirmations: FormatType<Numbers, ReturnFormat>;
		receipt: FormatType<TransactionReceipt, ReturnFormat>;
		latestBlockHash: FormatType<Bytes, ReturnFormat>;
	};
	error:
		| TransactionRevertedWithoutReasonError<FormatType<TransactionReceipt, ReturnFormat>>
		| TransactionRevertInstructionError<FormatType<TransactionReceipt, ReturnFormat>>
		| TransactionRevertWithCustomError<FormatType<TransactionReceipt, ReturnFormat>>
		| TransactionPollingTimeoutError
		| InvalidResponseError
		| ContractExecutionError;
};

export type SendTransactionEvents<ReturnFormat extends DataFormat> = SendTransactionEventsBase<
	ReturnFormat,
	Transaction
>;
export type SendSignedTransactionEvents<ReturnFormat extends DataFormat> =
	SendTransactionEventsBase<ReturnFormat, Bytes>;

export interface SendTransactionOptions<ResolveType = TransactionReceipt> {
	ignoreGasPricing?: boolean;
	transactionResolver?: (receipt: TransactionReceipt) => ResolveType;
	contractAbi?: ContractAbi;
	checkRevertBeforeSending?: boolean;
	ignoreFillingGasLimit?: boolean;
}

export interface SendSignedTransactionOptions<ResolveType = TransactionReceipt> {
	transactionResolver?: (receipt: TransactionReceipt) => ResolveType;
	contractAbi?: ContractAbi;
	checkRevertBeforeSending?: boolean;
}

export interface RevertReason {
	reason: string;
	signature?: HexString;
	data?: HexString;
}

export interface RevertReasonWithCustomError extends RevertReason {
	customErrorName: string;
	customErrorDecodedSignature: string;
	customErrorArguments: Record<string, unknown>;
}

export type TransactionMiddlewareData =
	| Transaction
	| TransactionWithFromLocalWalletIndex
	| TransactionWithToLocalWalletIndex
	| TransactionWithFromAndToLocalWalletIndex;

export interface TransactionMiddleware {
	// for transaction processing before signing
	processTransaction(
		transaction: TransactionMiddlewareData,
		options?: { [key: string]: unknown },
	): Promise<TransactionMiddlewareData>;
}

export type CustomTransactionSchema = {
	type: string;
	properties: Record<string, Schema>;
};
