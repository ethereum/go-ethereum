import { BlockOutput, DataFormat, EthExecutionAPI, JsonRpcSubscriptionResult, JsonRpcSubscriptionResultOld, JsonRpcNotification, Log, HexString, Web3APIParams, Web3APISpec } from 'web3-types';
import { Web3SubscriptionManager } from './web3_subscription_manager.js';
import { Web3EventEmitter, Web3EventMap } from './web3_event_emitter.js';
import { Web3RequestManager } from './web3_request_manager.js';
type CommonSubscriptionEvents = {
    data: unknown;
    error: Error;
    connected: string;
};
export declare abstract class Web3Subscription<EventMap extends Web3EventMap, ArgsType = any, API extends Web3APISpec = EthExecutionAPI, CombinedEventMap extends CommonSubscriptionEvents = EventMap & CommonSubscriptionEvents> extends Web3EventEmitter<CombinedEventMap> {
    readonly args: ArgsType;
    private readonly _subscriptionManager;
    private readonly _lastBlock?;
    private readonly _returnFormat;
    protected _id?: HexString;
    constructor(args: ArgsType, options: {
        subscriptionManager: Web3SubscriptionManager;
        returnFormat?: DataFormat;
    });
    /**
     * @deprecated This constructor overloading should not be used
     */
    constructor(args: ArgsType, options: {
        requestManager: Web3RequestManager<API>;
        returnFormat?: DataFormat;
    });
    get id(): string | undefined;
    get lastBlock(): BlockOutput | undefined;
    subscribe(): Promise<string>;
    processSubscriptionData(data: JsonRpcSubscriptionResult | JsonRpcSubscriptionResultOld<Log> | JsonRpcNotification<Log>): void;
    sendSubscriptionRequest(): Promise<string>;
    protected get returnFormat(): DataFormat;
    protected get subscriptionManager(): Web3SubscriptionManager<API, {
        [key: string]: Web3SubscriptionConstructor<API>;
    }>;
    resubscribe(): Promise<void>;
    unsubscribe(): Promise<void>;
    sendUnsubscribeRequest(): Promise<void>;
    protected formatSubscriptionResult(data: CombinedEventMap['data']): CombinedEventMap["data"];
    _processSubscriptionResult(data: CombinedEventMap['data'] | unknown): void;
    _processSubscriptionError(error: Error): void;
    protected _buildSubscriptionParams(): Web3APIParams<API, 'eth_subscribe'>;
}
export type Web3SubscriptionConstructor<API extends Web3APISpec, SubscriptionType extends Web3Subscription<any, any, API> = Web3Subscription<any, any, API>> = (new (args: any, options: {
    subscriptionManager: Web3SubscriptionManager<API>;
    returnFormat?: DataFormat;
} | {
    requestManager: Web3RequestManager<API>;
    returnFormat?: DataFormat;
}) => SubscriptionType) | (new (args: any, options: {
    subscriptionManager: Web3SubscriptionManager<API>;
    returnFormat?: DataFormat;
}) => SubscriptionType) | (new (args: any, options: {
    requestManager: Web3RequestManager<API>;
    returnFormat?: DataFormat;
}) => SubscriptionType);
export {};
//# sourceMappingURL=web3_subscriptions.d.ts.map