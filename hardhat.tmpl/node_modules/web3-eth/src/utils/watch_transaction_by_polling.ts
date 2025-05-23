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
import { Bytes, EthExecutionAPI, TransactionReceipt } from 'web3-types';
import { Web3Context, Web3PromiEvent } from 'web3-core';
import { format, numberToHex } from 'web3-utils';
import { ethRpcMethods } from 'web3-rpc-methods';

import { DataFormat } from 'web3-types';
import { JsonSchema } from 'web3-validator';
import { SendSignedTransactionEvents, SendTransactionEvents } from '../types.js';
import { transactionReceiptSchema } from '../schemas.js';

export type Web3PromiEventEventTypeBase<ReturnFormat extends DataFormat> =
	| SendTransactionEvents<ReturnFormat>
	| SendSignedTransactionEvents<ReturnFormat>;

export type WaitProps<ReturnFormat extends DataFormat, ResolveType = TransactionReceipt> = {
	web3Context: Web3Context<EthExecutionAPI>;
	transactionReceipt: TransactionReceipt;
	customTransactionReceiptSchema?: JsonSchema;
	transactionPromiEvent: Web3PromiEvent<ResolveType, Web3PromiEventEventTypeBase<ReturnFormat>>;
	returnFormat: ReturnFormat;
};

/**
 * This function watches a Transaction by subscribing to new heads.
 * It is used by `watchTransactionForConfirmations`, in case the provider does not support subscription.
 * And it is also used by `watchTransactionBySubscription`, as a fallback, if the subscription failed for any reason.
 */
export const watchTransactionByPolling = <
	ReturnFormat extends DataFormat,
	ResolveType = TransactionReceipt,
>({
	web3Context,
	transactionReceipt,
	transactionPromiEvent,
	customTransactionReceiptSchema,
	returnFormat,
}: WaitProps<ReturnFormat, ResolveType>) => {
	// Having a transactionReceipt means that the transaction has already been included
	// in at least one block, so we start with 1
	let confirmations = 1;
	const intervalId = setInterval(() => {
		(async () => {
			if (confirmations >= web3Context.transactionConfirmationBlocks) {
				clearInterval(intervalId);
				return;
			}

			const nextBlock = await ethRpcMethods.getBlockByNumber(
				web3Context.requestManager,
				numberToHex(BigInt(transactionReceipt.blockNumber) + BigInt(confirmations)),
				false,
			);

			if (nextBlock?.hash) {
				confirmations += 1;

				transactionPromiEvent.emit('confirmation', {
					confirmations: format({ format: 'uint' }, confirmations, returnFormat),
					receipt: format(
						customTransactionReceiptSchema ?? transactionReceiptSchema,
						transactionReceipt,
						returnFormat,
					),
					latestBlockHash: format(
						{ format: 'bytes32' },
						nextBlock.hash as Bytes,
						returnFormat,
					),
				});
			}
		})() as unknown;
	}, web3Context.transactionReceiptPollingInterval ?? web3Context.transactionPollingInterval);
};
