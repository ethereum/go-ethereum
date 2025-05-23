export type Timer = ReturnType<typeof setInterval>;
export type Timeout = ReturnType<typeof setTimeout>;
/**
 * An alternative to the node function `isPromise` that exists in `util/types` because it is not available on the browser.
 * @param object - to check if it is a `Promise`
 * @returns `true` if it is an `object` or a `function` that has a `then` function. And returns `false` otherwise.
 */
export declare function isPromise(object: unknown): boolean;
export type AsyncFunction<T, K = unknown> = (...args: K[]) => Promise<T>;
export declare function waitWithTimeout<T>(awaitable: Promise<T> | AsyncFunction<T>, timeout: number, error: Error): Promise<T>;
export declare function waitWithTimeout<T>(awaitable: Promise<T> | AsyncFunction<T>, timeout: number): Promise<T | undefined>;
/**
 * Repeatedly calls an async function with a given interval until the result of the function is defined (not undefined or null),
 * or until a timeout is reached. It returns promise and intervalId.
 * @param func - The function to call.
 * @param interval - The interval in milliseconds.
 */
export declare function pollTillDefinedAndReturnIntervalId<T>(func: AsyncFunction<T>, interval: number): [Promise<Exclude<T, undefined>>, Timer];
/**
 * Repeatedly calls an async function with a given interval until the result of the function is defined (not undefined or null),
 * or until a timeout is reached.
 * pollTillDefinedAndReturnIntervalId() function should be used instead of pollTillDefined if you need IntervalId in result.
 * This function will be deprecated in next major release so use pollTillDefinedAndReturnIntervalId().
 * @param func - The function to call.
 * @param interval - The interval in milliseconds.
 */
export declare function pollTillDefined<T>(func: AsyncFunction<T>, interval: number): Promise<Exclude<T, undefined>>;
/**
 * Enforce a timeout on a promise, so that it can be rejected if it takes too long to complete
 * @param timeout - The timeout to enforced in milliseconds.
 * @param error - The error to throw if the timeout is reached.
 * @returns A tuple of the timeout id and the promise that will be rejected if the timeout is reached.
 *
 * @example
 * ```ts
 * const [timerId, promise] = web3.utils.rejectIfTimeout(100, new Error('time out'));
 * ```
 */
export declare function rejectIfTimeout(timeout: number, error: Error): [Timer, Promise<never>];
/**
 * Sets an interval that repeatedly executes the given cond function with the specified interval between each call.
 * If the condition is met, the interval is cleared and a Promise that rejects with the returned value is returned.
 * @param cond - The function/condition to call.
 * @param interval - The interval in milliseconds.
 * @returns - an array with the interval ID and the Promise.
 */
export declare function rejectIfConditionAtInterval<T>(cond: AsyncFunction<T | undefined>, interval: number): [Timer, Promise<never>];
//# sourceMappingURL=promise_helpers.d.ts.map