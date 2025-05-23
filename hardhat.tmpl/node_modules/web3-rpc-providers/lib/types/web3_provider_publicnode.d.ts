import { EthExecutionAPI, Web3APISpec } from 'web3-types';
import { HttpProviderOptions } from 'web3-providers-http';
import { Network, SocketOptions, Transport } from './types.js';
import { Web3ExternalProvider } from './web3_provider.js';
export declare class PublicNodeProvider<API extends Web3APISpec = EthExecutionAPI> extends Web3ExternalProvider<API> {
    constructor(network?: Network, transport?: Transport, host?: string, providerConfigOptions?: HttpProviderOptions | SocketOptions);
    static readonly networkHostMap: {
        [key: string]: string;
    };
    getRPCURL(network: Network, transport: Transport, _: string, _host: string): string;
}
//# sourceMappingURL=web3_provider_publicnode.d.ts.map