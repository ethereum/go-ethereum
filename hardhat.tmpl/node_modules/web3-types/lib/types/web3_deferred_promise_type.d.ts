export interface Web3DeferredPromiseInterface<T> extends Promise<T> {
    state: 'pending' | 'fulfilled' | 'rejected';
    resolve(value: T | PromiseLike<T>): void;
    reject(reason?: unknown): void;
    startTimer(): void;
}
//# sourceMappingURL=web3_deferred_promise_type.d.ts.map