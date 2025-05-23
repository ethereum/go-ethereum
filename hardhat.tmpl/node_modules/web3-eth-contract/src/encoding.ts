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

import { format, isNullish, keccak256 } from 'web3-utils';

import {
	AbiConstructorFragment,
	AbiEventFragment,
	AbiFunctionFragment,
	Filter,
	HexString,
	Topic,
	FMT_NUMBER,
	FMT_BYTES,
	ContractOptions,
} from 'web3-types';

import {
	decodeFunctionCall,
	decodeFunctionReturn,
	encodeEventSignature,
	encodeFunctionSignature,
	encodeParameter,
	encodeParameters,
	inferTypesAndEncodeParameters,
	isAbiConstructorFragment,
	jsonInterfaceMethodToString,
} from 'web3-eth-abi';

import { blockSchema, ALL_EVENTS } from 'web3-eth';
import { Web3ContractError } from 'web3-errors';

export { decodeEventABI } from 'web3-eth';

type Writeable<T> = { -readonly [P in keyof T]: T[P] };
export const encodeEventABI = (
	{ address }: ContractOptions,
	event: AbiEventFragment & { signature: string },
	options?: Filter,
) => {
	const topics = options?.topics;
	const filter = options?.filter ?? {};
	const opts: Writeable<Filter> = {};

	if (!isNullish(options?.fromBlock)) {
		opts.fromBlock = format(blockSchema.properties.number, options?.fromBlock, {
			number: FMT_NUMBER.HEX,
			bytes: FMT_BYTES.HEX,
		});
	}
	if (!isNullish(options?.toBlock)) {
		opts.toBlock = format(blockSchema.properties.number, options?.toBlock, {
			number: FMT_NUMBER.HEX,
			bytes: FMT_BYTES.HEX,
		});
	}

	if (topics && Array.isArray(topics)) {
		opts.topics = [...topics] as Topic[];
	} else {
		opts.topics = [];
		// add event signature
		if (event && !event.anonymous && ![ALL_EVENTS, 'allEvents'].includes(event.name)) {
			opts.topics.push(
				event.signature ?? encodeEventSignature(jsonInterfaceMethodToString(event)),
			);
		}

		// add event topics (indexed arguments)
		if (![ALL_EVENTS, 'allEvents'].includes(event.name) && event.inputs) {
			for (const input of event.inputs) {
				if (!input.indexed) {
					continue;
				}

				const value = filter[input.name];
				if (!value) {
					// eslint-disable-next-line no-null/no-null
					opts.topics.push(null);
					continue;
				}

				// TODO: https://github.com/ethereum/web3.js/issues/344
				// TODO: deal properly with components
				if (Array.isArray(value)) {
					opts.topics.push(value.map(v => encodeParameter(input.type, v)));
				} else if (input.type === 'string') {
					opts.topics.push(keccak256(value as string));
				} else {
					opts.topics.push(encodeParameter(input.type, value));
				}
			}
		}
	}

	if (!opts.topics.length) delete opts.topics;

	if (address) {
		opts.address = address.toLowerCase();
	}

	return opts;
};

export const encodeMethodABI = (
	abi: AbiFunctionFragment | AbiConstructorFragment,
	args: unknown[],
	deployData?: HexString,
) => {
	const inputLength = Array.isArray(abi.inputs) ? abi.inputs.length : 0;
	if (abi.inputs && inputLength !== args.length) {
		throw new Web3ContractError(
			`The number of arguments is not matching the methods required number. You need to pass ${inputLength} arguments.`,
		);
	}

	let params: string;
	if (abi.inputs) {
		params = encodeParameters(Array.isArray(abi.inputs) ? abi.inputs : [], args).replace(
			'0x',
			'',
		);
	} else {
		params = inferTypesAndEncodeParameters(args).replace('0x', '');
	}

	if (isAbiConstructorFragment(abi)) {
		if (!deployData)
			throw new Web3ContractError(
				'The contract has no contract data option set. This is necessary to append the constructor parameters.',
			);

		if (!deployData.startsWith('0x')) {
			return `0x${deployData}${params}`;
		}

		return `${deployData}${params}`;
	}

	return `${encodeFunctionSignature(abi)}${params}`;
};

/** @deprecated import `decodeFunctionCall` from ''web3-eth-abi' instead. */
export const decodeMethodParams = decodeFunctionCall;
/** @deprecated import `decodeFunctionReturn` from ''web3-eth-abi' instead. */
export const decodeMethodReturn = decodeFunctionReturn;
