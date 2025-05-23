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
import { ethRpcMethods } from 'web3-rpc-methods';
import { DataFormat, EthExecutionAPI, Numbers, Log, FilterParams } from 'web3-types';
import { format, numberToHex } from 'web3-utils';
import { isNullish } from 'web3-validator';
import { logSchema } from './schemas.js';

/**
 * View additional documentations here: {@link Web3Eth.createNewPendingTransactionFilter}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param returnFormat ({@link DataFormat}) Return format
 */
export async function createNewPendingTransactionFilter<ReturnFormat extends DataFormat>(
	web3Context: Web3Context<EthExecutionAPI>,
	returnFormat: ReturnFormat,
) {
	const response = await ethRpcMethods.newPendingTransactionFilter(web3Context.requestManager);

	return format(
		{ format: 'uint' },
		response as Numbers,
		returnFormat ?? web3Context.defaultReturnFormat,
	);
}

/**
 * View additional documentations here: {@link Web3Eth.createNewFilter}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param filter ({@link FilterParam}) Filter param optional having from-block to-block address or params
 * @param returnFormat ({@link DataFormat}) Return format
 */
export async function createNewFilter<ReturnFormat extends DataFormat>(
	web3Context: Web3Context<EthExecutionAPI>,
	filter: FilterParams,
	returnFormat: ReturnFormat,
) {
	// format type bigint or number toBlock and fromBlock to hexstring.
	let { toBlock, fromBlock } = filter;
	if (!isNullish(toBlock)) {
		if (typeof toBlock === 'number' || typeof toBlock === 'bigint') {
			toBlock = numberToHex(toBlock);
		}
	}
	if (!isNullish(fromBlock)) {
		if (typeof fromBlock === 'number' || typeof fromBlock === 'bigint') {
			fromBlock = numberToHex(fromBlock);
		}
	}

	const formattedFilter = { ...filter, fromBlock, toBlock };

	const response = await ethRpcMethods.newFilter(web3Context.requestManager, formattedFilter);

	return format(
		{ format: 'uint' },
		response as Numbers,
		returnFormat ?? web3Context.defaultReturnFormat,
	);
}

/**
 * View additional documentations here: {@link Web3Eth.createNewBlockFilter}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param returnFormat ({@link DataFormat}) Return format
 */
export async function createNewBlockFilter<ReturnFormat extends DataFormat>(
	web3Context: Web3Context<EthExecutionAPI>,
	returnFormat: ReturnFormat,
) {
	const response = await ethRpcMethods.newBlockFilter(web3Context.requestManager);

	return format(
		{ format: 'uint' },
		response as Numbers,
		returnFormat ?? web3Context.defaultReturnFormat,
	);
}

/**
 * View additional documentations here: {@link Web3Eth.uninstallFilter}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param filterIdentifier ({@link Numbers}) filter id
 */
export async function uninstallFilter(
	web3Context: Web3Context<EthExecutionAPI>,
	filterIdentifier: Numbers,
) {
	const response = await ethRpcMethods.uninstallFilter(
		web3Context.requestManager,
		numberToHex(filterIdentifier),
	);

	return response;
}

/**
 * View additional documentations here: {@link Web3Eth.getFilterChanges}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param filterIdentifier ({@link Numbers}) filter id
 */
export async function getFilterChanges<ReturnFormat extends DataFormat>(
	web3Context: Web3Context<EthExecutionAPI>,
	filterIdentifier: Numbers,
	returnFormat: ReturnFormat,
) {
	const response = await ethRpcMethods.getFilterChanges(
		web3Context.requestManager,
		numberToHex(filterIdentifier),
	);

	const result = response.map(res => {
		if (typeof res === 'string') {
			return res;
		}

		return format(
			logSchema,
			res as unknown as Log,
			returnFormat ?? web3Context.defaultReturnFormat,
		);
	});

	return result;
}

/**
 * View additional documentations here: {@link Web3Eth.getFilterLogs}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param filterIdentifier ({@link Numbers}) filter id
 */
export async function getFilterLogs<ReturnFormat extends DataFormat>(
	web3Context: Web3Context<EthExecutionAPI>,
	filterIdentifier: Numbers,
	returnFormat: ReturnFormat,
) {
	const response = await ethRpcMethods.getFilterLogs(
		web3Context.requestManager,
		numberToHex(filterIdentifier),
	);

	const result = response.map(res => {
		if (typeof res === 'string') {
			return res;
		}

		return format(
			logSchema,
			res as unknown as Log,
			returnFormat ?? web3Context.defaultReturnFormat,
		);
	});

	return result;
}
