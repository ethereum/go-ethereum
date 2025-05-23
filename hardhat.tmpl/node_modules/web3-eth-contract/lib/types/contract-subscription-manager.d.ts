import { Web3SubscriptionConstructor, Web3SubscriptionManager } from 'web3-core';
import { EthExecutionAPI, ContractAbi, DataFormat } from 'web3-types';
import { Contract } from './contract.js';
/**
 * Similar to `Web3SubscriptionManager` but has a reference to the Contract that uses
 */
export declare class ContractSubscriptionManager<API extends EthExecutionAPI, RegisteredSubs extends {
    [key: string]: Web3SubscriptionConstructor<API>;
} = any> extends Web3SubscriptionManager<API, RegisteredSubs> {
    readonly parentContract: Contract<ContractAbi>;
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
    constructor(self: Web3SubscriptionManager<API, RegisteredSubs>, parentContract: Contract<ContractAbi>);
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
}
//# sourceMappingURL=contract-subscription-manager.d.ts.map