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
	DataFormat,
	DEFAULT_RETURN_FORMAT,
	EIP1193Provider,
	JsonRpcNotification,
	JsonRpcSubscriptionResult,
	JsonRpcSubscriptionResultOld,
	Log,
	Web3APISpec,
	Web3BaseProvider,
} from 'web3-types';
import { ProviderError, SubscriptionError } from 'web3-errors';
import { isNullish } from 'web3-utils';
import { isSupportSubscriptions } from './utils.js';
import { Web3RequestManager, Web3RequestManagerEvent } from './web3_request_manager.js';
// eslint-disable-next-line import/no-cycle
import { Web3SubscriptionConstructor } from './web3_subscriptions.js';

type ShouldUnsubscribeCondition = ({
	id,
	sub,
}: {
	id: string;
	sub: unknown;
}) => boolean | undefined;

export class Web3SubscriptionManager<
	API extends Web3APISpec = Web3APISpec,
	RegisteredSubs extends { [key: string]: Web3SubscriptionConstructor<API> } = {
		[key: string]: Web3SubscriptionConstructor<API>;
	},
> {
	private readonly _subscriptions: Map<
		string,
		InstanceType<RegisteredSubs[keyof RegisteredSubs]>
	> = new Map();

	/**
	 *
	 * @param - requestManager
	 * @param - registeredSubscriptions
	 *
	 * @example
	 * ```ts
	 * const requestManager = new Web3RequestManager("ws://localhost:8545");
	 * const subscriptionManager = new Web3SubscriptionManager(requestManager, {});
	 * ```
	 */
	public constructor(
		requestManager: Web3RequestManager<API>,
		registeredSubscriptions: RegisteredSubs,
	);
	/**
	 * @deprecated This constructor overloading should not be used
	 */
	public constructor(
		requestManager: Web3RequestManager<API>,
		registeredSubscriptions: RegisteredSubs,
		tolerateUnlinkedSubscription: boolean,
	);
	public constructor(
		public readonly requestManager: Web3RequestManager<API>,
		public readonly registeredSubscriptions: RegisteredSubs,
		private readonly tolerateUnlinkedSubscription: boolean = false,
	) {
		this.requestManager.on(Web3RequestManagerEvent.BEFORE_PROVIDER_CHANGE, async () => {
			await this.unsubscribe();
		});

		this.requestManager.on(Web3RequestManagerEvent.PROVIDER_CHANGED, () => {
			this.clear();
			this.listenToProviderEvents();
		});

		this.listenToProviderEvents();
	}

	private listenToProviderEvents() {
		const providerAsWebProvider = this.requestManager.provider as Web3BaseProvider;
		if (
			!this.requestManager.provider ||
			(typeof providerAsWebProvider?.supportsSubscriptions === 'function' &&
				!providerAsWebProvider?.supportsSubscriptions())
		) {
			return;
		}

		if (typeof (this.requestManager.provider as EIP1193Provider<API>).on === 'function') {
			if (
				typeof (this.requestManager.provider as EIP1193Provider<API>).request === 'function'
			) {
				// Listen to provider messages and data
				(this.requestManager.provider as EIP1193Provider<API>).on(
					'message',
					// eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/no-unsafe-argument
					(message: any) => this.messageListener(message),
				);
			} else {
				// eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/no-unsafe-argument
				providerAsWebProvider.on<Log>('data', (data: any) => this.messageListener(data));
			}
		}
	}

	protected messageListener(
		data?:
			| JsonRpcSubscriptionResult
			| JsonRpcSubscriptionResultOld<Log>
			| JsonRpcNotification<Log>,
	) {
		if (!data) {
			throw new SubscriptionError('Should not call messageListener with no data. Type was');
		}
		const subscriptionId =
			(data as JsonRpcNotification).params?.subscription ||
			(data as JsonRpcSubscriptionResultOld).data?.subscription ||
			(data as JsonRpcSubscriptionResult).id?.toString(16);

		// Process if the received data is related to a subscription
		if (subscriptionId) {
			const sub = this._subscriptions.get(subscriptionId);
			sub?.processSubscriptionData(data);
		}
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
		const Klass: RegisteredSubs[T] = this.registeredSubscriptions[name];
		if (!Klass) {
			throw new SubscriptionError('Invalid subscription type');
		}

		// eslint-disable-next-line @typescript-eslint/no-unsafe-argument
		const subscription = new Klass(args ?? undefined, {
			subscriptionManager: this as Web3SubscriptionManager<API, RegisteredSubs>,
			returnFormat,
			// eslint.disable-next-line @typescript-eslint/no-unsafe-any
		} as any) as InstanceType<RegisteredSubs[T]>;

		await this.addSubscription(subscription);

		return subscription;
	}

	/**
	 * Will returns all subscriptions.
	 */
	public get subscriptions() {
		return this._subscriptions;
	}

	/**
	 *
	 * Adds an instance of {@link Web3Subscription} and subscribes to it
	 *
	 * @param sub - A {@link Web3Subscription} object
	 */
	public async addSubscription(sub: InstanceType<RegisteredSubs[keyof RegisteredSubs]>) {
		if (!this.requestManager.provider) {
			throw new ProviderError('Provider not available');
		}

		if (!this.supportsSubscriptions()) {
			throw new SubscriptionError('The current provider does not support subscriptions');
		}

		if (sub.id && this._subscriptions.has(sub.id)) {
			throw new SubscriptionError(`Subscription with id "${sub.id}" already exists`);
		}

		await sub.sendSubscriptionRequest();

		if (isNullish(sub.id)) {
			throw new SubscriptionError('Subscription is not subscribed yet.');
		}

		this._subscriptions.set(sub.id, sub);

		return sub.id;
	}

	/**
	 * Will clear a subscription
	 *
	 * @param id - The subscription of type {@link Web3Subscription}  to remove
	 */
	public async removeSubscription(sub: InstanceType<RegisteredSubs[keyof RegisteredSubs]>) {
		const { id } = sub;

		if (isNullish(id)) {
			throw new SubscriptionError(
				'Subscription is not subscribed yet. Or, had already been unsubscribed but not through the Subscription Manager.',
			);
		}

		if (!this._subscriptions.has(id) && !this.tolerateUnlinkedSubscription) {
			throw new SubscriptionError(`Subscription with id "${id.toString()}" does not exists`);
		}

		await sub.sendUnsubscribeRequest();
		this._subscriptions.delete(id);
		return id;
	}
	/**
	 * Will unsubscribe all subscriptions that fulfill the condition
	 *
	 * @param condition - A function that access and `id` and a `subscription` and return `true` or `false`
	 * @returns An array of all the un-subscribed subscriptions
	 */
	public async unsubscribe(condition?: ShouldUnsubscribeCondition) {
		const result = [];
		for (const [id, sub] of this.subscriptions.entries()) {
			if (!condition || (typeof condition === 'function' && condition({ id, sub }))) {
				result.push(this.removeSubscription(sub));
			}
		}

		return Promise.all(result);
	}

	/**
	 * Clears all subscriptions
	 */
	public clear() {
		this._subscriptions.clear();
	}

	/**
	 * Check whether the current provider supports subscriptions.
	 *
	 * @returns `true` or `false` depending on if the current provider supports subscriptions
	 */
	public supportsSubscriptions(): boolean {
		return isNullish(this.requestManager.provider)
			? false
			: isSupportSubscriptions(this.requestManager.provider);
	}
}
