import { DataFormat, JsonRpcNotification, JsonRpcSubscriptionResult, JsonRpcSubscriptionResultOld, Log, Web3APISpec } from 'web3-types';
import { Web3RequestManager } from './web3_request_manager.js';
import { Web3SubscriptionConstructor } from './web3_subscriptions.js';
type ShouldUnsubscribeCondition = ({ id, sub, }: {
    id: string;
    sub: unknown;
}) => boolean | undefined;
export declare class Web3SubscriptionManager<API extends Web3APISpec = Web3APISpec, RegisteredSubs extends {
    [key: string]: Web3SubscriptionConstructor<API>;
} = {
    [key: string]: Web3SubscriptionConstructor<API>;
}> {
    readonly requestManager: Web3RequestManager<API>;
    readonly registeredSubscriptions: RegisteredSubs;
    private readonly tolerateUnlinkedSubscription;
    private readonly _subscriptions;
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
    constructor(requestManager: Web3RequestManager<API>, registeredSubscriptions: RegisteredSubs);
    /**
     * @deprecated This constructor overloading should not be used
     */
    constructor(requestManager: Web3RequestManager<API>, registeredSubscriptions: RegisteredSubs, tolerateUnlinkedSubscription: boolean);
    private listenToProviderEvents;
    protected messageListener(data?: JsonRpcSubscriptionResult | JsonRpcSubscriptionResultOld<Log> | JsonRpcNotification<Log>): void;
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
    subscribe<T extends keyof RegisteredSubs>(name: T, args?: ConstructorParameters<RegisteredSubs[T]>[0], returnFormat?: DataFormat): Promise<InstanceType<RegisteredSubs[T]>>;
    /**
     * Will returns all subscriptions.
     */
    get subscriptions(): Map<string, InstanceType<RegisteredSubs[keyof RegisteredSubs]>>;
    /**
     *
     * Adds an instance of {@link Web3Subscription} and subscribes to it
     *
     * @param sub - A {@link Web3Subscription} object
     */
    addSubscription(sub: InstanceType<RegisteredSubs[keyof RegisteredSubs]>): Promise<string>;
    /**
     * Will clear a subscription
     *
     * @param id - The subscription of type {@link Web3Subscription}  to remove
     */
    removeSubscription(sub: InstanceType<RegisteredSubs[keyof RegisteredSubs]>): Promise<string>;
    /**
     * Will unsubscribe all subscriptions that fulfill the condition
     *
     * @param condition - A function that access and `id` and a `subscription` and return `true` or `false`
     * @returns An array of all the un-subscribed subscriptions
     */
    unsubscribe(condition?: ShouldUnsubscribeCondition): Promise<string[]>;
    /**
     * Clears all subscriptions
     */
    clear(): void;
    /**
     * Check whether the current provider supports subscriptions.
     *
     * @returns `true` or `false` depending on if the current provider supports subscriptions
     */
    supportsSubscriptions(): boolean;
}
export {};
//# sourceMappingURL=web3_subscription_manager.d.ts.map