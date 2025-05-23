import { Web3Context, Web3ContextInitOptions, Web3SubscriptionConstructor } from 'web3-core';
import { Web3Eth, RegisteredSubscription } from 'web3-eth';
import { ENS } from 'web3-eth-ens';
import { Iban } from 'web3-eth-iban';
import { Personal } from 'web3-eth-personal';
import { Net } from 'web3-net';
import * as utils from 'web3-utils';
import { EthExecutionAPI, SupportedProviders } from 'web3-types';
import { Web3EthInterface } from './types.js';
export declare class Web3<CustomRegisteredSubscription extends {
    [key: string]: Web3SubscriptionConstructor<EthExecutionAPI>;
} = RegisteredSubscription> extends Web3Context<EthExecutionAPI, CustomRegisteredSubscription & RegisteredSubscription> {
    static version: string;
    static utils: typeof utils;
    static requestEIP6963Providers: () => Promise<import("./web3_eip6963.js").EIP6963ProviderResponse>;
    static onNewProviderDiscovered: (callback: (providerEvent: import("./web3_eip6963.js").EIP6963ProvidersMapUpdateEvent) => void) => void;
    static modules: {
        Web3Eth: typeof Web3Eth;
        Iban: typeof Iban;
        Net: typeof Net;
        ENS: typeof ENS;
        Personal: typeof Personal;
    };
    utils: typeof utils;
    eth: Web3EthInterface;
    constructor(providerOrContext?: string | SupportedProviders<EthExecutionAPI> | Web3ContextInitOptions<EthExecutionAPI, CustomRegisteredSubscription>);
}
export default Web3;
//# sourceMappingURL=web3.d.ts.map