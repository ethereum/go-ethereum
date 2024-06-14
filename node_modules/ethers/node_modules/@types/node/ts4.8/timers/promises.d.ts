/**
 * The `timers/promises` API provides an alternative set of timer functions
 * that return `Promise` objects. The API is accessible via`require('timers/promises')`.
 *
 * ```js
 * import {
 *   setTimeout,
 *   setImmediate,
 *   setInterval,
 * } from 'timers/promises';
 * ```
 * @since v15.0.0
 */
declare module 'timers/promises' {
    import { TimerOptions } from 'node:timers';
    /**
     * ```js
     * import {
     *   setTimeout,
     * } from 'timers/promises';
     *
     * const res = await setTimeout(100, 'result');
     *
     * console.log(res);  // Prints 'result'
     * ```
     * @since v15.0.0
     * @param [delay=1] The number of milliseconds to wait before fulfilling the promise.
     * @param value A value with which the promise is fulfilled.
     */
    function setTimeout<T = void>(delay?: number, value?: T, options?: TimerOptions): Promise<T>;
    /**
     * ```js
     * import {
     *   setImmediate,
     * } from 'timers/promises';
     *
     * const res = await setImmediate('result');
     *
     * console.log(res);  // Prints 'result'
     * ```
     * @since v15.0.0
     * @param value A value with which the promise is fulfilled.
     */
    function setImmediate<T = void>(value?: T, options?: TimerOptions): Promise<T>;
    /**
     * Returns an async iterator that generates values in an interval of `delay` ms.
     *
     * ```js
     * import {
     *   setInterval,
     * } from 'timers/promises';
     *
     * const interval = 100;
     * for await (const startTime of setInterval(interval, Date.now())) {
     *   const now = Date.now();
     *   console.log(now);
     *   if ((now - startTime) > 1000)
     *     break;
     * }
     * console.log(Date.now());
     * ```
     * @since v15.9.0
     */
    function setInterval<T = void>(delay?: number, value?: T, options?: TimerOptions): AsyncIterable<T>;

    interface Scheduler {
        /**
         * ```js
         * import { scheduler } from 'node:timers/promises';
         *
         * await scheduler.wait(1000); // Wait one second before continuing
         * ```
         * An experimental API defined by the Scheduling APIs draft specification being developed as a standard Web Platform API.
         * Calling timersPromises.scheduler.wait(delay, options) is roughly equivalent to calling timersPromises.setTimeout(delay, undefined, options) except that the ref option is not supported.
         * @since v16.14.0
         * @experimental
         * @param [delay=1] The number of milliseconds to wait before fulfilling the promise.
         */
        wait: (delay?: number, options?: TimerOptions) => Promise<void>;
        /**
         * An experimental API defined by the Scheduling APIs draft specification being developed as a standard Web Platform API.
         * Calling timersPromises.scheduler.yield() is equivalent to calling timersPromises.setImmediate() with no arguments.
         * @since v16.14.0
         * @experimental
         */
        yield: () => Promise<void>;
    }

    const scheduler: Scheduler;
}
declare module 'node:timers/promises' {
    export * from 'timers/promises';
}
