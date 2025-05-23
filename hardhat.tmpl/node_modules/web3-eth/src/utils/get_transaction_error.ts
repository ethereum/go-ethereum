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

import { Web3Context } from 'web3-core';
import {
	TransactionRevertedWithoutReasonError,
	TransactionRevertInstructionError,
	TransactionRevertWithCustomError,
} from 'web3-errors';
import {
	DataFormat,
	FormatType,
	ContractAbi,
	TransactionCall,
	TransactionReceipt,
} from 'web3-types';
import { RevertReason, RevertReasonWithCustomError } from '../types.js';
// eslint-disable-next-line import/no-cycle
import { getRevertReason, parseTransactionError } from './get_revert_reason.js';

export async function getTransactionError<ReturnFormat extends DataFormat>(
	web3Context: Web3Context,
	transactionFormatted?: TransactionCall,
	transactionReceiptFormatted?: FormatType<TransactionReceipt, ReturnFormat>,
	receivedError?: unknown,
	contractAbi?: ContractAbi,
	knownReason?: string | RevertReason | RevertReasonWithCustomError,
) {
	let _reason: string | RevertReason | RevertReasonWithCustomError | undefined = knownReason;

	if (_reason === undefined) {
		if (receivedError !== undefined) {
			_reason = parseTransactionError(receivedError);
		} else if (web3Context.handleRevert && transactionFormatted !== undefined) {
			_reason = await getRevertReason(web3Context, transactionFormatted, contractAbi);
		}
	}

	let error:
		| TransactionRevertedWithoutReasonError<FormatType<TransactionReceipt, ReturnFormat>>
		| TransactionRevertInstructionError<FormatType<TransactionReceipt, ReturnFormat>>
		| TransactionRevertWithCustomError<FormatType<TransactionReceipt, ReturnFormat>>;
	if (_reason === undefined) {
		error = new TransactionRevertedWithoutReasonError<
			FormatType<TransactionReceipt, ReturnFormat>
		>(transactionReceiptFormatted);
	} else if (typeof _reason === 'string') {
		error = new TransactionRevertInstructionError<FormatType<TransactionReceipt, ReturnFormat>>(
			_reason,
			undefined,
			transactionReceiptFormatted,
		);
	} else if (
		(_reason as RevertReasonWithCustomError).customErrorName !== undefined &&
		(_reason as RevertReasonWithCustomError).customErrorDecodedSignature !== undefined &&
		(_reason as RevertReasonWithCustomError).customErrorArguments !== undefined
	) {
		const reasonWithCustomError: RevertReasonWithCustomError =
			_reason as RevertReasonWithCustomError;
		error = new TransactionRevertWithCustomError<FormatType<TransactionReceipt, ReturnFormat>>(
			reasonWithCustomError.reason,
			reasonWithCustomError.customErrorName,
			reasonWithCustomError.customErrorDecodedSignature,
			reasonWithCustomError.customErrorArguments,
			reasonWithCustomError.signature,
			transactionReceiptFormatted,
			reasonWithCustomError.data,
		);
	} else {
		error = new TransactionRevertInstructionError<FormatType<TransactionReceipt, ReturnFormat>>(
			_reason.reason,
			_reason.signature,
			transactionReceiptFormatted,
			_reason.data,
		);
	}

	return error;
}
