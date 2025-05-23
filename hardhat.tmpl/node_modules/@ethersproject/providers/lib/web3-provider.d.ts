import { Networkish } from "@ethersproject/networks";
import { JsonRpcProvider } from "./json-rpc-provider";
export declare type ExternalProvider = {
    isMetaMask?: boolean;
    isStatus?: boolean;
    host?: string;
    path?: string;
    sendAsync?: (request: {
        method: string;
        params?: Array<any>;
    }, callback: (error: any, response: any) => void) => void;
    send?: (request: {
        method: string;
        params?: Array<any>;
    }, callback: (error: any, response: any) => void) => void;
    request?: (request: {
        method: string;
        params?: Array<any>;
    }) => Promise<any>;
};
export declare type JsonRpcFetchFunc = (method: string, params?: Array<any>) => Promise<any>;
export declare class Web3Provider extends JsonRpcProvider {
    readonly provider: ExternalProvider;
    readonly jsonRpcFetchFunc: JsonRpcFetchFunc;
    constructor(provider: ExternalProvider | JsonRpcFetchFunc, network?: Networkish);
    send(method: string, params: Array<any>): Promise<any>;
}
//# sourceMappingURL=web3-provider.d.ts.map