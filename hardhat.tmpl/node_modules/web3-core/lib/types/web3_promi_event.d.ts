import { Web3EventCallback, Web3EventEmitter, Web3EventKey, Web3EventMap } from './web3_event_emitter.js';
export type PromiseExecutor<T> = (resolve: (data: T) => void, reject: (reason: unknown) => void) => void;
export declare class Web3PromiEvent<ResolveType, EventMap extends Web3EventMap> extends Web3EventEmitter<EventMap> implements Promise<ResolveType> {
    private readonly _promise;
    constructor(executor: PromiseExecutor<ResolveType>);
    [Symbol.toStringTag]: 'Promise';
    then<TResult1 = ResolveType, TResult2 = never>(onfulfilled?: ((value: ResolveType) => TResult1 | PromiseLike<TResult1>) | undefined, onrejected?: ((reason: unknown) => TResult2 | PromiseLike<TResult2>) | undefined): Promise<TResult1 | TResult2>;
    catch<TResult = never>(onrejected?: ((reason: unknown) => TResult | PromiseLike<TResult>) | undefined): Promise<ResolveType | TResult>;
    finally(onfinally?: (() => void) | undefined): Promise<ResolveType>;
    on<K extends Web3EventKey<EventMap>>(eventName: K, fn: Web3EventCallback<EventMap[K]>): this;
    once<K extends Web3EventKey<EventMap>>(eventName: K, fn: Web3EventCallback<EventMap[K]>): this;
}
//# sourceMappingURL=web3_promi_event.d.ts.map