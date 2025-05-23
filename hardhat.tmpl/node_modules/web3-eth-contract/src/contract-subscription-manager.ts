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

import { Web3SubscriptionConstructor, Web3SubscriptionManager } from 'web3-core';
import { EthExecutionAPI, ContractAbi, DataFormat, DEFAULT_RETURN_FORMAT } from 'web3-types';
// eslint-disable-next-line import/no-cycle
import { Contract } from './contract.js';

/**
 * Similar to `Web3SubscriptionManager` but has a reference to the Contract that uses
 */
export class ContractSubscriptionManager<
	API extends EthExecutionAPI,
	RegisteredSubs extends {
		[key: string]: Web3SubscriptionConstructor<API>;
	} = any, // = ContractSubscriptions
> extends Web3SubscriptionManager<API, RegisteredSubs> {
	public readonly parentContract: Contract<ContractAbi>;

	/**
	 *
	 * @param - Web3SubscriptionManager
	 * @param - parentContract
	 *
	 * @example
	 * ```ts
	 * const requestManager = new Web3RequestManager("ws://localhost:8545");
	 * const contract = new Contract(...)
	 * const subscriptionManager = new Web3SubscriptionManager(requestManager, {}, contract);
	 * ```
	 */
	public constructor(
		self: Web3SubscriptionManager<API, RegisteredSubs>,
		parentContract: Contract<ContractAbi>,
	) {
		super(self.requestManager, self.registeredSubscriptions);

		this.parentContract = parentContract;
	}

	/**
	 * Will create a new subscription
	 *
	 * @param name - The subscription you want to subscribe to
	 * @param args - Optional additional parameters, depending on the subscription type
	 * @param returnFormat- ({@link DataFormat} defaults to {@link DEFAULT_RETURN_FORMAT}) - Specifies how the return data from the call should be formatted.
	 *
	 * Will subscribe to a specific topic (note: name)
	 * @returns The subscription object
	 */
	public async subscribe<T extends keyof RegisteredSubs>(
		name: T,
		args?: ConstructorParameters<RegisteredSubs[T]>[0],
		returnFormat: DataFormat = DEFAULT_RETURN_FORMAT,
	): Promise<InstanceType<RegisteredSubs[T]>> {
		return super.subscribe(name, args ?? this.parentContract.options, returnFormat);
	}
}
