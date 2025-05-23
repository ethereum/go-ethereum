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

import { format, toHex } from 'web3-utils';
import { TransactionTypeParser, Web3Context } from 'web3-core';
import { EthExecutionAPI, HardforksOrdered, Transaction, ETH_DATA_FORMAT } from 'web3-types';
import { Web3ValidatorError, isNullish, validator } from 'web3-validator';
import { InvalidPropertiesForTransactionTypeError } from 'web3-errors';

import { InternalTransaction } from '../types.js';

// undefined is treated as null for JSON schema validator
const transactionType0x0Schema = {
	type: 'object',
	properties: {
		accessList: {
			type: 'null',
		},
		maxFeePerGas: {
			type: 'null',
		},
		maxPriorityFeePerGas: {
			type: 'null',
		},
	},
};
const transactionType0x1Schema = {
	type: 'object',
	properties: {
		maxFeePerGas: {
			type: 'null',
		},
		maxPriorityFeePerGas: {
			type: 'null',
		},
	},
};
const transactionType0x2Schema = {
	type: 'object',
	properties: {
		gasPrice: {
			type: 'null',
		},
	},
};

const validateTxTypeAndHandleErrors = (
	txSchema: object,
	tx: Transaction,
	txType: '0x0' | '0x1' | '0x2',
) => {
	try {
		validator.validateJSONSchema(txSchema, tx);
	} catch (error) {
		if (error instanceof Web3ValidatorError)
			// Erroneously reported error
			// eslint-disable-next-line @typescript-eslint/no-unsafe-call
			throw new InvalidPropertiesForTransactionTypeError(error.errors, txType);

		throw error;
	}
};

export const defaultTransactionTypeParser: TransactionTypeParser = transaction => {
	const tx = transaction as unknown as Transaction;
	if (!isNullish(tx.type)) {
		let txSchema;
		switch (tx.type) {
			case '0x0':
				txSchema = transactionType0x0Schema;
				break;
			case '0x1':
				txSchema = transactionType0x1Schema;
				break;
			case '0x2':
				txSchema = transactionType0x2Schema;
				break;

			default:
				return format({ format: 'uint' }, tx.type, ETH_DATA_FORMAT);
		}

		validateTxTypeAndHandleErrors(txSchema, tx, tx.type);

		return format({ format: 'uint' }, tx.type, ETH_DATA_FORMAT);
	}

	if (!isNullish(tx.maxFeePerGas) || !isNullish(tx.maxPriorityFeePerGas)) {
		validateTxTypeAndHandleErrors(transactionType0x2Schema, tx, '0x2');
		return '0x2';
	}

	if (!isNullish(tx.accessList)) {
		validateTxTypeAndHandleErrors(transactionType0x1Schema, tx, '0x1');
		return '0x1';
	}

	const givenHardfork = tx.hardfork ?? tx.common?.hardfork;

	if (!isNullish(givenHardfork)) {
		const hardforkIndex = Object.keys(HardforksOrdered).indexOf(givenHardfork);

		// givenHardfork is London or later, so EIP-2718 is supported
		if (hardforkIndex >= Object.keys(HardforksOrdered).indexOf('london'))
			return !isNullish(tx.gasPrice) ? '0x0' : '0x2';

		// givenHardfork is Berlin, tx.accessList is undefined, assume type is 0x0
		if (hardforkIndex === Object.keys(HardforksOrdered).indexOf('berlin')) return '0x0';
	}

	// gasprice is defined
	if (!isNullish(tx.gasPrice)) {
		validateTxTypeAndHandleErrors(transactionType0x0Schema, tx, '0x0');
		return '0x0';
	}

	// no transaction type can be inferred from properties, use default transaction type
	return undefined;
};

export const detectTransactionType = (
	transaction: InternalTransaction,
	web3Context?: Web3Context<EthExecutionAPI>,
) =>
	(web3Context?.transactionTypeParser ?? defaultTransactionTypeParser)(
		transaction as unknown as Record<string, unknown>,
	);

export const detectRawTransactionType = (transaction: Uint8Array) =>
	transaction[0] > 0x7f ? '0x0' : toHex(transaction[0]);
