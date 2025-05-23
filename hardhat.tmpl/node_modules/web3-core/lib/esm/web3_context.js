var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
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
import { isNullish } from 'web3-utils';
import { TransactionFactory } from 'web3-eth-accounts';
import { isSupportedProvider } from './utils.js';
import { Web3BatchRequest } from './web3_batch_request.js';
// eslint-disable-next-line import/no-cycle
import { Web3Config, Web3ConfigEvent } from './web3_config.js';
import { Web3RequestManager } from './web3_request_manager.js';
import { Web3SubscriptionManager } from './web3_subscription_manager.js';
export class Web3Context extends Web3Config {
    constructor(providerOrContext) {
        var _a;
        super();
        this.providers = Web3RequestManager.providers;
        // If "providerOrContext" is provided as "string" or an objects matching "SupportedProviders" interface
        if (isNullish(providerOrContext) ||
            (typeof providerOrContext === 'string' && providerOrContext.trim() !== '') ||
            isSupportedProvider(providerOrContext)) {
            this._requestManager = new Web3RequestManager(providerOrContext);
            this._subscriptionManager = new Web3SubscriptionManager(this._requestManager, {});
            return;
        }
        const { config, provider, requestManager, subscriptionManager, registeredSubscriptions, accountProvider, wallet, requestManagerMiddleware, } = providerOrContext;
        this.setConfig(config !== null && config !== void 0 ? config : {});
        this._requestManager =
            requestManager !== null && requestManager !== void 0 ? requestManager : new Web3RequestManager(provider, (_a = config === null || config === void 0 ? void 0 : config.enableExperimentalFeatures) === null || _a === void 0 ? void 0 : _a.useSubscriptionWhenCheckingBlockTimeout, requestManagerMiddleware);
        if (subscriptionManager) {
            this._subscriptionManager = subscriptionManager;
        }
        else {
            this._subscriptionManager = new Web3SubscriptionManager(this.requestManager, registeredSubscriptions !== null && registeredSubscriptions !== void 0 ? registeredSubscriptions : {});
        }
        if (accountProvider) {
            this._accountProvider = accountProvider;
        }
        if (wallet) {
            this._wallet = wallet;
        }
    }
    get requestManager() {
        return this._requestManager;
    }
    /**
     * Will return the current subscriptionManager ({@link Web3SubscriptionManager})
     */
    get subscriptionManager() {
        return this._subscriptionManager;
    }
    get wallet() {
        return this._wallet;
    }
    get accountProvider() {
        return this._accountProvider;
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    static fromContextObject(...args) {
        return new this(...args.reverse());
    }
    getContextObject() {
        var _a;
        return {
            config: this.config,
            provider: this.provider,
            requestManager: this.requestManager,
            subscriptionManager: this.subscriptionManager,
            registeredSubscriptions: (_a = this.subscriptionManager) === null || _a === void 0 ? void 0 : _a.registeredSubscriptions,
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
    use(ContextRef, ...args) {
        const newContextChild = new ContextRef(...[...args, this.getContextObject()]);
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
    link(parentContext) {
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
    registerPlugin(plugin) {
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
    get provider() {
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
    set provider(provider) {
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
    get currentProvider() {
        return this.requestManager.provider;
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
    set currentProvider(provider) {
        this.requestManager.setProvider(provider);
    }
    /**
     * Will return the givenProvider if available.
     *
     * When using web3.js in an Ethereum compatible browser, it will set with the current native provider by that browser. Will return the given provider by the (browser) environment, otherwise `undefined`.
     */
    // eslint-disable-next-line class-methods-use-this
    get givenProvider() {
        return Web3Context.givenProvider;
    }
    /**
     * Will set the provider.
     *
     * @param provider - {@link SupportedProviders} The provider to set
     * @returns Returns true if the provider was set
     */
    setProvider(provider) {
        this.provider = provider;
        return true;
    }
    setRequestManagerMiddleware(requestManagerMiddleware) {
        this.requestManager.setMiddleware(requestManagerMiddleware);
    }
    /**
     * Will return the {@link Web3BatchRequest} constructor.
     */
    get BatchRequest() {
        return Web3BatchRequest.bind(undefined, this._requestManager);
    }
    /**
     * This method allows extending the web3 modules.
     * Note: This method is only for backward compatibility, and It is recommended to use Web3 v4 Plugin feature for extending web3.js functionality if you are developing something new.
     */
    extend(extendObj) {
        var _a;
        // @ts-expect-error No index signature with a parameter of type 'string' was found on type 'Web3Context<API, RegisteredSubs>'
        if (extendObj.property && !this[extendObj.property])
            // @ts-expect-error No index signature with a parameter of type 'string' was found on type 'Web3Context<API, RegisteredSubs>'
            this[extendObj.property] = {};
        (_a = extendObj.methods) === null || _a === void 0 ? void 0 : _a.forEach(element => {
            const method = (...givenParams) => __awaiter(this, void 0, void 0, function* () {
                return this.requestManager.send({
                    method: element.call,
                    params: givenParams,
                });
            });
            if (extendObj.property)
                // @ts-expect-error No index signature with a parameter of type 'string' was found on type 'Web3Context<API, RegisteredSubs>'
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                this[extendObj.property][element.name] = method;
            // @ts-expect-error No index signature with a parameter of type 'string' was found on type 'Web3Context<API, RegisteredSubs>'
            else
                this[element.name] = method;
        });
        return this;
    }
}
Web3Context.providers = Web3RequestManager.providers;
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
export class Web3PluginBase extends Web3Context {
    // eslint-disable-next-line class-methods-use-this
    registerNewTransactionType(type, txClass) {
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
export class Web3EthPluginBase extends Web3PluginBase {
}
//# sourceMappingURL=web3_context.js.map