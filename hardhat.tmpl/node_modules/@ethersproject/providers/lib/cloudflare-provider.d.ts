import { Network } from "@ethersproject/networks";
import { UrlJsonRpcProvider } from "./url-json-rpc-provider";
export declare class CloudflareProvider extends UrlJsonRpcProvider {
    static getApiKey(apiKey: any): any;
    static getUrl(network: Network, apiKey?: any): string;
    perform(method: string, params: any): Promise<any>;
}
//# sourceMappingURL=cloudflare-provider.d.ts.map