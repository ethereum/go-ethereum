import { EIP1193Provider, LegacyRequestProvider, LegacySendAsyncProvider, LegacySendProvider, SupportedProviders, Web3APISpec, Web3BaseProvider, MetaMaskProvider } from 'web3-types';
export declare const isWeb3Provider: <API extends Web3APISpec>(provider: SupportedProviders<API>) => provider is Web3BaseProvider<API>;
export declare const isMetaMaskProvider: <API extends Web3APISpec>(provider: SupportedProviders<API>) => provider is MetaMaskProvider<API>;
export declare const isLegacyRequestProvider: <API extends Web3APISpec>(provider: SupportedProviders<API>) => provider is LegacyRequestProvider;
export declare const isEIP1193Provider: <API extends Web3APISpec>(provider: SupportedProviders<API>) => provider is EIP1193Provider<API>;
export declare const isLegacySendProvider: <API extends Web3APISpec>(provider: SupportedProviders<API>) => provider is LegacySendProvider;
export declare const isLegacySendAsyncProvider: <API extends Web3APISpec>(provider: SupportedProviders<API>) => provider is LegacySendAsyncProvider;
export declare const isSupportedProvider: <API extends Web3APISpec>(provider: SupportedProviders<API>) => provider is SupportedProviders<API>;
export declare const isSupportSubscriptions: <API extends Web3APISpec>(provider: SupportedProviders<API>) => boolean;
