import { EthExecutionAPI, HexString, Numbers, SupportedProviders, Transaction, Web3AccountProvider, Web3APISpec, Web3BaseProvider, Web3BaseWallet, Web3BaseWalletAccount } from 'web3-types';
import { BaseTransaction } from 'web3-eth-accounts';
import { ExtensionObject, RequestManagerMiddleware } from './types.js';
import { Web3BatchRequest } from './web3_batch_request.js';
import { Web3Config, Web3ConfigOptions } from './web3_config.js';
import { Web3RequestManager } from './web3_request_manager.js';
import { Web3SubscriptionConstructor } from './web3_subscriptions.js';
import { Web3SubscriptionManager } from './web3_subscription_manager.js';
export type Web3ContextObject<API extends Web3APISpec = unknown, RegisteredSubs extends {
    [key: string]: Web3SubscriptionConstructor<API>;
} = any> = {
    config: Web3ConfigOptions;
    provider?: SupportedProviders<API> | string;
    requestManager: Web3RequestManager<API>;
    subscriptionManager?: Web3SubscriptionManager<API, RegisteredSubs> | undefined;
    registeredSubscriptions?: RegisteredSubs;
    providers: typeof Web3RequestManager.providers;
    accountProvider?: Web3AccountProvider<Web3BaseWalletAccount>;
    wallet?: Web3BaseWallet<Web3BaseWalletAccount>;
};
export type Web3ContextInitOptions<API extends Web3APISpec = unknown, RegisteredSubs extends {
    [key: string]: Web3SubscriptionConstructor<API>;
} = any> = {
    config?: Partial<Web3ConfigOptions>;
    provider?: SupportedProviders<API> | string;
    requestManager?: Web3RequestManager<API>;
    subscriptionManager?: Web3SubscriptionManager<API, RegisteredSubs> | undefined;
    registeredSubscriptions?: RegisteredSubs;
    accountProvider?: Web3AccountProvider<Web3BaseWalletAccount>;
    wallet?: Web3BaseWallet<Web3BaseWalletAccount>;
    requestManagerMiddleware?: RequestManagerMiddleware<API>;
};
export type Web3ContextConstructor<T extends Web3Context, T2 extends unknown[]> = new (...args: [...extras: T2, context: Web3ContextObject]) => T;
export type Web3ContextFactory<T extends Web3Context, T2 extends unknown[]> = Web3ContextConstructor<T, T2> & {
    fromContextObject(this: Web3ContextConstructor<T, T2>, contextObject: Web3ContextObject): T;
};
export declare class Web3Context<API extends Web3APISpec = unknown, RegisteredSubs extends {
    [key: string]: Web3SubscriptionConstructor<API>;
} = any> extends Web3Config {
    static readonly providers: {
        HttpProvider: import("web3-types").Web3BaseProviderConstructor;
        WebsocketProvider: import("web3-types").Web3BaseProviderConstructor;
    };
    static givenProvider?: SupportedProviders<never>;
    readonly providers: {
        HttpProvider: import("web3-types").Web3BaseProviderConstructor;
        WebsocketProvider: import("web3-types").Web3BaseProviderConstructor;
    };
    protected _requestManager: Web3RequestManager<API>;
    protected _subscriptionManager: Web3SubscriptionManager<API, RegisteredSubs>;
    protected _accountProvider?: Web3AccountProvider<Web3BaseWalletAccount>;
    protected _wallet?: Web3BaseWallet<Web3BaseWalletAccount>;
    constructor(providerOrContext?: string | SupportedProviders<API> | Web3ContextInitOptions<API, RegisteredSubs>);
    get requestManager(): Web3RequestManager<API>;
    /**
     * Will return the current subscriptionManager ({@link Web3SubscriptionManager})
     */
    get subscriptionManager(): Web3SubscriptionManager<API, RegisteredSubs>;
    get wallet(): Web3BaseWallet<Web3BaseWalletAccount> | undefined;
    get accountProvider(): Web3AccountProvider<Web3BaseWalletAccount> | undefined;
    static fromContextObject<T extends Web3Context, T3 extends unknown[]>(this: Web3ContextConstructor<T, T3>, ...args: [Web3ContextObject, ...T3]): T;
    getContextObject(): Web3ContextObject<API, RegisteredSubs>;
    /**
     * Use to create new object of any type extended by `Web3Context`
     * and link it to current context. This can be used to initiate a global context object
     * and then use it to create new objects of any type extended by `Web3Context`.
     */
    use<T extends Web3Context, T2 extends unknown[]>(ContextRef: Web3ContextConstructor<T, T2>, ...args: [...T2]): T;
    /**
     * Link current context to another context.
     */
    link<T extends Web3Context>(parentContext: T): void;
    registerPlugin(plugin: Web3PluginBase): void;
    /**
     * Will return the current provider.
     *
     * @returns Returns the current provider
     * @example
     * ```ts
     * const web3 = new Web3Context("http://localhost:8545");
     * console.log(web3.provider);
     * > HttpProvider {
     * 	clientUrl: 'http://localhost:8545',
     * 	httpProviderOptions: undefined
     *  }
     * ```
     */
    get provider(): Web3BaseProvider<API> | undefined;
    /**
     * Will set the current provider.
     *
     * @param provider - The provider to set
     *
     * Accepted providers are of type {@link SupportedProviders}
     * @example
     * ```ts
     *  const web3Context = new web3ContextContext("http://localhost:8545");
     * web3Context.provider = "ws://localhost:8545";
     * console.log(web3Context.provider);
     * > WebSocketProvider {
     * _eventEmitter: EventEmitter {
     * _events: [Object: null prototype] {},
     * _eventsCount: 0,
     * ...
     * }
     * ```
     */
    set provider(provider: SupportedProviders<API> | string | undefined);
    /**
     * Will return the current provider. (The same as `provider`)
     *
     * @returns Returns the current provider
     * @example
     * ```ts
     * const web3Context = new Web3Context("http://localhost:8545");
     * console.log(web3Context.provider);
     * > HttpProvider {
     * 	clientUrl: 'http://localhost:8545',
     * 	httpProviderOptions: undefined
     *  }
     * ```
     */
    get currentProvider(): Web3BaseProvider<API> | undefined;
    /**
     * Will set the current provider. (The same as `provider`)
     *
     * @param provider - {@link SupportedProviders} The provider to set
     *
     * @example
     * ```ts
     *  const web3Context = new Web3Context("http://localhost:8545");
     * web3Context.currentProvider = "ws://localhost:8545";
     * console.log(web3Context.provider);
     * > WebSocketProvider {
     * _eventEmitter: EventEmitter {
     * _events: [Object: null prototype] {},
     * _eventsCount: 0,
     * ...
     * }
     * ```
     */
    set currentProvider(provider: SupportedProviders<API> | string | undefined);
    /**
     * Will return the givenProvider if available.
     *
     * When using web3.js in an Ethereum compatible browser, it will set with the current native provider by that browser. Will return the given provider by the (browser) environment, otherwise `undefined`.
     */
    get givenProvider(): SupportedProviders<never> | undefined;
    /**
     * Will set the provider.
     *
     * @param provider - {@link SupportedProviders} The provider to set
     * @returns Returns true if the provider was set
     */
    setProvider(provider?: SupportedProviders<API> | string): boolean;
    setRequestManagerMiddleware(requestManagerMiddleware: RequestManagerMiddleware<API>): void;
    /**
     * Will return the {@link Web3BatchRequest} constructor.
     */
    get BatchRequest(): new () => Web3BatchRequest;
    /**
     * This method allows extending the web3 modules.
     * Note: This method is only for backward compatibility, and It is recommended to use Web3 v4 Plugin feature for extending web3.js functionality if you are developing something new.
     */
    extend(extendObj: ExtensionObject): this;
}
/**
 * Extend this class when creating a plugin that either doesn't require {@link EthExecutionAPI},
 * or interacts with a RPC node that doesn't fully implement {@link EthExecutionAPI}.
 *
 * To add type support for RPC methods to the {@link Web3RequestManager},
 * define a {@link Web3APISpec} and pass it as a generic to Web3PluginBase like so:
 *
 * @example
 * ```ts
 * type CustomRpcApi = {
 *	custom_rpc_method: () => string;
 *	custom_rpc_method_with_parameters: (parameter1: string, parameter2: number) => string;
 * };
 *
 * class CustomPlugin extends Web3PluginBase<CustomRpcApi> {...}
 * ```
 */
export declare abstract class Web3PluginBase<API extends Web3APISpec = Web3APISpec> extends Web3Context<API> {
    abstract pluginNamespace: string;
    protected registerNewTransactionType<NewTxTypeClass extends typeof BaseTransaction<unknown>>(type: Numbers, txClass: NewTxTypeClass): void;
}
/**
 * Extend this class when creating a plugin that makes use of {@link EthExecutionAPI},
 * or depends on other Web3 packages (such as `web3-eth-contract`) that depend on {@link EthExecutionAPI}.
 *
 * To add type support for RPC methods to the {@link Web3RequestManager} (in addition to {@link EthExecutionAPI}),
 * define a {@link Web3APISpec} and pass it as a generic to Web3PluginBase like so:
 *
 * @example
 * ```ts
 * type CustomRpcApi = {
 *	custom_rpc_method: () => string;
 *	custom_rpc_method_with_parameters: (parameter1: string, parameter2: number) => string;
 * };
 *
 * class CustomPlugin extends Web3PluginBase<CustomRpcApi> {...}
 * ```
 */
export declare abstract class Web3EthPluginBase<API extends Web3APISpec = unknown> extends Web3PluginBase<API & EthExecutionAPI> {
}
export type TransactionBuilder<API extends Web3APISpec = unknown> = <ReturnType = Transaction>(options: {
    transaction: Transaction;
    web3Context: Web3Context<API>;
    privateKey?: HexString | Uint8Array;
    fillGasPrice?: boolean;
}) => Promise<ReturnType>;
