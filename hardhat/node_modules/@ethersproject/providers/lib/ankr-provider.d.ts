import { Network } from "@ethersproject/networks";
import { UrlJsonRpcProvider } from "./url-json-rpc-provider";
import type { ConnectionInfo } from "@ethersproject/web";
export declare class AnkrProvider extends UrlJsonRpcProvider {
    readonly apiKey: string;
    isCommunityResource(): boolean;
    static getApiKey(apiKey: any): any;
    static getUrl(network: Network, apiKey: any): ConnectionInfo;
}
//# sourceMappingURL=ankr-provider.d.ts.map