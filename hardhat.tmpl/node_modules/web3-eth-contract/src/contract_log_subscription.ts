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
	AbiEventFragment,
	LogsInput,
	HexString,
	Topic,
	DataFormat,
	EventLog,
	ContractAbiWithSignature,
} from 'web3-types';
import { Web3RequestManager, Web3Subscription, Web3SubscriptionManager } from 'web3-core';
import { decodeEventABI } from 'web3-eth';

/**
 * ContractLogsSubscription to be used to subscribe to events logs.
 *
 * Following events are supported and can be accessed with either {@link ContractLogsSubscription.once} or ${@link ContractLogsSubscription.on} methods.
 *
 * - **connected**: Emitted when the subscription is connected.
 * - **data**: Fires on each incoming event with the event object as argument.
 * - **changed**: Fires on each event which was removed from the blockchain. The event will have the additional property `removed: true`.
 * - **error**: Fires on each error.
 *
 * ```ts
 * const subscription = await myContract.events.MyEvent({
 *   filter: {myIndexedParam: [20,23], myOtherIndexedParam: '0x123456789...'}, // Using an array means OR: e.g. 20 or 23
 *   fromBlock: 0
 * });
 *
 * subscription.on("connected", function(subscriptionId){
 *   console.log(subscriptionId);
 * });
 *
 * subscription.on('data', function(event){
 *   console.log(event); // same results as the optional callback above
 * });
 *
 * subscription.on('changed', function(event){
 *   // remove event from local database
 * })
 *
 * subscription.on('error', function(error, receipt) { // If the transaction was rejected by the network with a receipt, the second parameter will be the receipt.
 *   ...
 * });
 *
 * // event output example
 * > {
 *   returnValues: {
 *       myIndexedParam: 20,
 *       myOtherIndexedParam: '0x123456789...',
 *       myNonIndexParam: 'My String'
 *   },
 *   raw: {
 *       data: '0x7f9fade1c0d57a7af66ab4ead79fade1c0d57a7af66ab4ead7c2c2eb7b11a91385',
 *       topics: ['0xfd43ade1c09fade1c0d57a7af66ab4ead7c2c2eb7b11a91ffdd57a7af66ab4ead7', '0x7f9fade1c0d57a7af66ab4ead79fade1c0d57a7af66ab4ead7c2c2eb7b11a91385']
 *   },
 *   event: 'MyEvent',
 *   signature: '0xfd43ade1c09fade1c0d57a7af66ab4ead7c2c2eb7b11a91ffdd57a7af66ab4ead7',
 *   logIndex: 0,
 *   transactionIndex: 0,
 *   transactionHash: '0x7f9fade1c0d57a7af66ab4ead79fade1c0d57a7af66ab4ead7c2c2eb7b11a91385',
 *   blockHash: '0xfd43ade1c09fade1c0d57a7af66ab4ead7c2c2eb7b11a91ffdd57a7af66ab4ead7',
 *   blockNumber: 1234,
 *   address: '0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe'
 * }
 * ```
 */
export class ContractLogsSubscription extends Web3Subscription<
	{
		data: EventLog;
		changed: EventLog & { removed: true };
	},
	// eslint-disable-next-line @typescript-eslint/ban-types
	{ address?: HexString; topics?: (Topic | Topic[] | null)[]; abi: AbiEventFragment }
> {
	/**
	 * Address of tye contract
	 */
	public readonly address?: HexString;

	/**
	 * The list of topics subscribed
	 */
	// eslint-disable-next-line @typescript-eslint/ban-types
	public readonly topics?: (Topic | Topic[] | null)[];

	/**
	 * The {@doclink glossary#json-interface-abi | JSON Interface} of the event.
	 */
	public readonly abi: AbiEventFragment & { signature: HexString };

	public readonly jsonInterface: ContractAbiWithSignature;

	public constructor(
		args: {
			address?: HexString;
			// eslint-disable-next-line @typescript-eslint/ban-types
			topics?: (Topic | Topic[] | null)[];
			abi: AbiEventFragment & { signature: HexString };
			jsonInterface: ContractAbiWithSignature;
		},
		options: { subscriptionManager: Web3SubscriptionManager; returnFormat?: DataFormat },
	);
	/**
	 * @deprecated This constructor overloading should not be used
	 */
	public constructor(
		args: {
			address?: HexString;
			// eslint-disable-next-line @typescript-eslint/ban-types
			topics?: (Topic | Topic[] | null)[];
			abi: AbiEventFragment & { signature: HexString };
			jsonInterface: ContractAbiWithSignature;
		},
		options: { requestManager: Web3RequestManager; returnFormat?: DataFormat },
	);
	public constructor(
		args: {
			address?: HexString;
			// eslint-disable-next-line @typescript-eslint/ban-types
			topics?: (Topic | Topic[] | null)[];
			abi: AbiEventFragment & { signature: HexString };
			jsonInterface: ContractAbiWithSignature;
		},
		options: (
			| { subscriptionManager: Web3SubscriptionManager }
			| { requestManager: Web3RequestManager }
		) & {
			returnFormat?: DataFormat;
		},
	) {
		super(
			args,
			options as { subscriptionManager: Web3SubscriptionManager; returnFormat?: DataFormat },
		);

		this.address = args.address;
		this.topics = args.topics;
		this.abi = args.abi;
		this.jsonInterface = args.jsonInterface;
	}

	protected _buildSubscriptionParams() {
		return ['logs', { address: this.address, topics: this.topics }];
	}

	protected formatSubscriptionResult(data: EventLog) {
		return decodeEventABI(this.abi, data as LogsInput, this.jsonInterface, super.returnFormat);
	}
}
/**
 * @deprecated LogsSubscription is renamed to ContractLogsSubscription in this package to not be confused with LogsSubscription at 'web3-eth'.
 */
export const LogsSubscription = ContractLogsSubscription;
