export declare class Lock {
    private permits;
    private promiseResolverQueue;
    /**
     * Returns a promise used to wait for a permit to become available. This method should be awaited on.
     * @returns  A promise that gets resolved when execution is allowed to proceed.
     */
    acquire(): Promise<boolean>;
    /**
     * Increases the number of permits by one. If there are other functions waiting, one of them will
     * continue to execute in a future iteration of the event loop.
     */
    release(): void;
}
//# sourceMappingURL=lock.d.ts.map