/// <reference types="node" />
import { EventEmitter } from "events";
import { EthereumProvider, JsonRpcRequest, JsonRpcResponse, RequestArguments } from "../../../types";
export type ProviderFactory = () => Promise<EthereumProvider>;
export type Listener = (...args: any[]) => void;
/**
 * A class that delays the (async) creation of its internal provider until the first call
 * to a JSON RPC method via request/send/sendAsync or the init method is called.
 */
export declare class LazyInitializationProviderAdapter implements EthereumProvider {
    private _providerFactory;
    protected provider: EthereumProvider | undefined;
    private _emitter;
    private _initializingPromise;
    constructor(_providerFactory: ProviderFactory);
    /**
     * Gets the internal wrapped provider.
     * Using it directly is discouraged and should be done with care,
     * use the public methods from the class like `request` and all event emitter methods instead
     */
    get _wrapped(): EventEmitter;
    init(): Promise<EthereumProvider>;
    request(args: RequestArguments): Promise<unknown>;
    send(method: string, params?: any[]): Promise<any>;
    sendAsync(payload: JsonRpcRequest, callback: (error: any, response: JsonRpcResponse) => void): void;
    addListener(event: string | symbol, listener: EventListener): this;
    on(event: string | symbol, listener: EventListener): this;
    once(event: string | symbol, listener: Listener): this;
    prependListener(event: string | symbol, listener: Listener): this;
    prependOnceListener(event: string | symbol, listener: Listener): this;
    removeListener(event: string | symbol, listener: Listener): this;
    off(event: string | symbol, listener: Listener): this;
    removeAllListeners(event?: string | symbol | undefined): this;
    setMaxListeners(n: number): this;
    getMaxListeners(): number;
    listeners(event: string | symbol): Function[];
    rawListeners(event: string | symbol): Function[];
    emit(event: string | symbol, ...args: any[]): boolean;
    eventNames(): Array<string | symbol>;
    listenerCount(type: string | symbol): number;
    private _getEmitter;
    private _getOrInitProvider;
}
//# sourceMappingURL=lazy-initialization.d.ts.map