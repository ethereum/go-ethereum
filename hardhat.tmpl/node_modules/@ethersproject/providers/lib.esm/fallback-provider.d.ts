import { Provider } from "@ethersproject/abstract-provider";
import { Network } from "@ethersproject/networks";
import { BaseProvider } from "./base-provider";
export interface FallbackProviderConfig {
    provider: Provider;
    priority?: number;
    stallTimeout?: number;
    weight?: number;
}
export declare class FallbackProvider extends BaseProvider {
    readonly providerConfigs: ReadonlyArray<FallbackProviderConfig>;
    readonly quorum: number;
    _highestBlockNumber: number;
    constructor(providers: Array<Provider | FallbackProviderConfig>, quorum?: number);
    detectNetwork(): Promise<Network>;
    perform(method: string, params: {
        [name: string]: any;
    }): Promise<any>;
}
//# sourceMappingURL=fallback-provider.d.ts.map