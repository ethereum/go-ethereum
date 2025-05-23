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
import { Web3RequestManager, Web3ConfigOptions } from 'web3-core';
import { toChecksumAddress, utf8ToHex } from 'web3-utils';
import { formatTransaction } from 'web3-eth';
import { Address, EthPersonalAPI, ETH_DATA_FORMAT, HexString, Transaction } from 'web3-types';
import { validator, isHexStrict } from 'web3-validator';
import { personalRpcMethods } from 'web3-rpc-methods';

export const getAccounts = async (requestManager: Web3RequestManager<EthPersonalAPI>) => {
	const result = await personalRpcMethods.getAccounts(requestManager);

	return result.map(toChecksumAddress);
};

export const newAccount = async (
	requestManager: Web3RequestManager<EthPersonalAPI>,
	password: string,
) => {
	validator.validate(['string'], [password]);

	const result = await personalRpcMethods.newAccount(requestManager, password);

	return toChecksumAddress(result);
};

export const unlockAccount = async (
	requestManager: Web3RequestManager<EthPersonalAPI>,
	address: Address,
	password: string,
	unlockDuration: number,
) => {
	validator.validate(['address', 'string', 'uint'], [address, password, unlockDuration]);

	return personalRpcMethods.unlockAccount(requestManager, address, password, unlockDuration);
};

export const lockAccount = async (
	requestManager: Web3RequestManager<EthPersonalAPI>,
	address: Address,
) => {
	validator.validate(['address'], [address]);

	return personalRpcMethods.lockAccount(requestManager, address);
};

export const importRawKey = async (
	requestManager: Web3RequestManager<EthPersonalAPI>,
	keyData: HexString,
	passphrase: string,
) => {
	validator.validate(['string', 'string'], [keyData, passphrase]);

	return personalRpcMethods.importRawKey(requestManager, keyData, passphrase);
};

export const sendTransaction = async (
	requestManager: Web3RequestManager<EthPersonalAPI>,
	tx: Transaction,
	passphrase: string,
	config?: Web3ConfigOptions,
) => {
	const formattedTx = formatTransaction(tx, ETH_DATA_FORMAT, {
		transactionSchema: config?.customTransactionSchema,
	});

	return personalRpcMethods.sendTransaction(requestManager, formattedTx, passphrase);
};

export const signTransaction = async (
	requestManager: Web3RequestManager<EthPersonalAPI>,
	tx: Transaction,
	passphrase: string,
	config?: Web3ConfigOptions,
) => {
	const formattedTx = formatTransaction(tx, ETH_DATA_FORMAT, {
		transactionSchema: config?.customTransactionSchema,
	});

	return personalRpcMethods.signTransaction(requestManager, formattedTx, passphrase);
};

export const sign = async (
	requestManager: Web3RequestManager<EthPersonalAPI>,
	data: HexString,
	address: Address,
	passphrase: string,
) => {
	validator.validate(['string', 'address', 'string'], [data, address, passphrase]);

	const dataToSign = isHexStrict(data) ? data : utf8ToHex(data);

	return personalRpcMethods.sign(requestManager, dataToSign, address, passphrase);
};

export const ecRecover = async (
	requestManager: Web3RequestManager<EthPersonalAPI>,
	signedData: HexString,
	signature: string,
) => {
	validator.validate(['string', 'string'], [signedData, signature]);

	const signedDataString = isHexStrict(signedData) ? signedData : utf8ToHex(signedData);

	return personalRpcMethods.ecRecover(requestManager, signedDataString, signature);
};
