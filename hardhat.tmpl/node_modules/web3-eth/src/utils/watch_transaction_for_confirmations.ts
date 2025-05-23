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
import { Bytes, EthExecutionAPI, Web3BaseProvider, TransactionReceipt } from 'web3-types';
import { Web3Context, Web3PromiEvent } from 'web3-core';
import { format } from 'web3-utils';
import { isNullish, JsonSchema } from 'web3-validator';

import {
	TransactionMissingReceiptOrBlockHashError,
	TransactionReceiptMissingBlockNumberError,
} from 'web3-errors';
import { DataFormat } from 'web3-types';
import { transactionReceiptSchema } from '../schemas.js';
import {
	watchTransactionByPolling,
	Web3PromiEventEventTypeBase,
} from './watch_transaction_by_polling.js';
import { watchTransactionBySubscription } from './watch_transaction_by_subscription.js';

export function watchTransactionForConfirmations<
	ReturnFormat extends DataFormat,
	Web3PromiEventEventType extends Web3PromiEventEventTypeBase<ReturnFormat>,
	ResolveType = TransactionReceipt,
>(
	web3Context: Web3Context<EthExecutionAPI>,
	transactionPromiEvent: Web3PromiEvent<ResolveType, Web3PromiEventEventType>,
	transactionReceipt: TransactionReceipt,
	transactionHash: Bytes,
	returnFormat: ReturnFormat,
	customTransactionReceiptSchema?: JsonSchema,
) {
	if (isNullish(transactionReceipt) || isNullish(transactionReceipt.blockHash))
		throw new TransactionMissingReceiptOrBlockHashError({
			receipt: transactionReceipt,
			blockHash: format({ format: 'bytes32' }, transactionReceipt?.blockHash, returnFormat),
			transactionHash: format({ format: 'bytes32' }, transactionHash, returnFormat),
		});

	if (!transactionReceipt.blockNumber)
		throw new TransactionReceiptMissingBlockNumberError({ receipt: transactionReceipt });

	// As we have the receipt, it's the first confirmation that tx is accepted.
	transactionPromiEvent.emit('confirmation', {
		confirmations: format({ format: 'uint' }, 1, returnFormat),
		receipt: format(
			customTransactionReceiptSchema ?? transactionReceiptSchema,
			transactionReceipt,
			returnFormat,
		),
		latestBlockHash: format({ format: 'bytes32' }, transactionReceipt.blockHash, returnFormat),
	} as Web3PromiEventEventType['confirmation']);

	// so a subscription for newBlockHeaders can be made instead of polling
	const provider: Web3BaseProvider = web3Context.requestManager.provider as Web3BaseProvider;
	if (provider && 'supportsSubscriptions' in provider && provider.supportsSubscriptions()) {
		watchTransactionBySubscription({
			web3Context,
			transactionReceipt,
			transactionPromiEvent,
			customTransactionReceiptSchema,
			returnFormat,
		});
	} else {
		watchTransactionByPolling({
			web3Context,
			transactionReceipt,
			transactionPromiEvent,
			customTransactionReceiptSchema,
			returnFormat,
		});
	}
}
