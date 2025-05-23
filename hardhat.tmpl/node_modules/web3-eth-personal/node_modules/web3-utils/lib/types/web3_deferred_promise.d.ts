import { Web3DeferredPromiseInterface } from 'web3-types';
/**
 * The class is a simple implementation of a deferred promise with optional timeout functionality,
 * which can be useful when dealing with asynchronous tasks.
 *
 */
export declare class Web3DeferredPromise<T> implements Promise<T>, Web3DeferredPromiseInterface<T> {
    [Symbol.toStringTag]: 'Promise';
    private readonly _promise;
    private _resolve;
    private _reject;
    private _state;
    private _timeoutId?;
    private readonly _timeoutInterval?;
    private readonly _timeoutMessage;
    /**
     *
     * @param timeout - (optional) The timeout in milliseconds.
     * @param eagerStart - (optional) If true, the timer starts as soon as the promise is created.
     * @param timeoutMessage - (optional) The message to include in the timeout erro that is thrown when the promise times out.
     */
    constructor({ timeout, eagerStart, timeoutMessage, }?: {
        timeout: number;
        eagerStart: boolean;
        timeoutMessage: string;
    });
    /**
     * Returns the current state of the promise.
     * @returns 'pending' | 'fulfilled' | 'rejected'
     */
    get state(): 'pending' | 'fulfilled' | 'rejected';
    /**
     *
     * @param onfulfilled - (optional) The callback to execute when the promise is fulfilled.
     * @param onrejected  - (optional) The callback to execute when the promise is rejected.
     * @returns
     */
    then<TResult1, TResult2>(onfulfilled?: (value: T) => TResult1 | PromiseLike<TResult1>, onrejected?: (reason: unknown) => TResult2 | PromiseLike<TResult2>): Promise<TResult1 | TResult2>;
    /**
     *
     * @param onrejected - (optional) The callback to execute when the promise is rejected.
     * @returns
     */
    catch<TResult>(onrejected?: (reason: any) => TResult | PromiseLike<TResult>): Promise<T | TResult>;
    /**
     *
     * @param onfinally - (optional) The callback to execute when the promise is settled (fulfilled or rejected).
     * @returns
     */
    finally(onfinally?: (() => void) | undefined): Promise<T>;
    /**
     * Resolves the current promise.
     * @param value - The value to resolve the promise with.
     */
    resolve(value: T | PromiseLike<T>): void;
    /**
     * Rejects the current promise.
     * @param reason - The reason to reject the promise with.
     */
    reject(reason?: unknown): void;
    /**
     * Starts the timeout timer for the promise.
     */
    startTimer(): void;
    private _checkTimeout;
    private _clearTimeout;
}
//# sourceMappingURL=web3_deferred_promise.d.ts.map