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
import { ExistingPluginNamespaceError } from 'web3-errors';
import {
	EthExecutionAPI,
	HexString,
	Numbers,
	SupportedProviders,
	Transaction,
	Web3AccountProvider,
	Web3APISpec,
	Web3BaseProvider,
	Web3BaseWallet,
	Web3BaseWalletAccount,
} from 'web3-types';
import { isNullish } from 'web3-utils';
import { BaseTransaction, TransactionFactory } from 'web3-eth-accounts';
import { isSupportedProvider } from './utils.js';
// eslint-disable-next-line import/no-cycle
import { ExtensionObject, RequestManagerMiddleware } from './types.js';
import { Web3BatchRequest } from './web3_batch_request.js';
// eslint-disable-next-line import/no-cycle
import { Web3Config, Web3ConfigEvent, Web3ConfigOptions } from './web3_config.js';
import { Web3RequestManager } from './web3_request_manager.js';
import { Web3SubscriptionConstructor } from './web3_subscriptions.js';
import { Web3SubscriptionManager } from './web3_subscription_manager.js';

// To avoid circular dependencies, we need to export type from here.
export type Web3ContextObject<
	API extends Web3APISpec = unknown,
	RegisteredSubs extends {
		[key: string]: Web3SubscriptionConstructor<API>;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
	} = any,
> = {
	config: Web3ConfigOptions;
	provider?: SupportedProviders<API> | string;
	requestManager: Web3RequestManager<API>;
	subscriptionManager?: Web3SubscriptionManager<API, RegisteredSubs> | undefined;
	registeredSubscriptions?: RegisteredSubs;
	providers: typeof Web3RequestManager.providers;
	accountProvider?: Web3AccountProvider<Web3BaseWalletAccount>;
	wallet?: Web3BaseWallet<Web3BaseWalletAccount>;
};

export type Web3ContextInitOptions<
	API extends Web3APISpec = unknown,
	RegisteredSubs extends {
		[key: string]: Web3SubscriptionConstructor<API>;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
	} = any,
> = {
	config?: Partial<Web3ConfigOptions>;
	provider?: SupportedProviders<API> | string;
	requestManager?: Web3RequestManager<API>;
	subscriptionManager?: Web3SubscriptionManager<API, RegisteredSubs> | undefined;
	registeredSubscriptions?: RegisteredSubs;
	accountProvider?: Web3AccountProvider<Web3BaseWalletAccount>;
	wallet?: Web3BaseWallet<Web3BaseWalletAccount>;
	requestManagerMiddleware?: RequestManagerMiddleware<API>;
};

// eslint-disable-next-line no-use-before-define
export type Web3ContextConstructor<T extends Web3Context, T2 extends unknown[]> = new (
	...args: [...extras: T2, context: Web3ContextObject]
) => T;

// To avoid circular dependencies, we need to export type from here.
export type Web3ContextFactory<
	// eslint-disable-next-line no-use-before-define
	T extends Web3Context,
	T2 extends unknown[],
> = Web3ContextConstructor<T, T2> & {
	fromContextObject(this: Web3ContextConstructor<T, T2>, contextObject: Web3ContextObject): T;
};

export class Web3Context<
	API extends Web3APISpec = unknown,
	RegisteredSubs extends {
		[key: string]: Web3SubscriptionConstructor<API>;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
	} = any,
> extends Web3Config {
	public static readonly providers = Web3RequestManager.providers;
	public static givenProvider?: SupportedProviders<never>;
	public readonly providers = Web3RequestManager.providers;
	protected _requestManager: Web3RequestManager<API>;
	protected _subscriptionManager: Web3SubscriptionManager<API, RegisteredSubs>;
	protected _accountProvider?: Web3AccountProvider<Web3BaseWalletAccount>;
	protected _wallet?: Web3BaseWallet<Web3BaseWalletAccount>;

	public constructor(
		providerOrContext?:
			| string
			| SupportedProviders<API>
			| Web3ContextInitOptions<API, RegisteredSubs>,
	) {
		super();

		// If "providerOrContext" is provided as "string" or an objects matching "SupportedProviders" interface
		if (
			isNullish(providerOrContext) ||
			(typeof providerOrContext === 'string' && providerOrContext.trim() !== '') ||
			isSupportedProvider(providerOrContext as SupportedProviders<API>)
		) {
			this._requestManager = new Web3RequestManager<API>(
				providerOrContext as undefined | string | SupportedProviders<API>,
			);
			this._subscriptionManager = new Web3SubscriptionManager(
				this._requestManager,
				{} as RegisteredSubs,
			);

			return;
		}

		const {
			config,
			provider,
			requestManager,
			subscriptionManager,
			registeredSubscriptions,
			accountProvider,
			wallet,
			requestManagerMiddleware,
		} = providerOrContext as Web3ContextInitOptions<API, RegisteredSubs>;

		this.setConfig(config ?? {});

		this._requestManager =
			requestManager ??
			new Web3RequestManager<API>(
				provider,
				config?.enableExperimentalFeatures?.useSubscriptionWhenCheckingBlockTimeout,
				requestManagerMiddleware,
			);

		if (subscriptionManager) {
			this._subscriptionManager = subscriptionManager;
		} else {
			this._subscriptionManager = new Web3SubscriptionManager(
				this.requestManager,
				registeredSubscriptions ?? ({} as RegisteredSubs),
			);
		}

		if (accountProvider) {
			this._accountProvider = accountProvider;
		}

		if (wallet) {
			this._wallet = wallet;
		}
	}

	public get requestManager() {
		return this._requestManager;
	}

	/**
	 * Will return the current subscriptionManager ({@link Web3SubscriptionManager})
	 */
	public get subscriptionManager() {
		return this._subscriptionManager;
	}

	public get wallet() {
		return this._wallet;
	}

	public get accountProvider() {
		return this._accountProvider;
	}

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	public static fromContextObject<T extends Web3Context, T3 extends unknown[]>(
		this: Web3ContextConstructor<T, T3>,
		...args: [Web3ContextObject, ...T3]
	) {
		return new this(...(args.reverse() as [...T3, Web3ContextObject]));
	}

	public getContextObject(): Web3ContextObject<API, RegisteredSubs> {
		return {
			config: this.config,
			provider: this.provider,
			requestManager: this.requestManager,
			subscriptionManager: this.subscriptionManager,
			registeredSubscriptions: this.subscriptionManager?.registeredSubscriptions,
			providers: this.providers,
			wallet: this.wallet,
			accountProvider: this.accountProvider,
		};
	}

	/**
	 * Use to create new object of any type extended by `Web3Context`
	 * and link it to current context. This can be used to initiate a global context object
	 * and then use it to create new objects of any type extended by `Web3Context`.
	 */
	public use<T extends Web3Context, T2 extends unknown[]>(
		ContextRef: Web3ContextConstructor<T, T2>,
		...args: [...T2]
	) {
		const newContextChild: T = new ContextRef(
			...([...args, this.getContextObject()] as unknown as [...T2, Web3ContextObject]),
		);

		this.on(Web3ConfigEvent.CONFIG_CHANGE, event => {
			// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
			newContextChild.setConfig({ [event.name]: event.newValue });
		});

		// @ts-expect-error No index signature with a parameter of type 'string' was found on type 'Web3Context<API, RegisteredSubs>'
		this[ContextRef.name] = newContextChild;

		return newContextChild;
	}

	/**
	 * Link current context to another context.
	 */
	public link<T extends Web3Context>(parentContext: T) {
		this.setConfig(parentContext.config);
		this._requestManager = parentContext.requestManager;
		this.provider = parentContext.provider;
		// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
		this._subscriptionManager = parentContext.subscriptionManager;
		this._wallet = parentContext.wallet;
		this._accountProvider = parentContext._accountProvider;

		parentContext.on(Web3ConfigEvent.CONFIG_CHANGE, event => {
			// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
			this.setConfig({ [event.name]: event.newValue });
		});
	}

	// eslint-disable-next-line no-use-before-define
	public registerPlugin(plugin: Web3PluginBase) {
		// @ts-expect-error No index signature with a parameter of type 'string' was found on type 'Web3Context<API, RegisteredSubs>'
		if (this[plugin.pluginNamespace] !== undefined)
			throw new ExistingPluginNamespaceError(plugin.pluginNamespace);

		const _pluginObject = {
			[plugin.pluginNamespace]: plugin,
		};
		_pluginObject[plugin.pluginNamespace].link(this);
		Object.assign(this, _pluginObject);
	}

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

	public get provider(): Web3BaseProvider<API> | undefined {
		return this.currentProvider;
	}

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

	public set provider(provider: SupportedProviders<API> | string | undefined) {
		this.requestManager.setProvider(provider);
	}

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
	public get currentProvider(): Web3BaseProvider<API> | undefined {
		return this.requestManager.provider as Web3BaseProvider<API>;
	}

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
	public set currentProvider(provider: SupportedProviders<API> | string | undefined) {
		this.requestManager.setProvider(provider);
	}

	/**
	 * Will return the givenProvider if available.
	 *
	 * When using web3.js in an Ethereum compatible browser, it will set with the current native provider by that browser. Will return the given provider by the (browser) environment, otherwise `undefined`.
	 */
	// eslint-disable-next-line class-methods-use-this
	public get givenProvider() {
		return Web3Context.givenProvider;
	}
	/**
	 * Will set the provider.
	 *
	 * @param provider - {@link SupportedProviders} The provider to set
	 * @returns Returns true if the provider was set
	 */
	public setProvider(provider?: SupportedProviders<API> | string): boolean {
		this.provider = provider;
		return true;
	}

	public setRequestManagerMiddleware(requestManagerMiddleware: RequestManagerMiddleware<API>) {
		this.requestManager.setMiddleware(requestManagerMiddleware);
	}

	/**
	 * Will return the {@link Web3BatchRequest} constructor.
	 */
	public get BatchRequest(): new () => Web3BatchRequest {
		return Web3BatchRequest.bind(
			undefined,
			this._requestManager as unknown as Web3RequestManager,
		);
	}

	/**
	 * This method allows extending the web3 modules.
	 * Note: This method is only for backward compatibility, and It is recommended to use Web3 v4 Plugin feature for extending web3.js functionality if you are developing something new.
	 */
	public extend(extendObj: ExtensionObject) {
		// @ts-expect-error No index signature with a parameter of type 'string' was found on type 'Web3Context<API, RegisteredSubs>'
		if (extendObj.property && !this[extendObj.property])
			// @ts-expect-error No index signature with a parameter of type 'string' was found on type 'Web3Context<API, RegisteredSubs>'
			this[extendObj.property] = {};

		extendObj.methods?.forEach(element => {
			const method = async (...givenParams: unknown[]) =>
				this.requestManager.send({
					method: element.call,
					params: givenParams,
				});

			if (extendObj.property)
				// @ts-expect-error No index signature with a parameter of type 'string' was found on type 'Web3Context<API, RegisteredSubs>'
				// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
				this[extendObj.property][element.name] = method;
			// @ts-expect-error No index signature with a parameter of type 'string' was found on type 'Web3Context<API, RegisteredSubs>'
			else this[element.name] = method;
		});
		return this;
	}
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
export abstract class Web3PluginBase<
	API extends Web3APISpec = Web3APISpec,
> extends Web3Context<API> {
	public abstract pluginNamespace: string;

	// eslint-disable-next-line class-methods-use-this
	protected registerNewTransactionType<NewTxTypeClass extends typeof BaseTransaction<unknown>>(
		type: Numbers,
		txClass: NewTxTypeClass,
	): void {
		TransactionFactory.registerTransactionType(type, txClass);
	}
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
export abstract class Web3EthPluginBase<API extends Web3APISpec = unknown> extends Web3PluginBase<
	API & EthExecutionAPI
> {}

// To avoid cycle dependency declare this type in this file
export type TransactionBuilder<API extends Web3APISpec = unknown> = <
	ReturnType = Transaction,
>(options: {
	transaction: Transaction;
	web3Context: Web3Context<API>;
	privateKey?: HexString | Uint8Array;
	fillGasPrice?: boolean;
}) => Promise<ReturnType>;
