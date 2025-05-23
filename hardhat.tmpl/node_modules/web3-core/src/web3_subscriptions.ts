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

// eslint-disable-next-line max-classes-per-file
import {
	BlockOutput,
	DEFAULT_RETURN_FORMAT,
	DataFormat,
	EthExecutionAPI,
	JsonRpcSubscriptionResult,
	JsonRpcSubscriptionResultOld,
	JsonRpcNotification,
	Log,
	HexString,
	Web3APIParams,
	Web3APISpec,
} from 'web3-types';
import { jsonRpc } from 'web3-utils';

// eslint-disable-next-line import/no-cycle
import { Web3SubscriptionManager } from './web3_subscription_manager.js';
import { Web3EventEmitter, Web3EventMap } from './web3_event_emitter.js';
import { Web3RequestManager } from './web3_request_manager.js';

type CommonSubscriptionEvents = {
	data: unknown; // Fires on each incoming block header.
	error: Error; // Fires when an error in the subscription occurs.
	connected: string; // Fires once after the subscription successfully connected. Returns the subscription id.
};

export abstract class Web3Subscription<
	EventMap extends Web3EventMap,
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	ArgsType = any,
	API extends Web3APISpec = EthExecutionAPI,
	// The following generic type is just to define the type `CombinedEventMap` and use it inside the class
	// 	it combines the user passed `EventMap` with the `CommonSubscriptionEvents`
	//	However, this type definition could be refactored depending on the closure of
	//	[Permit type alias declarations inside a class](https://github.com/microsoft/TypeScript/issues/7061)
	CombinedEventMap extends CommonSubscriptionEvents = EventMap & CommonSubscriptionEvents,
> extends Web3EventEmitter<CombinedEventMap> {
	private readonly _subscriptionManager: Web3SubscriptionManager<API>;
	private readonly _lastBlock?: BlockOutput;
	private readonly _returnFormat: DataFormat;
	protected _id?: HexString;

	public constructor(
		args: ArgsType,
		options: { subscriptionManager: Web3SubscriptionManager; returnFormat?: DataFormat },
	);
	/**
	 * @deprecated This constructor overloading should not be used
	 */
	public constructor(
		args: ArgsType,
		options: { requestManager: Web3RequestManager<API>; returnFormat?: DataFormat },
	);
	public constructor(
		public readonly args: ArgsType,
		options: (
			| { subscriptionManager: Web3SubscriptionManager }
			| { requestManager: Web3RequestManager<API> }
		) & {
			returnFormat?: DataFormat;
		},
	) {
		super();
		const { requestManager } = options as { requestManager: Web3RequestManager<API> };
		const { subscriptionManager } = options as { subscriptionManager: Web3SubscriptionManager };
		if (requestManager) {
			// eslint-disable-next-line deprecation/deprecation
			this._subscriptionManager = new Web3SubscriptionManager(requestManager, {}, true);
		} else {
			this._subscriptionManager = subscriptionManager;
		}

		this._returnFormat = options?.returnFormat ?? (DEFAULT_RETURN_FORMAT as DataFormat);
	}

	public get id() {
		return this._id;
	}

	public get lastBlock() {
		return this._lastBlock;
	}

	public async subscribe(): Promise<string> {
		return this._subscriptionManager.addSubscription(this);
	}

	public processSubscriptionData(
		data:
			| JsonRpcSubscriptionResult
			| JsonRpcSubscriptionResultOld<Log>
			| JsonRpcNotification<Log>,
	) {
		if (data?.data) {
			// for EIP-1193 provider
			this._processSubscriptionResult(data?.data?.result ?? data?.data);
		} else if (
			data &&
			jsonRpc.isResponseWithNotification(
				data as unknown as JsonRpcSubscriptionResult | JsonRpcNotification<Log>,
			)
		) {
			this._processSubscriptionResult(data?.params.result);
		}
	}

	public async sendSubscriptionRequest(): Promise<string> {
		this._id = await this._subscriptionManager.requestManager.send({
			method: 'eth_subscribe',
			params: this._buildSubscriptionParams(),
		});

		this.emit('connected', this._id);
		return this._id;
	}

	protected get returnFormat() {
		return this._returnFormat;
	}

	protected get subscriptionManager() {
		return this._subscriptionManager;
	}

	public async resubscribe() {
		await this.unsubscribe();
		await this.subscribe();
	}

	public async unsubscribe() {
		if (!this.id) {
			return;
		}

		await this._subscriptionManager.removeSubscription(this);
	}

	public async sendUnsubscribeRequest() {
		await this._subscriptionManager.requestManager.send({
			method: 'eth_unsubscribe',
			params: [this.id] as Web3APIParams<API, 'eth_unsubscribe'>,
		});
		this._id = undefined;
	}

	// eslint-disable-next-line class-methods-use-this
	protected formatSubscriptionResult(data: CombinedEventMap['data']) {
		return data;
	}

	public _processSubscriptionResult(data: CombinedEventMap['data'] | unknown) {
		this.emit('data', this.formatSubscriptionResult(data));
	}

	public _processSubscriptionError(error: Error) {
		this.emit('error', error);
	}

	// eslint-disable-next-line class-methods-use-this
	protected _buildSubscriptionParams(): Web3APIParams<API, 'eth_subscribe'> {
		// This should be overridden in the subclass
		throw new Error('Implement in the child class');
	}
}

export type Web3SubscriptionConstructor<
	API extends Web3APISpec,
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	SubscriptionType extends Web3Subscription<any, any, API> = Web3Subscription<any, any, API>,
> =
	| (new (
			// We accept any type of arguments here and don't deal with this type internally
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			args: any,
			options:
				| { subscriptionManager: Web3SubscriptionManager<API>; returnFormat?: DataFormat }
				| { requestManager: Web3RequestManager<API>; returnFormat?: DataFormat },
	  ) => SubscriptionType)
	| (new (
			args: any,
			options: {
				subscriptionManager: Web3SubscriptionManager<API>;
				returnFormat?: DataFormat;
			},
	  ) => SubscriptionType)
	| (new (
			args: any,
			options: { requestManager: Web3RequestManager<API>; returnFormat?: DataFormat },
	  ) => SubscriptionType);
