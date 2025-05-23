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

import { Bytes, Transaction } from 'web3-types';
import Eth from 'web3-eth';
import {
	decodeLog,
	decodeParameter,
	decodeParameters,
	encodeFunctionCall,
	encodeFunctionSignature,
	encodeParameter,
	encodeParameters,
} from 'web3-eth-abi';
import {
	encrypt,
	hashMessage,
	recover,
	recoverTransaction,
	sign,
	signTransaction,
	Wallet,
	Web3Account,
} from 'web3-eth-accounts';
import { Contract } from 'web3-eth-contract';
import { ENS } from 'web3-eth-ens';
import { Net } from 'web3-net';
import { Iban } from 'web3-eth-iban';
import { Personal } from 'web3-eth-personal';

export type { Web3Account, Wallet } from 'web3-eth-accounts';

/**
 * The Ethereum interface for main web3 object. It provides extra methods in addition to `web3-eth` interface.
 *
 * {@link web3_eth.Web3Eth} for details about the `Eth` interface.
 */
export interface Web3EthInterface extends Eth {
	/**
	 * Extended [Contract](/api/web3-eth-contract/class/Contract) constructor for main `web3` object. See [Contract](/api/web3-eth-contract/class/Contract) for further details.
	 *
	 * You can use `.setProvider` on this constructor to set provider for **all the instances** of the contracts which were created by `web3.eth.Contract`.
	 * Please check the {@doclink guides/web3_upgrade_guide/providers_migration_guide | following guide} to understand more about setting provider.
	 *
	 * ```ts
	 * web3.eth.Contract.setProvider(myProvider)
	 * ```
	 */
	Contract: typeof Contract;
	Iban: typeof Iban;
	net: Net;
	ens: ENS;
	abi: {
		encodeEventSignature: typeof encodeFunctionSignature;
		encodeFunctionCall: typeof encodeFunctionCall;
		encodeFunctionSignature: typeof encodeFunctionSignature;
		encodeParameter: typeof encodeParameter;
		encodeParameters: typeof encodeParameters;
		decodeParameter: typeof decodeParameter;
		decodeParameters: typeof decodeParameters;
		decodeLog: typeof decodeLog;
	};
	accounts: {
		create: () => Web3Account;
		privateKeyToAccount: (privateKey: Uint8Array | string) => Web3Account;
		signTransaction: (
			transaction: Transaction,
			privateKey: Bytes,
		) => ReturnType<typeof signTransaction>;
		recoverTransaction: typeof recoverTransaction;
		hashMessage: typeof hashMessage;
		sign: typeof sign;
		recover: typeof recover;
		encrypt: typeof encrypt;
		decrypt: (
			keystore: string,
			password: string,
			options?: Record<string, unknown>,
		) => Promise<Web3Account>;
		wallet: Wallet;
		privateKeyToAddress: (privateKey: Bytes) => string;
		privateKeyToPublicKey: (privateKey: Bytes, isCompressed: boolean) => string;
		parseAndValidatePrivateKey: (data: Bytes, ignoreLength?: boolean) => Uint8Array;
	};
	personal: Personal;
}
