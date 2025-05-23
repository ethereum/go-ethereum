import { Network, Networkish } from "@ethersproject/networks";
import { ConnectionInfo } from "@ethersproject/web";
import { CommunityResourcable } from "./formatter";
import { JsonRpcProvider, JsonRpcSigner } from "./json-rpc-provider";
export declare class StaticJsonRpcProvider extends JsonRpcProvider {
    detectNetwork(): Promise<Network>;
}
export declare abstract class UrlJsonRpcProvider extends StaticJsonRpcProvider implements CommunityResourcable {
    readonly apiKey: any;
    constructor(network?: Networkish, apiKey?: any);
    _startPending(): void;
    isCommunityResource(): boolean;
    getSigner(address?: string): JsonRpcSigner;
    listAccounts(): Promise<Array<string>>;
    static getApiKey(apiKey: any): any;
    static getUrl(network: Network, apiKey: any): string | ConnectionInfo;
}
//# sourceMappingURL=url-json-rpc-provider.d.ts.map