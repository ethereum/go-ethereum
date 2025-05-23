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
	AccessList,
	AccessListEntry,
	BaseTransactionAPI,
	Transaction1559UnsignedAPI,
	Transaction2930UnsignedAPI,
	TransactionCall,
	TransactionLegacyUnsignedAPI,
	Transaction,
	TransactionWithSenderAPI,
	ETH_DATA_FORMAT,
} from 'web3-types';
import { isAddress, isHexStrict, isHexString32Bytes, isNullish, isUInt } from 'web3-validator';
import {
	ChainMismatchError,
	HardforkMismatchError,
	ChainIdMismatchError,
	CommonOrChainAndHardforkError,
	Eip1559GasPriceError,
	InvalidGasOrGasPrice,
	InvalidMaxPriorityFeePerGasOrMaxFeePerGas,
	InvalidNonceOrChainIdError,
	InvalidTransactionCall,
	InvalidTransactionObjectError,
	InvalidTransactionWithSender,
	MissingChainOrHardforkError,
	MissingCustomChainError,
	MissingCustomChainIdError,
	MissingGasError,
	TransactionGasMismatchError,
	UnsupportedFeeMarketError,
} from 'web3-errors';
import { formatTransaction } from './utils/format_transaction.js';
import { CustomTransactionSchema, InternalTransaction } from './types.js';

export function isBaseTransaction(value: BaseTransactionAPI): boolean {
	if (!isNullish(value.to) && !isAddress(value.to)) return false;
	if (!isHexStrict(value.type) && !isNullish(value.type) && value.type.length !== 2) return false;
	if (!isHexStrict(value.nonce)) return false;
	if (!isHexStrict(value.gas)) return false;
	if (!isHexStrict(value.value)) return false;
	if (!isHexStrict(value.input)) return false;
	if (value.chainId && !isHexStrict(value.chainId)) return false;

	return true;
}

export function isAccessListEntry(value: AccessListEntry): boolean {
	if (!isNullish(value.address) && !isAddress(value.address)) return false;
	if (
		!isNullish(value.storageKeys) &&
		!value.storageKeys.every(storageKey => isHexString32Bytes(storageKey))
	)
		return false;

	return true;
}

export function isAccessList(value: AccessList): boolean {
	if (
		!Array.isArray(value) ||
		!value.every(accessListEntry => isAccessListEntry(accessListEntry))
	)
		return false;

	return true;
}

export function isTransaction1559Unsigned(value: Transaction1559UnsignedAPI): boolean {
	if (!isBaseTransaction(value)) return false;
	if (!isHexStrict(value.maxFeePerGas)) return false;
	if (!isHexStrict(value.maxPriorityFeePerGas)) return false;
	if (!isAccessList(value.accessList)) return false;

	return true;
}

export function isTransaction2930Unsigned(value: Transaction2930UnsignedAPI): boolean {
	if (!isBaseTransaction(value)) return false;
	if (!isHexStrict(value.gasPrice)) return false;
	if (!isAccessList(value.accessList)) return false;

	return true;
}

export function isTransactionLegacyUnsigned(value: TransactionLegacyUnsignedAPI): boolean {
	if (!isBaseTransaction(value)) return false;
	if (!isHexStrict(value.gasPrice)) return false;

	return true;
}

export function isTransactionWithSender(value: TransactionWithSenderAPI): boolean {
	if (!isAddress(value.from)) return false;
	if (!isBaseTransaction(value)) return false;
	if (
		!isTransaction1559Unsigned(value as Transaction1559UnsignedAPI) &&
		!isTransaction2930Unsigned(value as Transaction2930UnsignedAPI) &&
		!isTransactionLegacyUnsigned(value as TransactionLegacyUnsignedAPI)
	)
		return false;

	return true;
}

export function validateTransactionWithSender(value: TransactionWithSenderAPI) {
	if (!isTransactionWithSender(value)) throw new InvalidTransactionWithSender(value);
}

export function isTransactionCall(value: TransactionCall): boolean {
	if (!isNullish(value.from) && !isAddress(value.from)) return false;
	if (!isAddress(value.to)) return false;
	if (!isNullish(value.gas) && !isHexStrict(value.gas)) return false;
	if (!isNullish(value.gasPrice) && !isHexStrict(value.gasPrice)) return false;
	if (!isNullish(value.value) && !isHexStrict(value.value)) return false;
	if (!isNullish(value.data) && !isHexStrict(value.data)) return false;
	if (!isNullish(value.input) && !isHexStrict(value.input)) return false;
	if (!isNullish(value.type)) return false;
	if (isTransaction1559Unsigned(value as Transaction1559UnsignedAPI)) return false;
	if (isTransaction2930Unsigned(value as Transaction2930UnsignedAPI)) return false;

	return true;
}

export function validateTransactionCall(value: TransactionCall) {
	if (!isTransactionCall(value)) throw new InvalidTransactionCall(value);
}

export const validateCustomChainInfo = (transaction: InternalTransaction) => {
	if (!isNullish(transaction.common)) {
		if (isNullish(transaction.common.customChain)) throw new MissingCustomChainError();
		if (isNullish(transaction.common.customChain.chainId))
			throw new MissingCustomChainIdError();
		if (
			!isNullish(transaction.chainId) &&
			transaction.chainId !== transaction.common.customChain.chainId
		)
			throw new ChainIdMismatchError({
				txChainId: transaction.chainId,
				customChainId: transaction.common.customChain.chainId,
			});
	}
};
export const validateChainInfo = (transaction: InternalTransaction) => {
	if (
		!isNullish(transaction.common) &&
		!isNullish(transaction.chain) &&
		!isNullish(transaction.hardfork)
	) {
		throw new CommonOrChainAndHardforkError();
	}
	if (
		(!isNullish(transaction.chain) && isNullish(transaction.hardfork)) ||
		(!isNullish(transaction.hardfork) && isNullish(transaction.chain))
	)
		throw new MissingChainOrHardforkError({
			chain: transaction.chain,
			hardfork: transaction.hardfork,
		});
};
export const validateBaseChain = (transaction: InternalTransaction) => {
	if (!isNullish(transaction.common))
		if (!isNullish(transaction.common.baseChain))
			if (
				!isNullish(transaction.chain) &&
				transaction.chain !== transaction.common.baseChain
			) {
				throw new ChainMismatchError({
					txChain: transaction.chain,
					baseChain: transaction.common.baseChain,
				});
			}
};
export const validateHardfork = (transaction: InternalTransaction) => {
	if (!isNullish(transaction.common))
		if (!isNullish(transaction.common.hardfork))
			if (
				!isNullish(transaction.hardfork) &&
				transaction.hardfork !== transaction.common.hardfork
			) {
				throw new HardforkMismatchError({
					txHardfork: transaction.hardfork,
					commonHardfork: transaction.common.hardfork,
				});
			}
};

export const validateLegacyGas = (transaction: InternalTransaction) => {
	if (
		// This check is verifying gas and gasPrice aren't less than 0.
		isNullish(transaction.gas) ||
		!isUInt(transaction.gas) ||
		isNullish(transaction.gasPrice) ||
		!isUInt(transaction.gasPrice)
	)
		throw new InvalidGasOrGasPrice({
			gas: transaction.gas,
			gasPrice: transaction.gasPrice,
		});
	if (!isNullish(transaction.maxFeePerGas) || !isNullish(transaction.maxPriorityFeePerGas))
		throw new UnsupportedFeeMarketError({
			maxFeePerGas: transaction.maxFeePerGas,
			maxPriorityFeePerGas: transaction.maxPriorityFeePerGas,
		});
};

export const validateFeeMarketGas = (transaction: InternalTransaction) => {
	// These errors come from 1.x, so they must be checked before
	// InvalidMaxPriorityFeePerGasOrMaxFeePerGas to throw the same error
	// for the same code executing in 1.x
	if (!isNullish(transaction.gasPrice) && transaction.type === '0x2')
		throw new Eip1559GasPriceError(transaction.gasPrice);
	if (transaction.type === '0x0' || transaction.type === '0x1')
		throw new UnsupportedFeeMarketError({
			maxFeePerGas: transaction.maxFeePerGas,
			maxPriorityFeePerGas: transaction.maxPriorityFeePerGas,
		});

	if (
		isNullish(transaction.maxFeePerGas) ||
		!isUInt(transaction.maxFeePerGas) ||
		isNullish(transaction.maxPriorityFeePerGas) ||
		!isUInt(transaction.maxPriorityFeePerGas)
	)
		throw new InvalidMaxPriorityFeePerGasOrMaxFeePerGas({
			maxPriorityFeePerGas: transaction.maxPriorityFeePerGas,
			maxFeePerGas: transaction.maxFeePerGas,
		});
};

/**
 * This method checks if all required gas properties are present for either
 * legacy gas (type 0x0 and 0x1) OR fee market transactions (0x2)
 */
export const validateGas = (transaction: InternalTransaction) => {
	const gasPresent = !isNullish(transaction.gas) || !isNullish(transaction.gasLimit);
	const legacyGasPresent = gasPresent && !isNullish(transaction.gasPrice);
	const feeMarketGasPresent =
		gasPresent &&
		!isNullish(transaction.maxPriorityFeePerGas) &&
		!isNullish(transaction.maxFeePerGas);

	if (!legacyGasPresent && !feeMarketGasPresent)
		throw new MissingGasError({
			gas: transaction.gas,
			gasPrice: transaction.gasPrice,
			maxPriorityFeePerGas: transaction.maxPriorityFeePerGas,
			maxFeePerGas: transaction.maxFeePerGas,
		});

	if (legacyGasPresent && feeMarketGasPresent)
		throw new TransactionGasMismatchError({
			gas: transaction.gas,
			gasPrice: transaction.gasPrice,
			maxPriorityFeePerGas: transaction.maxPriorityFeePerGas,
			maxFeePerGas: transaction.maxFeePerGas,
		});

	(legacyGasPresent ? validateLegacyGas : validateFeeMarketGas)(transaction);
	(!isNullish(transaction.type) && transaction.type > '0x1'
		? validateFeeMarketGas
		: validateLegacyGas)(transaction);
};

export const validateTransactionForSigning = (
	transaction: InternalTransaction,
	overrideMethod?: (transaction: InternalTransaction) => void,
	options: {
		transactionSchema?: CustomTransactionSchema;
	} = { transactionSchema: undefined },
) => {
	if (!isNullish(overrideMethod)) {
		overrideMethod(transaction);
		return;
	}

	if (typeof transaction !== 'object' || isNullish(transaction))
		throw new InvalidTransactionObjectError(transaction);

	validateCustomChainInfo(transaction);
	validateChainInfo(transaction);
	validateBaseChain(transaction);
	validateHardfork(transaction);

	const formattedTransaction = formatTransaction(transaction as Transaction, ETH_DATA_FORMAT, {
		transactionSchema: options.transactionSchema,
	});
	validateGas(formattedTransaction);

	if (
		isNullish(formattedTransaction.nonce) ||
		isNullish(formattedTransaction.chainId) ||
		formattedTransaction.nonce.startsWith('-') ||
		formattedTransaction.chainId.startsWith('-')
	)
		throw new InvalidNonceOrChainIdError({
			nonce: transaction.nonce,
			chainId: transaction.chainId,
		});
};
