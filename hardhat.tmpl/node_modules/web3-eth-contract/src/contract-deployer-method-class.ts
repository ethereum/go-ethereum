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

import { Web3ContractError } from 'web3-errors';
import { sendTransaction, SendTransactionEvents, SendTransactionOptions } from 'web3-eth';
import { decodeFunctionCall } from 'web3-eth-abi';
import {
	AbiConstructorFragment,
	AbiFunctionFragment,
	ContractAbi,
	ContractConstructorArgs,
	Bytes,
	HexString,
	PayableCallOptions,
	DataFormat,
	DEFAULT_RETURN_FORMAT,
	ContractOptions,
	TransactionReceipt,
	TransactionCall,
} from 'web3-types';
import { format } from 'web3-utils';
import { isNullish } from 'web3-validator';
import { Web3PromiEvent } from 'web3-core';
import { encodeMethodABI } from './encoding.js';
import { NonPayableTxOptions, PayableTxOptions } from './types.js';
import { getSendTxParams } from './utils.js';
// eslint-disable-next-line import/no-cycle
import { Contract } from './contract.js';

export type ContractDeploySend<Abi extends ContractAbi> = Web3PromiEvent<
	// eslint-disable-next-line no-use-before-define
	Contract<Abi>,
	SendTransactionEvents<DataFormat>
>;

/*
 * This class is only supposed to be used for the return of `new Contract(...).deploy(...)` method.
 */
export class DeployerMethodClass<FullContractAbi extends ContractAbi> {
	protected readonly args: never[] | ContractConstructorArgs<FullContractAbi>;
	protected readonly constructorAbi: AbiConstructorFragment;
	protected readonly contractOptions: ContractOptions;
	protected readonly deployData?: string;

	protected _contractMethodDeploySend(tx: TransactionCall) {
		// eslint-disable-next-line no-use-before-define
		const returnTxOptions: SendTransactionOptions<Contract<FullContractAbi>> = {
			transactionResolver: (receipt: TransactionReceipt) => {
				if (receipt.status === BigInt(0)) {
					throw new Web3ContractError("code couldn't be stored", receipt);
				}

				const newContract = this.parent.clone();

				// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
				newContract.options.address = receipt.contractAddress;
				return newContract;
			},

			contractAbi: this.parent.options.jsonInterface,
			// TODO Should make this configurable by the user
			checkRevertBeforeSending: false,
		};

		return isNullish(this.parent.getTransactionMiddleware())
			? sendTransaction(this.parent, tx, this.parent.defaultReturnFormat, returnTxOptions) // not calling this with undefined Middleware because it will not break if Eth package is not updated
			: sendTransaction(
					this.parent,
					tx,
					this.parent.defaultReturnFormat,
					returnTxOptions,
					this.parent.getTransactionMiddleware(),
			  );
	}

	public constructor(
		// eslint-disable-next-line no-use-before-define
		public parent: Contract<FullContractAbi>,
		public deployOptions:
			| {
					/**
					 * The byte code of the contract.
					 */
					data?: HexString;
					input?: HexString;
					/**
					 * The arguments which get passed to the constructor on deployment.
					 */
					arguments?: ContractConstructorArgs<FullContractAbi>;
			  }
			| undefined,
	) {
		const { args, abi, contractOptions, deployData } = this.calculateDeployParams();

		this.args = args;
		this.constructorAbi = abi;
		this.contractOptions = contractOptions;
		this.deployData = deployData;
	}

	public send(options?: PayableTxOptions): ContractDeploySend<FullContractAbi> {
		const modifiedOptions = { ...options };

		const tx = this.populateTransaction(modifiedOptions);

		return this._contractMethodDeploySend(tx);
	}

	public populateTransaction(txOptions?: PayableTxOptions | NonPayableTxOptions) {
		const modifiedContractOptions = {
			...this.contractOptions,
			from: this.contractOptions.from ?? this.parent.defaultAccount ?? undefined,
		};

		// args, abi, contractOptions, deployData

		const tx = getSendTxParams({
			abi: this.constructorAbi,
			params: this.args as unknown[],
			options: { ...txOptions, dataInputFill: this.parent.contractDataInputFill },
			contractOptions: modifiedContractOptions,
		});

		// @ts-expect-error remove unnecessary field
		if (tx.dataInputFill) {
			// @ts-expect-error remove unnecessary field
			delete tx.dataInputFill;
		}
		return tx;
	}

	protected calculateDeployParams() {
		let abi = this.parent.options.jsonInterface.find(
			j => j.type === 'constructor',
		) as AbiConstructorFragment;
		if (!abi) {
			abi = {
				type: 'constructor',
				stateMutability: '',
			} as AbiConstructorFragment;
		}

		const _input = format(
			{ format: 'bytes' },
			this.deployOptions?.input ?? this.parent.options.input,
			DEFAULT_RETURN_FORMAT,
		);

		const _data = format(
			{ format: 'bytes' },
			this.deployOptions?.data ?? this.parent.options.data,
			DEFAULT_RETURN_FORMAT,
		);

		if ((!_input || _input.trim() === '0x') && (!_data || _data.trim() === '0x')) {
			throw new Web3ContractError('contract creation without any data provided.');
		}

		const args = this.deployOptions?.arguments ?? [];

		const contractOptions: ContractOptions = {
			...this.parent.options,
			input: _input,
			data: _data,
		};
		const deployData = _input ?? _data;

		return { args, abi, contractOptions, deployData };
	}

	public async estimateGas<ReturnFormat extends DataFormat = typeof DEFAULT_RETURN_FORMAT>(
		options?: PayableCallOptions,
		returnFormat: ReturnFormat = this.parent.defaultReturnFormat as ReturnFormat,
	) {
		const modifiedOptions = { ...options };
		return this.parent.contractMethodEstimateGas({
			abi: this.constructorAbi as AbiFunctionFragment,
			params: this.args as unknown[],
			returnFormat,
			options: modifiedOptions,
			contractOptions: this.contractOptions,
		});
	}

	public encodeABI() {
		return encodeMethodABI(
			this.constructorAbi,
			this.args as unknown[],
			format(
				{ format: 'bytes' },
				this.deployData as Bytes,
				this.parent.defaultReturnFormat as typeof DEFAULT_RETURN_FORMAT,
			),
		);
	}

	public decodeData(data: HexString) {
		return {
			...decodeFunctionCall(
				this.constructorAbi,
				data.replace(this.deployData as string, ''),
				false,
			),
			__method__: this.constructorAbi.type,
		};
	}
}
