import { JsonRpcApiPollingProvider } from "./provider-jsonrpc.js";
import type { JsonRpcError, JsonRpcPayload, JsonRpcResult, JsonRpcSigner } from "./provider-jsonrpc.js";
import type { Network, Networkish } from "./network.js";
/**
 *  The interface to an [[link-eip-1193]] provider, which is a standard
 *  used by most injected providers, which the [[BrowserProvider]] accepts
 *  and exposes the API of.
 */
export interface Eip1193Provider {
    /**
     *  See [[link-eip-1193]] for details on this method.
     */
    request(request: {
        method: string;
        params?: Array<any> | Record<string, any>;
    }): Promise<any>;
}
/**
 *  The possible additional events dispatched when using the ``"debug"``
 *  event on a [[BrowserProvider]].
 */
export type DebugEventBrowserProvider = {
    action: "sendEip1193Payload";
    payload: {
        method: string;
        params: Array<any>;
    };
} | {
    action: "receiveEip1193Result";
    result: any;
} | {
    action: "receiveEip1193Error";
    error: Error;
};
export type BrowserProviderOptions = {
    polling?: boolean;
    staticNetwork?: null | boolean | Network;
    cacheTimeout?: number;
    pollingInterval?: number;
};
/**
 *  A **BrowserProvider** is intended to wrap an injected provider which
 *  adheres to the [[link-eip-1193]] standard, which most (if not all)
 *  currently do.
 */
export declare class BrowserProvider extends JsonRpcApiPollingProvider {
    #private;
    /**
     *  Connnect to the %%ethereum%% provider, optionally forcing the
     *  %%network%%.
     */
    constructor(ethereum: Eip1193Provider, network?: Networkish, _options?: BrowserProviderOptions);
    send(method: string, params: Array<any> | Record<string, any>): Promise<any>;
    _send(payload: JsonRpcPayload | Array<JsonRpcPayload>): Promise<Array<JsonRpcResult | JsonRpcError>>;
    getRpcError(payload: JsonRpcPayload, error: JsonRpcError): Error;
    /**
     *  Resolves to ``true`` if the provider manages the %%address%%.
     */
    hasSigner(address: number | string): Promise<boolean>;
    getSigner(address?: number | string): Promise<JsonRpcSigner>;
}
//# sourceMappingURL=provider-browser.d.ts.map